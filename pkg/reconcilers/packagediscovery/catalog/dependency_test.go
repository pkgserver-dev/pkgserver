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
	"os"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	memstore "github.com/henderiw/apiserver-store/pkg/storebackend/memory"
	"github.com/kform-dev/kform/pkg/recorder"
	"github.com/kform-dev/kform/pkg/recorder/diag"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"sigs.k8s.io/yaml"
)

func getResources(paths []string) ([]any, error) {
	resources := []any{}
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		ko, err := fn.ParseKubeObject(b)
		if err != nil {
			return nil, err
		}
		if ko == nil {
			return nil, fmt.Errorf("cannot create a block without a kubeobject")
		}
		var v map[string]any
		if err := yaml.Unmarshal([]byte(ko.String()), &v); err != nil {
			return nil, fmt.Errorf("cannot unmarshal the kubeobject, err: %s", err.Error())
		}
		resources = append(resources, v)
	}
	return resources, nil
}

func TestDependencyResolve(t *testing.T) {
	cases := map[string]struct {
		paths            []string
		expectedGroup    string
		expectedVersion  string
		expectedKind     string
		expectedResource string
	}{
		"Normal": {
			paths: []string{
				"test_data/apiservice.yaml",
				"test_data/config.pkg.pkgserver.dev_packagevariants.yaml",
				"test_data/deployment.yaml",
				"test_data/rbac-cluster-role-controller-permissions.yaml",
			},
			expectedGroup:    "pkg.pkgserver.dev",
			expectedVersion:  "v1alpha1",
			expectedKind:     APIServiceName,
			expectedResource: APIServiceName,
		},
	}

	for name, tc := range cases {
		ctx := context.Background()
		resources, err := getResources(tc.paths)
		if err != nil {
			t.Errorf("cannot get resources, unexpected error\n%s", err.Error())
		}

		pkgID := pkgid.PackageID{
			Target:     pkgid.PkgTarget_Catalog,
			Repository: "dummy",
			Realm:      "ns1",
			Package:    "pkg1",
			Revision:   "rev1",
			Workspace:  "ws1",
		}
		catalogAPIStore := memstore.NewStore[*API]()
		catalogAPIStore.Create(ctx, getKey("rbac.authorization.k8s.io", "ClusterRole"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("", "Namespace"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("", "Secret"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("", "Event"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("config.pkg.pkgserver.dev", "Repository"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1alpha1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("config.pkg.pkgserver.dev", "PackageVariant"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1alpha1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("admissionregistration.k8s.io", "MutatingWebhookConfiguration"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("admissionregistration.k8s.io", "ValidatingWebhookConfiguration"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("flowcontrol.apiserver.k8s.io", "FlowSchema"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1alpha1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("flowcontrol.apiserver.k8s.io", "PriorityLevelConfiguration"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1alpha1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("pkg.pkgserver.dev", "PackageRevision"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1alpha1": {},
			}})
		catalogAPIStore.Create(ctx, getKey("pkg.pkgserver.dev", "PackageRevisionResource"), &API{
			Type: APIType_Core,
			Versions: map[string][]string{
				"v1alpha1": {},
			}})
		catalogGRMap := memstore.NewStore[string]()
		catalogGRMap.Create(ctx, getKey("", "namespaces"), "Namespace")
		catalogGRMap.Create(ctx, getKey("", "secrets"), "Secret")
		catalogGRMap.Create(ctx, getKey("", "events"), "Event")
		catalogGRMap.Create(ctx, getKey("config.pkg.pkgserver.dev", "repositories"), "Repository")
		catalogGRMap.Create(ctx, getKey("config.pkg.pkgserver.dev", "packagevariants"), "PackageVariant")
		catalogGRMap.Create(ctx, getKey("pkg.pkgserver.dev", "packagerevisions"), "PackageRevision")
		catalogGRMap.Create(ctx, getKey("pkg.pkgserver.dev", "packagerevisionresources"), "PackageRevisionResource")
		catalogGRMap.Create(ctx, getKey("admissionregistration.k8s.io", "mutatingwebhookconfigurations"), "MutatingWebhookConfiguration")
		catalogGRMap.Create(ctx, getKey("admissionregistration.k8s.io", "validatingwebhookconfigurations"), "ValidatingWebhookConfiguration")
		catalogGRMap.Create(ctx, getKey("flowcontrol.apiserver.k8s.io", "flowschemas"), "FlowSchema")
		catalogGRMap.Create(ctx, getKey("flowcontrol.apiserver.k8s.io", "prioritylevelconfigurations"), "PriorityLevelConfiguration")
		//catalogPkgStore: memstore.NewStore[*Dependency](),
		pkgRevRecorder := recorder.New[diag.Diagnostic]()
		d := newDependencyResolver(pkgRevRecorder, catalogAPIStore, catalogGRMap, pkgID)
		_ = d.resolve(ctx, []any{}, []any{}, resources)

		if pkgRevRecorder.Get().Error() != nil {
			t.Errorf("%s unexpected error\n%s", name, err.Error())
		}

		for _, gvk := range d.ListCoreDependencies() {
			fmt.Println("core dependency", gvk.String())
		}
		for gvk, x := range d.ListGVKPkgDependencies() {
			fmt.Println("pkg  dependency", gvk.String(), x)
		}

	}
}
