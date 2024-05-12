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
	"reflect"
	"strings"

	"github.com/henderiw/apiserver-store/pkg/storebackend"
	"github.com/henderiw/logger/log"
	"github.com/kform-dev/kform/pkg/recorder"
	"github.com/kform-dev/kform/pkg/recorder/diag"
	claimv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/claim/v1alpha1"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"golang.org/x/mod/semver"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

type dependencyResolver struct {
	recorder recorder.Recorder[diag.Diagnostic]
	// key: gk (ns:group, name:kind) -> Package(NS/Pkg), resource, kind, versions
	catalogAPIStore storebackend.Storer[*API]
	// key: gr (ns: group, name: resource) -> kind
	catalogGRMap storebackend.Storer[string]
	// the depdendency we are resolving
	dependency *Dependency
}

func getKey(ns string, name string) storebackend.Key {
	return storebackend.KeyFromNSN(
		types.NamespacedName{
			Namespace: ns,
			Name:      name,
		},
	)
}

func newDependencyResolver(recorder recorder.Recorder[diag.Diagnostic], catalogAPIStore storebackend.Storer[*API], catalogGRMap storebackend.Storer[string], pkgID pkgid.PackageID) *dependencyResolver {
	return &dependencyResolver{
		recorder:        recorder,
		catalogAPIStore: catalogAPIStore,
		catalogGRMap:    catalogGRMap,
		dependency:      NewDependency(pkgID),
	}
}

// resolve resolves the dependencies on the resources it gets provided
// regular resources
// -> get gvk
// -> for rbac rules: add itself as a regular dependency; for each rule treat it as a regular resource
// -> for pvar: lookup upstream and add this as a package dependency
// -> for reguler resources: lookup api based on gvk and determine if this is a core or pkg dependency
// package resources
// -> create an upstream package dependency
func (r *dependencyResolver) resolve(ctx context.Context, packages, inputs, resources []any) *Dependency {
	log := log.FromContext(ctx)
	log.Debug("resolve")
	// input resource can identify a dependency via the claimv1alpha1.PackageDependency
	// if so we capture this
	for _, krmResource := range inputs {
		//get apiVersion and kind from krmResource
		apiVersion, kind := getApiVersionKind(krmResource)
		gv, err := schema.ParseGroupVersion(apiVersion)
		if err != nil {
			r.recorder.Record(diag.DiagFromErr(err))
			continue
		}

		switch {
		case apiVersion == claimv1alpha1.SchemeGroupVersion.String() && kind == reflect.TypeOf(claimv1alpha1.PackageDependency{}).Name():
			// get pvar information
			if err := r.gatherDependencyfromPDEP(ctx, gv.WithKind(kind), krmResource); err != nil {
				r.dependency.AddGVKResolutionError(schema.FromAPIVersionAndKind(apiVersion, kind), err)
				continue
			}
		default:
		}
	}

	for _, krmResource := range resources {
		//get apiVersion and kind from krmResource
		apiVersion, kind := getApiVersionKind(krmResource)
		gv, err := schema.ParseGroupVersion(apiVersion)
		if err != nil {
			r.recorder.Record(diag.DiagFromErr(err))
			continue
		}
		//log.Info("resolve", "gvk", gv.WithKind(kind).String())
		switch {
		case apiVersion == rbacv1.SchemeGroupVersion.String() && kind == reflect.TypeOf(rbacv1.ClusterRole{}).Name():
			// get rbac information
			if err := r.gatherDependencyfromClusterRole(ctx, gv.WithKind(kind), krmResource); err != nil {
				r.dependency.AddGVKResolutionError(schema.FromAPIVersionAndKind(apiVersion, kind), err)
				continue
			}
		case apiVersion == rbacv1.SchemeGroupVersion.String() && kind == reflect.TypeOf(rbacv1.Role{}).Name():
			// get rbac information
			if err := r.gatherDependencyfromRole(ctx, gv.WithKind(kind), krmResource); err != nil {
				r.dependency.AddGVKResolutionError(schema.FromAPIVersionAndKind(apiVersion, kind), err)
				continue
			}
		case apiVersion == configv1alpha1.SchemeGroupVersion.String() && kind == reflect.TypeOf(configv1alpha1.PackageVariant{}).Name():
			// get pvar information
			if err := r.gatherDependencyfromPVAR(ctx, gv.WithKind(kind), krmResource); err != nil {
				r.dependency.AddGVKResolutionError(schema.FromAPIVersionAndKind(apiVersion, kind), err)
				continue
			}
		default:
			if err := r.gatherDependencyfromResource(ctx, gv.WithKind(kind)); err != nil {
				r.dependency.AddGVKResolutionError(schema.FromAPIVersionAndKind(apiVersion, kind), err)
				continue
			}
		}
	}
	// TODO update this to ensure we get the correct parameters
	// This is the package dependency due to a mixin.
	for _, krmResource := range packages {
		apiVersion, kind := getApiVersionKind(krmResource)
		gv, err := schema.ParseGroupVersion(apiVersion)
		if err != nil {
			r.recorder.Record(diag.DiagFromErr(err))
			continue
		}

		r.dependency.AddPkgDependency(schema.GroupVersionKind{
			Group:   gv.Group,
			Version: gv.Version,
			Kind:    kind,
		}, pkgid.Upstream{Repository: "tbd", Revision: "*", Realm: "", Package: "tbd"})

	}
	r.dependency.AddResolutionError(r.recorder.Get().Error())
	return r.dependency
}

