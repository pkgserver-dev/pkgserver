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
	"reflect"

	"github.com/pkgserver-dev/pkgserver/apis/condition"
	"github.com/pkgserver-dev/pkgserver/apis/pkgrevid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PackageRevisionResourcesSpec defines the desired state of PackageRevisionResources
type PackageRevisionResourcesSpec struct {
	PackageRevID pkgrevid.PackageRevID `json:"packageRevID" protobuf:"bytes,6,opt,name=packageRevID"`
	// Resources define the content of the resources key is the name of the KRM file,
	// value defines the the content of the KRM reource
	Resources map[string]string `json:"resources,omitempty" protobuf:"bytes,1,opt,name=resources"`
}

// PackageRevisionResourcesStatus defines the observed state of PackageRevisionResources
type PackageRevisionResourcesStatus struct {
	// ConditionedStatus provides the status of the Readiness using conditions
	// if the condition is true the other attributes in the status are meaningful
	condition.ConditionedStatus `json:",inline" protobuf:"bytes,1,opt,name=conditionedStatus"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//	PackageRevisionResource is the Schema for the PackageRevisionResource API
//
// +k8s:openapi-gen=true
type PackageRevisionResources struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   PackageRevisionResourcesSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status PackageRevisionResourcesStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// PackageRevisionResourceList contains a list of PackageRevisionResources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PackageRevisionResourcesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []PackageRevisionResources `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// PackageRevisionResources type metadata.
var (
	PackageRevisionResourcesKind = reflect.TypeOf(PackageRevisionResources{}).Name()
)
