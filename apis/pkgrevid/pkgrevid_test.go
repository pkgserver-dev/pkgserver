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

package pkgrevid

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseTag(t *testing.T) {
	cases := map[string]struct {
		tag           string
		catalog       bool
		expectedPkgID *PackageRevID
		expectedError bool
	}{
		"Catalog-Normal-Realm": {
			tag:     "a/b/c/v3",
			catalog: true,
			expectedPkgID: &PackageRevID{
				Target:   PkgTarget_Catalog,
				Realm:    "a/b",
				Package:  "c",
				Revision: "v3",
			},
			expectedError: false,
		},
		"Catalog-Normal-NoRealm": {
			tag:     "c/v3",
			catalog: true,
			expectedPkgID: &PackageRevID{
				Target:   PkgTarget_Catalog,
				Realm:    "",
				Package:  "c",
				Revision: "v3",
			},
			expectedError: false,
		},
		"Catalog-Error": {
			tag:           "v3",
			catalog:       true,
			expectedError: true,
		},
		"Target-Normal-Realm": {
			tag: "a/b/c/d/v3",
			expectedPkgID: &PackageRevID{
				Target:   "a",
				Realm:    "b/c",
				Package:  "d",
				Revision: "v3",
			},
			expectedError: false,
		},
		"Target-Normal-NoRealm": {
			tag: "a/d/v3",
			expectedPkgID: &PackageRevID{
				Target:   "a",
				Realm:    "",
				Package:  "d",
				Revision: "v3",
			},
			expectedError: false,
		},
		"Target-Error": {
			tag:           "a/v3",
			expectedError: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			pkgID, err := ParseTag(tc.tag, tc.catalog)
			if err != nil {
				if !tc.expectedError {
					t.Errorf("unexpected error, got: %s", err.Error())
				}
				return
			}
			if tc.expectedError {
				t.Errorf("expected error, got: nil")
				return
			}

			if diff := cmp.Diff(*pkgID, *tc.expectedPkgID); diff != "" {
				t.Errorf("want %v, got: %v", *tc.expectedPkgID, *pkgID)
			}
		})
	}
}

func TestParseBranch(t *testing.T) {
	cases := map[string]struct {
		tag           string
		catalog       bool
		expectedPkgID *PackageRevID
		expectedError bool
	}{
		"Catalog-Normal-Realm": {
			tag:     "a/b/c/ws3",
			catalog: true,
			expectedPkgID: &PackageRevID{
				Target:    PkgTarget_Catalog,
				Realm:     "a/b",
				Package:   "c",
				Workspace: "ws3",
			},
			expectedError: false,
		},
		"Catalog-Normal-NoRealm": {
			tag:     "c/ws3",
			catalog: true,
			expectedPkgID: &PackageRevID{
				Target:    PkgTarget_Catalog,
				Realm:     "",
				Package:   "c",
				Workspace: "ws3",
			},
			expectedError: false,
		},
		"Catalog-Error": {
			tag:           "ws3",
			catalog:       true,
			expectedError: true,
		},
		"Target-Normal-Realm": {
			tag: "a/b/c/d/ws3",
			expectedPkgID: &PackageRevID{
				Target:    "a",
				Realm:     "b/c",
				Package:   "d",
				Workspace: "ws3",
			},
			expectedError: false,
		},
		"Target-Normal-NoRealm": {
			tag: "a/d/ws3",
			expectedPkgID: &PackageRevID{
				Target:    "a",
				Realm:     "",
				Package:   "d",
				Workspace: "ws3",
			},
			expectedError: false,
		},
		"Target-Error": {
			tag:           "a/ws3",
			expectedError: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			pkgID, err := ParseBranch(tc.tag, tc.catalog)
			if err != nil {
				if !tc.expectedError {
					t.Errorf("unexpected error, got: %s", err.Error())
				}
				return
			}
			if tc.expectedError {
				t.Errorf("expected error, got: nil")
				return
			}

			if diff := cmp.Diff(*pkgID, *tc.expectedPkgID); diff != "" {
				t.Errorf("want %v, got: %v", *tc.expectedPkgID, *pkgID)
			}
		})
	}
}
