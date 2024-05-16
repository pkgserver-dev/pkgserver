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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type AdoptionPolicy string

const (
	AdoptionPolicyAdoptExisting AdoptionPolicy = "adoptExisting"
	AdoptionPolicyAdoptNone     AdoptionPolicy = "adoptNone"
)

type DeletionPolicy string

const (
	DeletionPolicyDelete DeletionPolicy = "delete"
	DeletionPolicyOrphan DeletionPolicy = "orphan"
)

// PackageVariantSpec defines the desired state of PackageVariant
type PackageVariantSpec struct {
	// Upstream defines the upstream PackageRevision reference
	Upstream pkgrevid.Upstream `json:"upstream,omitempty" protobuf:"bytes,1,opt,name=upstream"`
	// Downstream defines the downstream Package information
	Downstream pkgrevid.Downstream `json:"downstream,omitempty" protobuf:"bytes,2,opt,name=downstream"`
	// PackageContext defines the context of the PackageVariant
	PackageContext PackageContext `json:"packageContext,omitempty" protobuf:"bytes,3,opt,name=packageContext"`
	// +kubebuilder:validation:Enum=adoptExisting;adoptNone;
	// +kubebuilder:default:="adoptNone"
	AdoptionPolicy AdoptionPolicy `json:"adoptionPolicy,omitempty" protobuf:"bytes,4,opt,name=adoptionPolicy"`
	// +kubebuilder:validation:Enum=delete;orphan;
	// +kubebuilder:default:="delete"
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty" protobuf:"bytes,5,opt,name=deletionPolicy"`
}

// PackageContext defines the context of the Package
type PackageContext struct {
	// Inputs define the inputs defined for the PackageContext
	//+kubebuilder:pruning:PreserveUnknownFields
	Inputs []runtime.RawExtension `json:"inputs,omitempty" protobuf:"bytes,1,rep,name=inputs"`
	// Annotations is a key value map to be copied to the PackageContext Annotations.
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,2,rep,name=annotations"`
	// Labels is a key value map to be copied to the PackageContext Labels.
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,3,rep,name=labels"`
	// ReadinessGates define the conditions that need to be acted upon before considering the PackageRevision
	// ready for approval
	ReadinessGates []condition.ReadinessGate `json:"readinessGates,omitempty" protobuf:"bytes,4,rep,name=readinessGates"`
}

// PackageVariantStatus defines the observed state of PackageVariant
type PackageVariantStatus struct {
	// ConditionedStatus provides the status of the PackageVariant using conditions
	condition.ConditionedStatus `json:",inline" protobuf:"bytes,1,opt,name=conditionedStatus"`
}

// +kubebuilder:object:root=true
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"

// +kubebuilder:resource:categories={pkg}
// PackageVariant is the PackageVariant for the PackageVariant API
// +k8s:openapi-gen=true
type PackageVariant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   PackageVariantSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status PackageVariantStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +kubebuilder:object:root=true
// PackageVariantList contains a list of PackageVariants
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PackageVariantList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []PackageVariant `json:"items" protobuf:"bytes,2,rep,name=items"`
}

var (
	PackageVariantKind = reflect.TypeOf(PackageVariant{}).Name()
)
