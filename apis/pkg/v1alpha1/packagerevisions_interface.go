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
	"fmt"
	"regexp"
	"strings"

	"github.com/henderiw/apiserver-builder/pkg/builder/resource"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const PackageRevisionPlural = "packagerevisions"
const PackageRevisionSingular = "packagerevision"

var _ resource.Object = &PackageRevision{}
var _ resource.ObjectList = &PackageRevisionList{}
var _ resource.ObjectWithStatusSubResource = &PackageRevision{}

func (PackageRevisionStatus) SubResourceName() string {
	return fmt.Sprintf("%s/%s", PackageRevisionPlural, "status")
}

func (r PackageRevisionStatus) CopyTo(obj resource.ObjectWithStatusSubResource) {
	pkgRev, ok := obj.(*PackageRevision)
	if ok {
		pkgRev.Status = r
	}
}

func (r *PackageRevision) GetStatus() resource.StatusSubResource {
	return r.Status
}

func (r *PackageRevision) GetSingularName() string {
	return PackageRevisionSingular
}

func (PackageRevision) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    SchemeGroupVersion.Group,
		Version:  SchemeGroupVersion.Version,
		Resource: PackageRevisionPlural,
	}
}

// IsStorageVersion returns true -- v1alpha1.Config is used as the internal version.
// IsStorageVersion implements resource.Object.
func (PackageRevision) IsStorageVersion() bool {
	return true
}

// GetObjectMeta implements resource.Object
func (r *PackageRevision) GetObjectMeta() *metav1.ObjectMeta {
	return &r.ObjectMeta
}

// NamespaceScoped returns true to indicate Fortune is a namespaced resource.
// NamespaceScoped implements resource.Object.
func (PackageRevision) NamespaceScoped() bool {
	return true
}

// New implements resource.Object
func (PackageRevision) New() runtime.Object {
	return &PackageRevision{}
}

// NewList implements resource.Object
func (PackageRevision) NewList() runtime.Object {
	return &PackageRevisionList{}
}

// GetCondition returns the condition based on the condition kind
func (r *PackageRevision) GetCondition(t condition.ConditionType) condition.Condition {
	return r.Status.GetCondition(t)
}

// HasCondition returns the if the condition is set
func (r *PackageRevision) HasCondition(t condition.ConditionType) bool {
	return r.Status.HasCondition(t)
}

// SetConditions sets the conditions on the resource. it allows for 0, 1 or more conditions
// to be set at once
func (r *PackageRevision) SetConditions(c ...condition.Condition) {
	r.Status.SetConditions(c...)
}

func (r *PackageRevision) ValidateRepository() error {
	repoName := pkgid.GetRepoNameFromPkgRevName(r.GetName())
	if repoName != r.Spec.PackageID.Repository {
		return fmt.Errorf("the name of the %s, must start with the repo name as it is used in lookups for %s, got name: %s, spec: %s",
			PackageRevisionSingular,
			PackageRevisionResourcesSingular,
			repoName,
			r.Spec.PackageID.Repository,
		)
	}
	return nil
}

func (r *PackageRevision) ValidateDiscoveryAnnotation() error {
	a := r.GetAnnotations()
	err := fmt.Errorf("discovery annotation not present")
	if len(a) == 0 {
		return err
	}
	if _, ok := a[DiscoveredPkgRevKey]; !ok {
		return err
	}
	return nil
}

// GetListMeta returns the ListMeta
func (r *PackageRevisionList) GetListMeta() *metav1.ListMeta {
	return &r.ListMeta
}

// BuildPackageRevision returns an PackageRevision from a client Object a Spec/Status
func BuildPackageRevision(meta metav1.ObjectMeta, spec PackageRevisionSpec, status PackageRevisionStatus) *PackageRevision {
	return &PackageRevision{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.Identifier(),
			Kind:       PackageRevisionKind,
		},
		ObjectMeta: meta,
		Spec:       spec,
		Status:     status,
	}
}

// ValidateWorkspaceName validates WorkspaceName. It must:
//   - be at least 1 and at most 63 characters long
//   - contain only lowercase alphanumeric characters or '-'
//   - start and end with an alphanumeric character.
//
// '/ ' should never be allowed, because we use '/' to
// delimit branch names (e.g. the 'drafts/' prefix).
func ValidateWorkspaceName(workspace string) error {
	wn := string(workspace)
	if len(wn) > 63 || len(wn) == 0 {
		return fmt.Errorf("workspaceName %q must be at least 1 and at most 63 characters long", wn)
	}
	if strings.HasPrefix(wn, "-") || strings.HasSuffix(wn, "-") {
		return fmt.Errorf("workspaceName %q must start and end with an alphanumeric character", wn)
	}

	match, err := regexp.MatchString(`^[a-z0-9-]+$`, wn)
	if err != nil {
		return err
	}
	if !match {
		return fmt.Errorf("workspaceName %q must be comprised only of lowercase alphanumeric characters and '-'", wn)
	}
	return nil
}

