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

import "path"

// +k8s:openapi-gen=true
type Upstream struct {
	// Repository defines the name of the Repository object containing this package.
	Repository string `json:"repository,omitempty" protobuf:"bytes,1,opt,name=repository"`
	// Realm defines the scope in which the package is relevant
	Realm string `json:"realm,omitempty" protobuf:"bytes,2,opt,name=realm"`
	// Package defines the name of package in the repository.
	Package string `json:"package,omitempty" protobuf:"bytes,3,opt,name=package"`
	// Revision defines the revision of the package once published
	Revision string `json:"revision,omitempty" protobuf:"bytes,4,opt,name=revision"`
}

func (r *Upstream) PkgRevName() string {
	pkgID := &PackageRevID{}
	return pkgID.PkgRevString()
}

func (r *Upstream) PkgString() string {
	return path.Join(r.Realm, r.Package)
}

// +k8s:openapi-gen=true
type Downstream struct {
	// Target defines the target for the package; not relevant for catalog packages
	// e.g. a cluster
	Target string `json:"target,omitempty" protobuf:"bytes,1,opt,name=target"`
	// Repository defines the name of the Repository object containing this package.
	Repository string `json:"repository,omitempty" protobuf:"bytes,2,opt,name=repository"`
	// Realm defines the scope in which the package is relevant
	Realm string `json:"realm,omitempty" protobuf:"bytes,3,opt,name=realm"`
	// Package defines the name of package in the repository.
	Package string `json:"package,omitempty" protobuf:"bytes,4,opt,name=package"`
}

func (r *Downstream) PkgString() string {
	return path.Join(r.Realm, r.Package)
}
