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

package packagerevisionresource

import (
	"github.com/henderiw/apiserver-store/pkg/generic/registry"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewTableConvertor(gr schema.GroupResource) registry.TableConvertor {
	return registry.TableConvertor{
		Resource: gr,
		Cells: func(obj runtime.Object) []interface{} {
			prr, ok := obj.(*pkgv1alpha1.PackageRevisionResources)
			if !ok {
				return nil
			}
			return []interface{}{
				prr.Name,
				prr.GetCondition(condition.ConditionTypeReady).Status,
				prr.Spec.PackageRevID.Repository,
				prr.Spec.PackageRevID.Target,
				prr.Spec.PackageRevID.Realm,
				prr.Spec.PackageRevID.Package,
				prr.Spec.PackageRevID.Revision,
				prr.Spec.PackageRevID.Workspace,
				len(prr.Spec.Resources),
			}
		},
		Columns: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Ready", Type: "string"},
			{Name: "Repository", Type: "string"},
			{Name: "Realm", Type: "string"},
			{Name: "Package", Type: "string"},
			{Name: "Package", Type: "string"},
			{Name: "Revision", Type: "string"},
			{Name: "workspace", Type: "string"},
			{Name: "Files", Type: "integer"},
		},
	}
}
