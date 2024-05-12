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
	"strings"

	"github.com/pkgserver-dev/pkgserver/apis/condition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetCondition returns the condition based on the condition kind
func (r *Repository) GetCondition(t condition.ConditionType) condition.Condition {
	return r.Status.GetCondition(t)
}

// SetConditions sets the conditions on the resource. it allows for 0, 1 or more conditions
// to be set at once
func (r *Repository) SetConditions(c ...condition.Condition) {
	r.Status.SetConditions(c...)
}

func (r *Repository) Validate() error {
	switch repositoryType := r.Spec.Type; repositoryType {
	case RepositoryTypeOCI:
		if r.Spec.Oci == nil {
			return fmt.Errorf("cannot open a git repository without a git spec")
		}
		if r.Spec.Oci.Registry == "" {
			return fmt.Errorf("cannot open a oci registry without a URL")
		}
		return fmt.Errorf("oci type not implemented")
	case RepositoryTypeGit:
		if r.Spec.Git == nil {
			return fmt.Errorf("cannot open a git repository without a git spec")
		}
		if r.Spec.Git.URL == "" {
			return fmt.Errorf("cannot open a git repository without a repository URL")
		}
	default:
		// should not happen since the api validation will handle this
		return fmt.Errorf("unsupported type %q", repositoryType)
	}
	return nil
}

func (r *Repository) GetKey() string {
	switch repositoryType := r.Spec.Type; repositoryType {
	case RepositoryTypeOCI:
		if r.Spec.Oci != nil {
			return fmt.Sprintf("oci://%s", r.Spec.Oci.Registry)
		}
	case RepositoryTypeGit:
		if r.Spec.Git != nil {
			return fmt.Sprintf("git://%s%s", r.Spec.Git.URL, r.Spec.Git.Directory)
		}
	}
	return ""
}

func (r *Repository) GetURL() string {
	switch repositoryType := r.Spec.Type; repositoryType {
	case RepositoryTypeOCI:
		if r.Spec.Oci != nil {
			return r.Spec.Oci.Registry
		}
	case RepositoryTypeGit:
		if r.Spec.Git != nil {
			return r.Spec.Git.URL
		}
	}
	return ""
}

func (r *Repository) GetCredentialSecret() string {
	switch repositoryType := r.Spec.Type; repositoryType {
	case RepositoryTypeOCI:
		if r.Spec.Oci != nil {
			return r.Spec.Oci.Credentials
		}
	case RepositoryTypeGit:
		if r.Spec.Git != nil {
			return r.Spec.Git.Credentials
		}
	}
	return ""
}

func (r *Repository) GetRef() string {
	switch repositoryType := r.Spec.Type; repositoryType {
	case RepositoryTypeGit:
		if r.Spec.Git != nil {
			return r.Spec.Git.Ref
		}
	}
	return ""
}

func (r *Repository) GetDirectory() string {
	switch repositoryType := r.Spec.Type; repositoryType {
	case RepositoryTypeGit:
		if r.Spec.Git != nil {
			return strings.Trim(r.Spec.Git.Directory, "/")
		}
	}
	return ""
}

// BuildRepository returns an Repository from a client Object a Spec/Status
func BuildRepository(meta metav1.ObjectMeta, spec RepositorySpec, status RepositoryStatus) *Repository {
	return &Repository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.Identifier(),
			Kind:       RepositoryKind,
		},
		ObjectMeta: meta,
		Spec:       spec,
		Status:     status,
	}
}
