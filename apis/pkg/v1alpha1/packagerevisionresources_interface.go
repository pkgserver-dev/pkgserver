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
	"context"

	"github.com/henderiw/apiserver-builder/pkg/builder/resource"
	"github.com/henderiw/apiserver-builder/pkg/builder/resource/resourcestrategy"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const PackageRevisionResourcesPlural = "packagerevisionresourceses"
const PackageRevisionResourcesSingular = "packagerevisionresources"

var _ resource.Object = &PackageRevisionResources{}
var _ resource.ObjectList = &PackageRevisionResourcesList{}

//var _ resource.ObjectWithStatusSubResource = &PackageRevisionResources{}

/*
func (PackageRevisionResourcesStatus) CopyTo(resource.ObjectWithStatusSubResource) {}

func (PackageRevisionResourcesStatus) SubResourceName() string {
	return fmt.Sprintf("%s/%s", PackageRevisionResourcesPlural, "status")
}

func (r *PackageRevisionResources) GetStatus() resource.StatusSubResource {
	return r.Status
}
*/

func (r *PackageRevisionResources) GetSingularName() string {
	return PackageRevisionResourcesSingular
}

func (PackageRevisionResources) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    SchemeGroupVersion.Group,
		Version:  SchemeGroupVersion.Version,
		Resource: PackageRevisionResourcesPlural,
	}
}

// IsStorageVersion returns true -- v1alpha1.Config is used as the internal version.
// IsStorageVersion implements resource.Object.
func (PackageRevisionResources) IsStorageVersion() bool {
	return true
}

// GetObjectMeta implements resource.Object
func (r *PackageRevisionResources) GetObjectMeta() *metav1.ObjectMeta {
	return &r.ObjectMeta
}

// NamespaceScoped returns true to indicate Fortune is a namespaced resource.
// NamespaceScoped implements resource.Object.
func (PackageRevisionResources) NamespaceScoped() bool {
	return true
}

// New implements resource.Object
func (PackageRevisionResources) New() runtime.Object {
	return &PackageRevisionResources{}
}

// NewList implements resource.Object
func (PackageRevisionResources) NewList() runtime.Object {
	return &PackageRevisionResourcesList{}
}

// GetCondition returns the condition based on the condition kind
func (r *PackageRevisionResources) GetCondition(t condition.ConditionType) condition.Condition {
	return r.Status.GetCondition(t)
}

// SetConditions sets the conditions on the resource. it allows for 0, 1 or more conditions
// to be set at once
func (r *PackageRevisionResources) SetConditions(c ...condition.Condition) {
	r.Status.SetConditions(c...)
}

// GetListMeta returns the ListMeta
func (r *PackageRevisionResourcesList) GetListMeta() *metav1.ListMeta {
	return &r.ListMeta
}

// BuildPackageRevisionResources returns an BuildPackageRevisionResources from a client Object a Spec/Status
func BuildPackageRevisionResources(meta metav1.ObjectMeta, spec PackageRevisionResourcesSpec, status PackageRevisionResourcesStatus) *PackageRevisionResources {
	return &PackageRevisionResources{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.Identifier(),
			Kind:       PackageRevisionResourcesKind,
		},
		ObjectMeta: meta,
		Spec:       spec,
		Status:     status,
	}
}

var _ resourcestrategy.TableConverter = &PackageRevisionResources{}
var _ resourcestrategy.TableConverter = &PackageRevisionResourcesList{}

var (
	pkgRevResourceDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: "the name of the cluster"},
		{Name: "Repository", Type: "string", Format: "repository", Description: "the name of the cluster"},
		{Name: "Package", Type: "string", Format: "package", Description: "the name of the cluster"},
		{Name: "Revision", Type: "string", Format: "revision", Description: "the name of the cluster"},
		{Name: "Workspace", Type: "string", Format: "workspace", Description: "the name of the cluster"},
		{Name: "Files", Type: "integer", Format: "files", Description: "the number of files in the package revision"},
	}
)

func (in *PackageRevisionResources) ConvertToTable(ctx context.Context, tableOptions runtime.Object) (*metav1.Table, error) {
	return &metav1.Table{
		ColumnDefinitions: pkgRevResourceDefinitions,
		Rows:              []metav1.TableRow{printPkgRevResourcesResource(in)},
	}, nil
}

func (in *PackageRevisionResourcesList) ConvertToTable(ctx context.Context, tableOptions runtime.Object) (*metav1.Table, error) {
	t := &metav1.Table{
		ColumnDefinitions: pkgRevResourceDefinitions,
	}
	for _, c := range in.Items {
		t.Rows = append(t.Rows, printPkgRevResourcesResource(&c))
	}
	return t, nil
}

func printPkgRevResourcesResource(c *PackageRevisionResources) metav1.TableRow {
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: c},
	}
	row.Cells = append(row.Cells, c.Name)
	row.Cells = append(row.Cells, c.Spec.PackageID.Repository)
	row.Cells = append(row.Cells, c.Spec.PackageID.Package)
	row.Cells = append(row.Cells, c.Spec.PackageID.Revision)
	row.Cells = append(row.Cells, c.Spec.PackageID.Workspace)
	row.Cells = append(row.Cells, len(c.Spec.Resources))
	return row
}

// ConvertPackageRevisionFieldSelector is the schema conversion function for normalizing the the FieldSelector for PackageRevision
func ConvertPackageRevisionResourcesFieldSelector(label, value string) (internalLabel, internalValue string, err error) {
	return ConvertPackageRevisionsFieldSelector(label, value)
}
