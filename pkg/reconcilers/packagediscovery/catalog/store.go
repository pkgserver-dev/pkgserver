/*
Copyright 2024 Nokia.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package catalog

import (
	"context"
	"fmt"

	"github.com/henderiw/apiserver-store/pkg/storebackend"
	memstore "github.com/henderiw/apiserver-store/pkg/storebackend/memory"
	"github.com/henderiw/logger/log"
	"github.com/kform-dev/kform/pkg/recorder"
	"github.com/kform-dev/kform/pkg/recorder/diag"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	APIServiceName = "apiService"
)

type APIType int

const (
	APIType_Core APIType = iota
	APIType_Package
)

func (r APIType) String() string {
	switch r {
	case APIType_Core:
		return "core"
	case APIType_Package:
		return "package"
	}
	return "unknown"
}

type API struct {
	Type     APIType
	Kind     string
	Resource string
	PkgID    pkgid.PackageID
	Versions map[string][]string // key is version and value is pkgRev
}

type Store struct {
	recorder recorder.Recorder[diag.Diagnostic]
	// key: gk (ns:group, name:kind) -> Package(NS/Pkg), resource, kind, versions
	catalogAPIStore storebackend.Storer[*API]
	// key: gr (ns: group, name: resource) -> kind
	catalogGRMap storebackend.Storer[string]
	// key: pkg/pkgRevName -> per gvk dependencies
	catalogPkgStore storebackend.Storer[*Dependency]
}

func NewStore() *Store {
	return &Store{
		catalogAPIStore: memstore.NewStore[*API](),
		catalogGRMap:    memstore.NewStore[string](),
		catalogPkgStore: memstore.NewStore[*Dependency](),
	}
}

func (r *Store) InitializeRecorder(recorder recorder.Recorder[diag.Diagnostic]) {
	r.recorder = recorder
}

func (r *Store) DeletePkgRev(ctx context.Context, cr *pkgv1alpha1.PackageRevision) {
	log := log.FromContext(ctx)
	// delete the pkgRev from all the references
	r.catalogAPIStore.UpdateWithFn(ctx, func(ctx context.Context, key storebackend.Key, api *API) *API {
		// delete the pkgRevs from the api versions if this is a package API
		if api.Type == APIType_Package {
			for v, pkgRevs := range api.Versions {
				for i, pkgRevName := range pkgRevs {
					if pkgRevName == cr.Name {
						// delete the entry
						api.Versions[v] = append(pkgRevs[:i], pkgRevs[i+1:]...)
					}
				}
				// if no more pkgrevs use this version we can delete the api
				if len(api.Versions[v]) == 0 {
					delete(api.Versions, v)
				}
			}
		}
		// TODO do we need to delete the api altogether if there is no more ref
		return api
	})
	// dont delete the catalogGRMap since this should never change

	// delete the pkgrevision
	key := getKey(cr.Spec.PackageID.PkgString(), cr.Spec.PackageID.Revision)
	if err := r.catalogPkgStore.Delete(ctx, key); err != nil {
		log.Error("cannot delete pkgrev from pkgStore")
		r.recorder.Record(diag.DiagFromErr(err))
	}
}

func (r *Store) UpdatePkgRevAPI(ctx context.Context, cr *pkgv1alpha1.PackageRevision, outputs []*yaml.RNode) {
	log := log.FromContext(ctx)

	// we gather the apis that are referenced in this package revision
	// k8s api extensions are defined on the basis of CRD(s) or APIServices
	apiReferences := newAPIreferences(r.recorder)
	apiReferences.gatherAPIs(ctx, outputs)
	// dont proceed if there was an error found
	if r.recorder.Get().HasError() {

		return
	}

	// update the gr map and sort the entries before updating the apiStore
	for _, gvkr := range apiReferences.list() {
		gvkr := gvkr
		log.Debug("gvkr", "group", gvkr.group, "version", gvkr.version, "kind", gvkr.kind, "resource", gvkr.resource)
		// TODO check if there is a change
		// update the catalog Group, Resource -> Kind mapping
		if err := r.catalogGRMap.Update(ctx, getKey(gvkr.group, gvkr.resource), gvkr.kind); err != nil {
			r.recorder.Record(diag.DiagFromErr(err))
		}

		// update the apiStore
		r.catalogAPIStore.UpdateWithKeyFn(ctx, getKey(gvkr.group, gvkr.kind), func(ctx context.Context, api *API) *API {
			if api == nil {
				// no entry was found
				versions := map[string][]string{}
				versions[gvkr.version] = []string{cr.Name}
				return &API{
					Type:     APIType_Package,
					Kind:     gvkr.kind,
					Resource: gvkr.resource,
					PkgID:    cr.Spec.PackageID,
					Versions: versions, // hold all the version
				}
			}
			api.Type = APIType_Package
			api.PkgID = cr.Spec.PackageID
			// update an existing entry
			if len(api.Versions) == 0 { // initialize versions if none exist
				api.Versions = map[string][]string{}
			}
			if len(api.Versions[gvkr.version]) == 0 { // initialize versions entry if none exist
				api.Versions[gvkr.version] = []string{}
			}
			found := false
			for _, pkgRevName := range api.Versions[gvkr.version] {
				if pkgRevName == cr.Name {
					found = true
					continue
				}
			}
			if !found {
				api.Versions[gvkr.version] = append(api.Versions[gvkr.version], cr.Name)
			}
			return api
		})
	}
}

// UpdateAPIfromAPIResources updates the API store from watching the apiResources on the k8s API
func (r *Store) UpdateAPIfromAPIResources(ctx context.Context, apiResources []*metav1.APIResourceList) {
	log := log.FromContext(ctx)
	for _, apiResourceList := range apiResources {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			continue
		}
		for _, apiResource := range apiResourceList.APIResources {
			log.Debug("UpdateAPIfromAPIResources", "group", gv.Group, "version", gv.Version, "kind", apiResource.Kind, "resource", apiResource.Name)
			// update gr map
			r.catalogGRMap.Update(ctx, getKey(gv.Group, apiResource.Name), apiResource.Kind)

			// update api store
			key := getKey(gv.Group, apiResource.Kind)
			r.catalogAPIStore.UpdateWithKeyFn(ctx, key, func(ctx context.Context, api *API) *API {
				if api == nil {
					// no entry was found
					versions := map[string][]string{}
					versions[gv.Version] = []string{}
					return &API{
						Type:     APIType_Core,
						Kind:     apiResource.Kind,
						Resource: apiResource.Name,
						Versions: versions, // hold all the version
					}
				}
				// when an api is non nil, it means the entry exists and this is likely
				// filled with a api from a package, core resources don't change within
				// a deployment of k8s
				return api
			})
		}
	}
}

func (r *Store) UpdatePkgRevDependencies(ctx context.Context, cr *pkgv1alpha1.PackageRevision, packages, inputs, resources []*yaml.RNode) {
	log := log.FromContext(ctx)
	dr := newDependencyResolver(
		r.recorder,
		r.catalogAPIStore,
		r.catalogGRMap,
		cr.Spec.PackageID,
	)
	dep := dr.resolve(ctx, packages, inputs, resources)
	if err := r.catalogPkgStore.Update(ctx, getKey(cr.Spec.PackageID.PkgString(), cr.Spec.PackageID.Revision), dep); err != nil {
		log.Error("cannot update pkg dependencies")
	}
}

func (r *Store) GetPkgRevDependencies(ctx context.Context, cr *pkgv1alpha1.PackageRevision) (bool, []pkgid.Upstream) {
	log := log.FromContext(ctx)
	// the catalog should be fetched using the upstream identifier in the package

	r.catalogPkgStore.List(ctx, func(ctx context.Context, key storebackend.Key, dep *Dependency) {
		log.Debug("pkgrevs", "key ns", key.Namespace, "key name", key.Name, "resolution", dep.HasResolutionError(), "deps", dep.ListPkgDependencies())
	})
	dep, err := r.catalogPkgStore.Get(ctx, getKey(cr.Spec.Upstream.PkgString(), cr.Spec.Upstream.Revision))
	if err != nil {
		log.Error("failed to fetch catalog package info", "pkgId", cr.Spec.Upstream.PkgString(), "revision", cr.Spec.Upstream.Revision, "error", err)
		return false, nil
	}
	log.Debug("get pkgRev dependencies", "resolution error", dep.HasResolutionError(), "dependencies", dep.ListPkgDependencies())
	//dep.PrintPkgDependencies()
	//dep.PrintResolutionErrors()
	return !dep.HasResolutionError(), dep.ListPkgDependencies()
}

func (r *Store) Print(ctx context.Context) {
	/*
		fmt.Println("****** API resources ******")
		r.catalogAPIStore.List(ctx, func(ctx context.Context, key storebackend.Key, api *API) {
			fmt.Printf("type: %s, group: %s, kind: %s, resource: %s, versions: %v\n", api.Type.String(), key.Namespace, key.Name, api.Resource, api.Versions)
		})
		fmt.Println("****************************")
		fmt.Println("******** GR MAPPING ********")
		r.catalogGRMap.List(ctx, func(ctx context.Context, key storebackend.Key, kind string) {
			fmt.Println(key.Namespace, key.Name, kind)
		})
		fmt.Println("****************************")
	*/
	fmt.Println("**** Package Catalog ****")
	r.catalogPkgStore.List(ctx, func(ctx context.Context, key storebackend.Key, dep *Dependency) {
		fmt.Println("package", key.Namespace, "revision", key.Name)
		dep.PrintResolutionErrors()
		dep.PrintCoreDependencies()
		dep.PrintPkgDependencies()
	})
	fmt.Println("****************************")
}

/*
func (r *apiDiscoverer) PrintPkgResources(ctx context.Context) {

}
*/

/*
func getApiVersionKind(krmResource any) (string, string) {
	apiVersion := ""
	kind := ""
	switch x := krmResource.(type) {
	case map[string]any:
		for k, v := range x {
			//fmt.Println("output filename:", k, v)
			if k == "apiVersion" {
				if reflect.TypeOf(v).Kind() == reflect.String {
					apiVersion = v.(string)
				}
			}
			if k == "kind" {
				if reflect.TypeOf(v).Kind() == reflect.String {
					kind = v.(string)
				}
			}
		}
	}
	return apiVersion, kind
}
*/