func ValidateUpdateLifeCycle(newPkgRev, oldPkgRev *PackageRevision) error {
	// validate lifecycle
	switch oldPkgRev.Spec.Lifecycle {
	case PackageRevisionLifecycleDraft:
		switch newPkgRev.Spec.Lifecycle {
		case PackageRevisionLifecycleDraft:
		case PackageRevisionLifecycleProposed:
		case PackageRevisionLifecycleDeletionProposed:
		case PackageRevisionLifecyclePublished:
			return fmt.Errorf("unsupported lifecycle transition %s -> %s", oldPkgRev.Spec.Lifecycle, newPkgRev.Spec.Lifecycle)
		default:
			return fmt.Errorf("unsupported lifecycle %s", newPkgRev.Spec.Lifecycle)
		}
	case PackageRevisionLifecycleProposed:
		switch newPkgRev.Spec.Lifecycle {
		case PackageRevisionLifecycleDraft:
		case PackageRevisionLifecycleProposed:
		case PackageRevisionLifecycleDeletionProposed:
		case PackageRevisionLifecyclePublished:
		default:
			return fmt.Errorf("unsupported lifecycle %s", newPkgRev.Spec.Lifecycle)
		}
	case PackageRevisionLifecyclePublished:
		switch newPkgRev.Spec.Lifecycle {
		case PackageRevisionLifecycleDraft:
			return fmt.Errorf("unsupported lifecycle transition %s -> %s", oldPkgRev.Spec.Lifecycle, newPkgRev.Spec.Lifecycle)
		case PackageRevisionLifecycleProposed:
			return fmt.Errorf("unsupported lifecycle transition %s -> %s", oldPkgRev.Spec.Lifecycle, newPkgRev.Spec.Lifecycle)
		case PackageRevisionLifecycleDeletionProposed:
		case PackageRevisionLifecyclePublished:
		default:
			return fmt.Errorf("unsupported lifecycle %s", newPkgRev.Spec.Lifecycle)
		}
	case PackageRevisionLifecycleDeletionProposed:
		switch newPkgRev.Spec.Lifecycle {
		case PackageRevisionLifecycleDraft:
			return fmt.Errorf("unsupported lifecycle transition %s -> %s", oldPkgRev.Spec.Lifecycle, newPkgRev.Spec.Lifecycle)
		case PackageRevisionLifecycleProposed:
			return fmt.Errorf("unsupported lifecycle transition %s -> %s", oldPkgRev.Spec.Lifecycle, newPkgRev.Spec.Lifecycle)
		case PackageRevisionLifecycleDeletionProposed:
		case PackageRevisionLifecyclePublished:
		default:
			return fmt.Errorf("unsupported lifecycle %s", newPkgRev.Spec.Lifecycle)
		}
	default:
		return fmt.Errorf("unsupported lifecycle %s", newPkgRev.Spec.Lifecycle)
	}
	return nil
}

// ConvertPackageRevisionsFieldSelector is the schema conversion function for normalizing the FieldSelector for PackageRevisions
func ConvertPackageRevisionsFieldSelector(label, value string) (internalLabel, internalValue string, err error) {
	switch label {
	case "metadata.name":
		return label, value, nil
	case "metadata.namespace":
		return label, value, nil
	case "spec.packageID.target", "spec.packageID.repository", "spec.packageID.realm", "spec.packageID.package", "spec.packageID.workspace", "spec.packageID.revision", "spec.lifecycle":
		return label, value, nil
	default:
		return "", "", fmt.Errorf("%q is not a known field selector", label)
	}
}

func (r *PackageRevision) HasReadinessGate(cType condition.ConditionType) bool {
	for _, readinessGate := range r.Spec.ReadinessGates {
		if readinessGate.ConditionType == cType {
			return true
		}
	}
	return false
}

func (r *PackageRevision) NextReadinessGate(cType condition.ConditionType) condition.ConditionType {
	idx := 0
	for i, readinessGate := range r.Spec.ReadinessGates {
		if readinessGate.ConditionType == cType {
			idx = i
			break
		}
	}
	if idx < len(r.Spec.ReadinessGates)-1 {
		return r.Spec.ReadinessGates[idx+1].ConditionType
	}
	return condition.ConditionTypeEnd
}

func (r *PackageRevision) GetPreviousCondition(cType condition.ConditionType) condition.Condition {
	for i, readinessGate := range r.Spec.ReadinessGates {
		if readinessGate.ConditionType == cType {
			if i == 0 {
				return condition.ConditionTrue()
			} else {
				return r.GetCondition(r.Spec.ReadinessGates[i-1].ConditionType)
			}
		}
	}
	// if this condition was not found we return false
	return condition.ConditionFalse()
}
