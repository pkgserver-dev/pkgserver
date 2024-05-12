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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RepositoryType string

const (
	RepositoryTypeGit RepositoryType = "git"
	RepositoryTypeOCI RepositoryType = "oci"
)

// RepositorySpec defines the desired state of Repository
type RepositorySpec struct {
	// Type of the repository (i.e. git, OCI)
	// +kubebuilder:validation:Enum=git;oci
	// +kubebuilder:default:="git"
	Type RepositoryType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type"`
	// Git repository details. Required if `type` is `git`. Ignored if `type` is not `git`.
	Git *GitRepository `json:"git,omitempty" protobuf:"bytes,2,opt,name=git"`
	// OCI repository details. Required if `type` is `oci`. Ignored if `type` is not `oci`.
	Oci *OciRepository `json:"oci,omitempty" protobuf:"bytes,3,opt,name=oci"`
	// The repository is a deployment repository;
	// When set to true this is considered a WET package; when false this is a DRY package
	Deployment bool `json:"deployment,omitempty" protobuf:"varint,4,opt,name=deployment"`
}

// GitRepository describes a Git repository.
// TODO: authentication methods
type GitRepository struct {
	// URL specifies the base URL for a given repository for example:
	//   `https://github.com/GoogleCloudPlatform/catalog.git`
	URL string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`
	// Name of the reference (typically a branch) containing the packages. Finalized packages will be committed to this branch (if the repository allows write access). If unspecified, defaults to "main".
	Ref string `json:"ref,omitempty" protobuf:"bytes,2,opt,name=ref"`
	// Directory within the Git repository where the packages are stored. If unspecified, defaults to root directory.
	Directory string `json:"directory,omitempty" protobuf:"bytes,3,opt,name=directory"`
	// Credentials defines the name of the secret that holds the credentials to connect to a repository
	Credentials string `json:"credentials,omitempty" protobuf:"bytes,4,opt,name=credentials"`
}

// OciRepository describes a repository compatible with the Open Container Registry standard.
type OciRepository struct {
	// Registry is the address of the OCI registry
	Registry string `json:"registry,omitempty" protobuf:"bytes,1,opt,name=registry"`
	// Credentials defines the name of the secret that holds the credentials to connect to the OCI registry
	Credentials string `json:"credentials,omitempty" protobuf:"bytes,2,opt,name=credentials"`
}

// RepositoryStatus defines the observed state of Repository
type RepositoryStatus struct {
	// ConditionedStatus provides the status of the Repository using conditions
	condition.ConditionedStatus `json:",inline" protobuf:"bytes,1,opt,name=conditionedStatus"`
}

// +kubebuilder:object:root=true
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="DEPLOYMENT",type=boolean,JSONPath=`.spec.deployment`
// +kubebuilder:printcolumn:name="TYPE",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="ADDRESS",type=string,JSONPath=`.spec['git','oci']['url','registry']`
// +kubebuilder:resource:categories={kform,pkg}
// Repository is the Repository for the Repository API
// +k8s:openapi-gen=true
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   RepositorySpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status RepositoryStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +kubebuilder:object:root=true
// RepositoryList contains a list of Repositorys
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RepositoryList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Repository `json:"items" protobuf:"bytes,2,rep,name=items"`
}

var (
	RepositoryKind = reflect.TypeOf(Repository{}).Name()
)
