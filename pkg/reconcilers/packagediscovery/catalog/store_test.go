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

	"github.com/google/go-cmp/cmp"
	"github.com/henderiw/apiserver-store/pkg/storebackend"
	"github.com/kform-dev/kform/pkg/recorder"
	"github.com/kform-dev/kform/pkg/recorder/diag"
	kformtypes "github.com/kform-dev/kform/pkg/syntax/types"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	//"sigs.k8s.io/yaml"
)

func getOutput(path string) ([]*yaml.RNode, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rn, err := yaml.Parse(string(b))
	if err != nil {
		return nil, err
	}
	/*
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
	*/
	return []*yaml.RNode{rn}, nil
}

func TestCatalogAPIStoreUpdateAPIsFromPkgRev(t *testing.T) {

	pkgName := "pkg1"
	pkgID := pkgid.PackageID{
		Target:     pkgid.PkgTarget_Catalog,
		Repository: "repo",
		Realm:      "realm",
		Package:    pkgName,
		Workspace:  "ws1",
	}
	pkgRevName := pkgID.PkgRevString()
	cases := map[string]struct {
		path             string
		expectedGroup    string
		expectedVersion  string
		expectedKind     string
		expectedResource string
	}{
		"apiService": {
			path:             "test_data/apiservice.yaml",
			expectedGroup:    "pkg.pkgserver.dev",
			expectedVersion:  "v1alpha1",
			expectedKind:     APIServiceName,
			expectedResource: APIServiceName,
		},
		"crd": {
			path:             "test_data/config.pkg.pkgserver.dev_packagevariants.yaml",
			expectedGroup:    "config.pkg.pkgserver.dev",
			expectedVersion:  "v1alpha1",
			expectedKind:     "PackageVariant",
			expectedResource: "packagevariants",
		},
		"deployment": {
			path: "test_data/deployment.yaml",
		},
		"rbac": {
			path: "test_data/rbac-cluster-role-controller-permissions.yaml",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			outputs, err := getOutput(tc.path)
			if err != nil {
				t.Errorf("cannot get output resources, unexpected error\n%s", err.Error())
			}

			for _, rn := range outputs {
				fmt.Println(rn.MustString())
			}

			pkgID := pkgid.PackageID{
				Repository: "dummy",
				Realm:      pkgID.Realm,
				Package:    pkgID.Package,
				Revision:   "v1",
				Workspace:  "ws1",
			}
			pkgRev := pkgv1alpha1.BuildPackageRevision(
				metav1.ObjectMeta{Name: pkgRevName},
				pkgv1alpha1.PackageRevisionSpec{
					PackageID: pkgID,
				},
				pkgv1alpha1.PackageRevisionStatus{},
			)

			pkgRevRecorder := recorder.New[diag.Diagnostic]()
			ctx = context.WithValue(ctx, kformtypes.CtxKeyRecorder, pkgRevRecorder)
			r := NewStore()
			r.InitializeRecorder(pkgRevRecorder)
			r.UpdatePkgRevAPI(
				ctx,
				pkgRev,
				outputs,
			)
			if pkgRevRecorder.Get().Error() != nil {
				t.Errorf("unexpected error %s", pkgRevRecorder.Get().Error().Error())
			}

			apiEntries := map[storebackend.Key]*API{}
			r.catalogAPIStore.List(ctx, func(ctx context.Context, key storebackend.Key, api *API) {
				apiEntries[key] = api
				//fmt.Printf("apistore key: %s api %v\n", key.String(), api)
			})
			if len(apiEntries) != 0 {
				if tc.expectedGroup == "" {
					t.Errorf("expecting 0 entries, got: %v", apiEntries)
					return
				}
				key := storebackend.KeyFromNSN(types.NamespacedName{
					Namespace: tc.expectedGroup,
					Name:      tc.expectedKind,
				})
				api, ok := apiEntries[key]
				if !ok {
					t.Errorf("wrong key expecting: %s got: %v", key.String(), apiEntries)
					return
				}

				if diff := cmp.Diff(API{
					Type:     APIType_Package,
					Kind:     tc.expectedKind,
					Resource: tc.expectedResource,
					PkgID:    pkgID,
					Versions: map[string][]string{
						tc.expectedVersion: {pkgRevName},
					}}, *api); diff != "" {
					t.Errorf("-want, +got:\n%s", diff)
				}
			} else {
				if tc.expectedGroup != "" {
					t.Errorf("expecting 1 entry, got 0")
					return
				}
			}

			//if len(apiEntries) == 0 &&
			grMapapiEntries := map[storebackend.Key]string{}
			r.catalogGRMap.List(ctx, func(ctx context.Context, key storebackend.Key, kind string) {
				grMapapiEntries[key] = kind
				//fmt.Printf("grmap key: %s kind %s\n", key.String(), kind)
			})

			if len(grMapapiEntries) != 0 {
				if tc.expectedGroup == "" {
					t.Errorf("expecting 0 entries, got: %v", grMapapiEntries)
					return
				}
				key := storebackend.KeyFromNSN(types.NamespacedName{
					Namespace: tc.expectedGroup,
					Name:      tc.expectedResource,
				})
				kind, ok := grMapapiEntries[key]
				if !ok {
					t.Errorf("wrong key expecting: %s got: %v", key.String(), grMapapiEntries)
					return
				}

				if diff := cmp.Diff(tc.expectedKind, kind); diff != "" {
					t.Errorf("-want, +got:\n%s", diff)
				}
			} else {
				if tc.expectedGroup != "" {
					t.Errorf("expecting 1 entry, got 0")
					return
				}
			}
		})
	}
}
