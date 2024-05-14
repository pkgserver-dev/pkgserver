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
	"reflect"

	"github.com/henderiw/logger/log"
	"github.com/kform-dev/kform/pkg/recorder"
	"github.com/kform-dev/kform/pkg/recorder/diag"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	apiv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	//"sigs.k8s.io/yaml"
)

// per resource
//

// Namespace contains apiVersion
// Name contains kind/resource
type gvkr struct {
	group    string
	version  string
	kind     string
	resource string
}
type apiReferences struct {
	recorder recorder.Recorder[diag.Diagnostic]
	refs     sets.Set[gvkr]
}

func newAPIreferences(recorder recorder.Recorder[diag.Diagnostic]) *apiReferences {
	return &apiReferences{
		recorder: recorder,
		refs:     sets.Set[gvkr]{},
	}
}

func (r apiReferences) gatherAPIs(ctx context.Context, outputs []*yaml.RNode) {
	for _, rn := range outputs {
		//get apiVersion and kind from krmResource
		/*apiVersion, kind := getApiVersionKind(krmResource)*/
		//schema.ParseGroupVersion(apiVersion)
		switch {
		case rn.GetApiVersion() == extv1.SchemeGroupVersion.String() && rn.GetKind() == reflect.TypeOf(extv1.CustomResourceDefinition{}).Name():
			// get crd information
			if err := r.gatherAPIfromCRD(ctx, rn); err != nil {
				// we don't err since it allows us to record all errors
				// error is due to marshaling -> record as error
				r.recorder.Record(diag.DiagFromErr(err))
			}
		case rn.GetApiVersion() == apiv1.SchemeGroupVersion.String() && rn.GetKind() == reflect.TypeOf(apiv1.APIService{}).Name():
			// get apiService information
			if err := r.gatherAPIfromAPIService(ctx, rn); err != nil {
				// we don't err since it allows us to record all errors
				// error is due to marshaling -> record as error
				r.recorder.Record(diag.DiagFromErr(err))
			}
		default:
			// do nothing
		}
	}
}

func (r *apiReferences) gatherAPIfromCRD(ctx context.Context, rn *yaml.RNode) error {
	log := log.FromContext(ctx)
	/*
		b, err := yaml.Marshal(krmResource)
		if err != nil {
			log.Error("cannot marshal crd")
			return err
		}
	*/
	yamlString, err := rn.String()
	if err != nil {
		log.Error("cannot marshal crd")
		return err
	}

	crd := &extv1.CustomResourceDefinition{}
	if err := yaml.Unmarshal([]byte(yamlString), crd); err != nil {
		log.Error("cannot unmarshal crd")
		return err
	}
	for _, version := range crd.Spec.Versions {
		r.refs.Insert(gvkr{
			group:    crd.Spec.Group,
			version:  version.Name,
			kind:     crd.Spec.Names.Kind,
			resource: crd.Spec.Names.Plural,
		})
	}
	// insert a wildcard for rbac resolution -> too dangerous as it leads to unpreditable results
	/*
		r.refs.Insert(gvkr{
			group:    crd.Spec.Group,
			version:  "*",
			kind:     "*",
			resource: "*",s
		})
	*/
	return nil
}

func (r *apiReferences) gatherAPIfromAPIService(ctx context.Context, rn *yaml.RNode) error {
	log := log.FromContext(ctx)
	/*
	b, err := yaml.Marshal(krmResource)
	if err != nil {
		log.Error("cannot marshal apiService")
		return err
	}
	*/
	yamlString, err := rn.String()
	if err != nil {
		log.Error("cannot marshal crd")
		return err
	}
	apiService := &apiv1.APIService{}
	if err := yaml.Unmarshal([]byte(yamlString), apiService); err != nil {
		log.Error("cannot unmarshal apiService")
		return err
	}
	r.refs.Insert(gvkr{
		group:    apiService.Spec.Group,
		version:  apiService.Spec.Version,
		kind:     APIServiceName,
		resource: APIServiceName,
	})
	// insert a wildcard for rbac resolution -> too dangerous as it leads to unpreditable results
	/*
		r.refs.Insert(gvkr{
			group:    apiService.Spec.Group,
			version:  "*",
			kind:     "*",
			resource: "*",
		})
	*/
	return nil
}

func (r *apiReferences) list() []gvkr {
	return r.refs.UnsortedList()
}