func (r *dependencyResolver) gatherDependencyfromClusterRole(ctx context.Context, gvk schema.GroupVersionKind, krmResource any) error {
	log := log.FromContext(ctx)
	b, err := yaml.Marshal(krmResource)
	if err != nil {
		log.Error("cannot marshal crd")
		return err
	}
	clusterRole := &rbacv1.ClusterRole{}
	if err := yaml.Unmarshal(b, clusterRole); err != nil {
		log.Error("cannot unmarshal crd")
		return err
	}
	// first resolve the rbac gvk
	api, err := r.catalogAPIStore.Get(ctx, getKey(gvk.Group, gvk.Kind))
	if err != nil {
		return err
	}
	if _, ok := api.Versions[gvk.Version]; !ok {
		return fmt.Errorf("cannot resolve version for gvk: %s", gvk.String())
	}
	// add the dependency for the rbac clusterrole resource
	//api.PkgID = *api.PkgID.DeepCopy()
	r.addDependency(gvk, api)
	// add the policy rules dependencies
	return r.addPolicyRuleDependencies(ctx, gvk, clusterRole.Rules)
}

func (r *dependencyResolver) gatherDependencyfromRole(ctx context.Context, gvk schema.GroupVersionKind, krmResource any) error {
	log := log.FromContext(ctx)
	b, err := yaml.Marshal(krmResource)
	if err != nil {
		log.Error("cannot marshal crd")
		return err
	}
	role := &rbacv1.Role{}
	if err := yaml.Unmarshal(b, role); err != nil {
		log.Error("cannot unmarshal crd")
		return err
	}
	// first resolve the rbac gvk
	api, err := r.catalogAPIStore.Get(ctx, getKey(gvk.Group, gvk.Kind))
	if err != nil {
		return err
	}
	if _, ok := api.Versions[gvk.Version]; !ok {
		return fmt.Errorf("cannot resolve version for gvk: %s", gvk.String())
	}
	//api.PkgID = *api.PkgID.DeepCopy()
	r.addDependency(gvk, api)

	return r.addPolicyRuleDependencies(ctx, gvk, role.Rules)
}

func (r *dependencyResolver) addPolicyRuleDependencies(ctx context.Context, gvk schema.GroupVersionKind, rules []rbacv1.PolicyRule) error {
	// add the policy rules dependencies
	for _, rule := range rules {
		for _, group := range rule.APIGroups {
			for _, resource := range rule.Resources {
				parts := strings.Split(resource, "/")
				resource := parts[0]
				// wildcard resources is dangerous as it leads to unpredictable package dependencies
				if resource == "*" {
					continue
				}
				kind, err := r.catalogGRMap.Get(ctx, getKey(group, resource))
				if err != nil {
					// check if this is an api service
					kind, err = r.catalogGRMap.Get(ctx, getKey(group, APIServiceName))
					if err != nil {
						r.dependency.AddGVKResolutionWarning(schema.GroupVersionKind{Group: group}, fmt.Errorf("policy rule resource %s not found for gvk: %s", resource, gvk.String()))
						continue
					}
				}
				api, err := r.catalogAPIStore.Get(ctx, getKey(group, kind))
				if err != nil {
					r.dependency.AddGVKResolutionWarning(schema.GroupVersionKind{Group: group, Kind: kind}, fmt.Errorf("policy rule kind %s not found for gvk: %s", kind, gvk.String()))
					continue
				}
				gvk := schema.GroupVersionKind{
					Group:   group,
					Version: "*",
					Kind:    kind,
				}
				//api.PkgID = *api.PkgID.DeepCopy()
				if api.PkgID.PkgString() != r.dependency.PkgString() {
					r.addDependency(gvk, api)
				}
			}
		}
	}
	return nil
}

