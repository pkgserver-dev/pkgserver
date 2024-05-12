/*
Copyright 2023 Nokia.

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
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type PackageRevisionLifecycle string

const (
	PackageRevisionLifecycleDraft            PackageRevisionLifecycle = "draft"
	PackageRevisionLifecycleProposed         PackageRevisionLifecycle = "proposed"
	PackageRevisionLifecyclePublished        PackageRevisionLifecycle = "published"
	PackageRevisionLifecycleDeletionProposed PackageRevisionLifecycle = "deletionProposed"
)

type TaskType string

const (
	TaskTypeInit  TaskType = "init"
	TaskTypeClone TaskType = "clone"
)

type UpdatePolicy string

const (
	UpdatePolicy_Loose  UpdatePolicy = "loose"
	UpdatePolicy_Strict UpdatePolicy = "strict"
)

// PackageRevisionSpec defines the desired state of PackageRevision
type PackageRevisionSpec struct {
	PackageID pkgid.PackageID `json:"packageID" protobuf:"bytes,1,opt,name=packageID"`
	// Lifecycle defines the lifecycle of the resource
	Lifecycle PackageRevisionLifecycle `json:"lifecycle,omitempty" protobuf:"bytes,2,opt,name=lifecycle"`
	// UpdatePolicy is the policy used to update resources
	// Loose updates the resources w/o removing the other entries in the package
	// strict updates the resources and removes unnessesary entries in the package
	UpdatePolicy UpdatePolicy `json:"updatePolicy,omitempty" protobuf:"bytes,3,rep,name=updatePolicy"`
	// Task is the task to be performed when creating this package revisision
	Tasks []Task `json:"tasks,omitempty" protobuf:"bytes,4,rep,name=tasks"`
	// ReadinessGates define the conditions that need to be acted upon before considering the PackageRevision
	// ready for approval
	ReadinessGates []condition.ReadinessGate `json:"readinessGates,omitempty" protobuf:"bytes,5,rep,name=readinessGates"`
	// Upstream identifies the upstream this package is originated from
	Upstream *pkgid.Upstream `json:"upstream,omitempty" protobuf:"bytes,6,opt,name=upstream"`
	// Inputs define the inputs defined for the PackageContext
	//+kubebuilder:pruning:PreserveUnknownFields
	Inputs []runtime.RawExtension `json:"inputs,omitempty" protobuf:"bytes,7,rep,name=inputs"`
}

type Task struct {
	Type TaskType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=TaskType"`
}

// PackageRevisionStatus defines the observed state of PackageRevision
type PackageRevisionStatus struct {
	// ConditionedStatus provides the status of the Readiness using conditions
	// if the condition is true the other attributes in the status are meaningful
	condition.ConditionedStatus `json:",inline" yaml:",inline" protobuf:"bytes,1,opt,name=conditionedStatus"`

	// PublishedBy is the identity of the user who approved the packagerevision.
	PublishedBy string `json:"publishedBy,omitempty" protobuf:"bytes,2,opt,name=publishedBy"`

	// PublishedAt is the time when the packagerevision were approved.
	PublishedAt metav1.Time `json:"publishTimestamp,omitempty" protobuf:"bytes,3,opt,name=publishTimestamp"`

	Dependencies *PackageRevisionDependencies `json:"dependencies,omitempty" protobuf:"bytes,4,opt,name=dependencies"`
}

type PackageRevisionDependencies struct {
	// Summary Error in case an error was discovered
	Error *string `json:"error,omitempty" protobuf:"bytes,1,opt,name=error"`
	// Errors define the detailed error per reference
	Errors []*PackageRevisionDependencyError `json:"errors,omitempty" protobuf:"bytes,2,rep,name=errors"`
	// Warnings define the detailed warning per reference
	Warnings []*PackageRevisionDependencyError `json:"warnings,omitempty" protobuf:"bytes,3,rep,name=warnings"`
	// Dependencies define the dependency details per reference
	Dependencies []*PackageRevisionDependency `json:"dependencies,omitempty" protobuf:"bytes,4,rep,name=dependencies"`
}

type PackageRevisionDependencyType string

const (
	PackageRevisionDependencyType_Core    PackageRevisionDependencyType = "core"
	PackageRevisionDependencyType_Package PackageRevisionDependencyType = "package"
	PackageRevisionDependencyType_Runtime PackageRevisionDependencyType = "runtime"
)

type PackageRevisionDependencyError struct {
	// Reference the error is sourced from (apiversion, kind, namespace, name)
	Ref *corev1.ObjectReference `json:"ref,omitempty" protobuf:"bytes,1,opt,name=ref"`
	// Reason describes the reason why the dependency failed
	Reason string `json:"reason,omitempty" protobuf:"bytes,2,opt,name=reason"`
}

type PackageRevisionDependency struct {
	// Type defines the type of dependency we refer to
	Type PackageRevisionDependencyType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type"`
	// Reference the dependency is sourced from (apiversion, kind, namespace, name)
	Ref *corev1.ObjectReference `json:"ref,omitempty" protobuf:"bytes,2,opt,name=ref"`
	// PackageDependencies define the package dependencies the reference depend upon (repository, realm, package, (revision)
	PackageDependencies []*pkgid.Upstream `json:"packageDependencies,omitempty" protobuf:"bytes,3,rep,name=packageDependencies"`
	// RuntimeDependencies define the runtime dependencies the reference depend upon (apiversion, kind, namespace, name)
	RuntimeDependencies []*corev1.ObjectReference `json:"runtimeDependencies,omitempty" protobuf:"bytes,4,rep,name=runtimeDependencies"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//	PackageRevision is the Schema for the PackageRevision API
//
// +k8s:openapi-gen=true
type PackageRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   PackageRevisionSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status PackageRevisionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// PackageRevisionList contains a list of PackageRevisions
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PackageRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []PackageRevision `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// PackageRevision type metadata.
var (
	PackageRevisionKind = reflect.TypeOf(PackageRevision{}).Name()
)
