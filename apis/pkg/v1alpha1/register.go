/*
Copyright 2023 The Nokia.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "pkg.pkgserver.dev"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion contains the API group and version information for the types in this package.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	// AddToScheme applies all the stored functions to the scheme. A non-nil error
	// indicates that one function failed and the attempt was abandoned.
	AddToScheme = localSchemeBuilder.AddToScheme

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	schemeBuilder      runtime.SchemeBuilder
	localSchemeBuilder = &schemeBuilder
)

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func init() {
	localSchemeBuilder.Register(AddKnownTypes)
}

// Adds the list of known types to the given scheme.
func AddKnownTypes(scheme *runtime.Scheme) error {
	// +kubebuilder:scaffold:install

	scheme.AddKnownTypes(SchemeGroupVersion,
		&PackageRevision{},
		&PackageRevisionList{},
		&PackageRevisionResources{},
		&PackageRevisionResourcesList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