func (r *dependencyResolver) gatherDependencyfromPVAR(ctx context.Context, gvk schema.GroupVersionKind, krmResource any) error {
	log := log.FromContext(ctx)
	b, err := yaml.Marshal(krmResource)
	if err != nil {
		log.Error("cannot marshal packageVariant")
		return err
	}
	log.Debug("gatherDependencyfromPVAR", "string", string(b))
	pvar := &configv1alpha1.PackageVariant{}
	if err := yaml.Unmarshal(b, pvar); err != nil {
		log.Error("cannot unmarshal packageVariant")
		return err
	}
	log.Debug("gatherDependencyfromPVAR", "upstream", pvar.Spec)
	upstream := *pvar.Spec.Upstream.DeepCopy()
	if !semver.IsValid(pvar.Spec.Upstream.Revision) {
		upstream.Revision = "*"
	}
	r.dependency.AddPkgDependency(gvk, upstream)
	return nil
}

func (r *dependencyResolver) gatherDependencyfromPDEP(ctx context.Context, gvk schema.GroupVersionKind, krmResource any) error {
	log := log.FromContext(ctx)
	b, err := yaml.Marshal(krmResource)
	if err != nil {
		log.Error("cannot marshal packageDependency")
		return err
	}
	log.Debug("gatherDependencyfromPDEP", "string", string(b))
	pdep := &claimv1alpha1.PackageDependency{}
	if err := yaml.Unmarshal(b, pdep); err != nil {
		log.Error("cannot unmarshal packageDependency")
		return err
	}
	log.Debug("gatherDependencyfromPDEP", "upstream", pdep.Spec)
	upstream := *pdep.Spec.Upstream.DeepCopy()
	if !semver.IsValid(pdep.Spec.Upstream.Revision) {
		upstream.Revision = "*"
	}
	r.dependency.AddPkgDependency(gvk, upstream)
	return nil
}

func (r *dependencyResolver) gatherDependencyfromResource(ctx context.Context, gvk schema.GroupVersionKind) error {
	log := log.FromContext(ctx)

	r.catalogAPIStore.List(ctx, func(ctx context.Context, key storebackend.Key, api *API) {
		log.Debug("gatherDependencyfromResource list key", "namespace", key.Namespace, "name", key.Name)
	})

	api, err := r.catalogAPIStore.Get(ctx, getKey(gvk.Group, gvk.Kind))
	if err != nil {
		log.Error("gatherDependencyfromResource failed", "group", gvk.Group, "kind", gvk.Kind, "err", err.Error())
		return fmt.Errorf("cannot find api reference: group: %s, kind: %s", gvk.Group, gvk.Kind)
	}
	if _, ok := api.Versions[gvk.Version]; !ok {
		return fmt.Errorf("cannot resolve version for gvk: %s", gvk.String())
	}
	//api.PkgID = *api.PkgID.DeepCopy()
	r.addDependency(gvk, api)
	return nil
}

func (r *dependencyResolver) addDependency(gvk schema.GroupVersionKind, api *API) {
	//fmt.Println("addDependency", gvk.String(), api.Type.String(), api)
	if api.Type == APIType_Core {
		r.dependency.AddCoreDependency(gvk)
		return
	}

	upstream := pkgid.Upstream{
		Repository: api.PkgID.Repository,
		Realm:      api.PkgID.Realm,
		Package:    api.PkgID.Package,
		Revision:   "*",
	}
	r.dependency.AddPkgDependency(gvk, upstream)
}

func (r *dependencyResolver) ListCoreDependencies() []schema.GroupVersionKind {
	return r.dependency.ListCoreDependencies()

}

func (r *dependencyResolver) ListGVKPkgDependencies() map[schema.GroupVersionKind][]pkgid.Upstream {
	return r.dependency.ListGVKPkgDependencies()
}
