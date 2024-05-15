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

package v1alpha1

import (
	"fmt"

	"github.com/henderiw/apiserver-builder/pkg/builder/resource"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type PackageRevisionApproval struct{}

var _ resource.ArbitrarySubResource = &PackageRevisionApproval{}

func (r *PackageRevisionApproval) Destroy() {}

// New returns an empty object that can be used with Create and Update after request data has been put into it.
// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
func (a *PackageRevisionApproval) New() runtime.Object {
	return &PackageRevision{}
}

func (r *PackageRevisionApproval) SubResourceName() string {
	return fmt.Sprintf("%s/%s", PackageRevisionPlural, "aproval")
}
