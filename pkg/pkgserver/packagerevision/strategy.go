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

package packagerevision

import (
	"context"
	"fmt"

	"github.com/henderiw/logger/log"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewStrategy creates and returns a fischerStrategy instance
func NewStrategy(typer runtime.ObjectTyper, client client.Client) packageRevisionStrategy {
	return packageRevisionStrategy{
		ObjectTyper:   typer,
		NameGenerator: names.SimpleNameGenerator,
		client:        client,
	}
}

// MatchPackageRevision is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchPackageRevision(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// GetAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not a PackageRevision
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	api, ok := obj.(*pkgv1alpha1.PackageRevision)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a PackageRevision")
	}
	return labels.Set(api.ObjectMeta.Labels), SelectableFields(api), nil
}

// SelectableFields returns a field set that represents the object.
func SelectableFields(obj *pkgv1alpha1.PackageRevision) fields.Set {
	return fields.Set{
		"metadata.namespace":        obj.Namespace,
		"metadata.name":             obj.Name,
		"spec.packageRevID.target":     obj.Spec.PackageRevID.Target,
		"spec.packageRevID.repository": obj.Spec.PackageRevID.Repository,
		"spec.packageRevID.package":    obj.Spec.PackageRevID.Package,
		"spec.packageRevID.revision":   obj.Spec.PackageRevID.Revision,
		"spec.packageRevID.workspace":  obj.Spec.PackageRevID.Workspace,
		"spec.packageRevID.lifecycle":  string(obj.Spec.Lifecycle),
	}
	//fieldSet := generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

func Indexers() *cache.Indexers {
	return &cache.Indexers{
		"f:spec.packageRevID.target": func(obj interface{}) ([]string, error) {
			pkgRev, ok := obj.(*pkgv1alpha1.PackageRevision)
			if !ok {
				return []string{}, nil
			}
			return []string{pkgRev.Spec.PackageRevID.Target}, nil
		},
		"f:spec.packageRevID.repository": func(obj interface{}) ([]string, error) {
			pkgRev, ok := obj.(*pkgv1alpha1.PackageRevision)
			if !ok {
				return []string{}, nil
			}
			return []string{pkgRev.Spec.PackageRevID.Repository}, nil
		},
		"f:spec.packageRevID.realm": func(obj interface{}) ([]string, error) {
			pkgRev, ok := obj.(*pkgv1alpha1.PackageRevision)
			if !ok {
				return []string{}, nil
			}
			return []string{pkgRev.Spec.PackageRevID.Realm}, nil
		},
		"f:spec.packageRevID.package": func(obj interface{}) ([]string, error) {
			pkgRev, ok := obj.(*pkgv1alpha1.PackageRevision)
			if !ok {
				return []string{}, nil
			}
			return []string{pkgRev.Spec.PackageRevID.Package}, nil
		},
		"f:spec.packageRevID.revision": func(obj interface{}) ([]string, error) {
			pkgRev, ok := obj.(*pkgv1alpha1.PackageRevision)
			if !ok {
				return []string{}, nil
			}
			return []string{pkgRev.Spec.PackageRevID.Revision}, nil
		},
		"f:spec.packageRevID.workspace": func(obj interface{}) ([]string, error) {
			pkgRev, ok := obj.(*pkgv1alpha1.PackageRevision)
			if !ok {
				return []string{}, nil
			}
			return []string{pkgRev.Spec.PackageRevID.Revision}, nil
		},
		"f:spec.lifecycle": func(obj interface{}) ([]string, error) {
			pkgRev, ok := obj.(*pkgv1alpha1.PackageRevision)
			if !ok {
				return []string{}, nil
			}
			return []string{string(pkgRev.Spec.Lifecycle)}, nil
		},
	}
}

var _ rest.RESTCreateStrategy = packageRevisionStrategy{}
var _ rest.RESTUpdateStrategy = packageRevisionStrategy{}
var _ rest.RESTDeleteStrategy = packageRevisionStrategy{}

type packageRevisionStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
	client client.Client
}

func (packageRevisionStrategy) NamespaceScoped() bool {
	return true
}

func (packageRevisionStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (packageRevisionStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (r packageRevisionStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	// only used for create validation
	log := log.FromContext(ctx)
	log.Debug("validate packageRevision create")
	var allErrs field.ErrorList
	pkgRev, ok := obj.(*pkgv1alpha1.PackageRevision)
	if !ok {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath(""),
			obj,
			fmt.Errorf("expected PackageRevision object, got %T", obj).Error(),
		))
		return allErrs
	} else {
		// validate the repo name in the header -> used to select the repo
		err := pkgRev.ValidateRepository()
		if err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec.packageRevID.repository"),
				pkgRev.Spec.PackageRevID.Repository,
				err.Error(),
			))
		} else {
			// validate if the repo exists
			// TBD if this is too much
			repo := configv1alpha1.Repository{}
			err := r.client.Get(ctx, types.NamespacedName{
				Name:      pkgRev.Spec.PackageRevID.Repository,
				Namespace: pkgRev.Namespace,
			}, &repo)
			if err != nil {
				allErrs = append(allErrs, field.Invalid(
					field.NewPath("spec.packageRevID.repository"),
					pkgRev.Spec.PackageRevID.Repository,
					err.Error(),
				))
			}
		}
	}

	switch pkgRev.Spec.Lifecycle {
	case pkgv1alpha1.PackageRevisionLifecyclePublished:
		if err := pkgRev.ValidateDiscoveryAnnotation(); err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("metadata.annotations"),
				pkgRev.Annotations,
				fmt.Sprintf("discovery annotation is not present"),
			))
		}
		if pkgRev.Spec.PackageRevID.Revision == "" {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec.packageRevID.revision"),
				pkgRev.Spec.PackageRevID.Revision,
				fmt.Sprintf("revision cannot be empty when published"),
			))
		}
		/*
			if pkgRev.Spec.Revision != pkgRev.Spec.Workspace {
				allErrs = append(allErrs, field.Invalid(
					field.NewPath("spec.revision"),
					pkgRev.Annotations,
					fmt.Sprintf("revision and workspace mismatch"),
				))
			}
		*/
		if pkgRev.Status.GetCondition(condition.ConditionTypeReady).Status == v1.ConditionFalse {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("status.conditions"),
				pkgRev.Status.GetCondition(condition.ConditionTypeReady),
				fmt.Sprintf("ready condition must be true"),
			))
		}
	case pkgv1alpha1.PackageRevisionLifecycleDraft:
		if pkgRev.Spec.PackageRevID.Revision != "" {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec.packageRevID.revision"),
				pkgRev.Spec.PackageRevID.Revision,
				fmt.Sprintf("revision must be empty"),
			))
		}
		if err := pkgv1alpha1.ValidateWorkspaceName(pkgRev.Spec.PackageRevID.Workspace); err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec.packageRevID.workspace"),
				pkgRev.Spec.PackageRevID.Workspace,
				err.Error(),
			))
		}
		// check workspace name is unique
		opts := []client.ListOption{
			client.InNamespace(pkgRev.Namespace),
			/* fiels selectors dont work
			client.MatchingFields{
				"spec.repository": pkgRev.Spec.Repository,
				"spec.package":    pkgRev.Spec.Package,
				"spec.workspace":  pkgRev.Spec.Workspace,
			},
			*/
		}
		existingPkgRevs := pkgv1alpha1.PackageRevisionList{}
		if err := r.client.List(ctx, &existingPkgRevs, opts...); err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec.packageRevID.workspace"),
				pkgRev.Spec.PackageRevID.Workspace,
				err.Error(),
			))
		} else {
			/*
				for _, existingPkgRev := range existingPkgRevs.Items {
					allErrs = append(allErrs, field.Invalid(
						field.NewPath("spec.workspace"),
						pkgRev.Spec.Workspace,
						fmt.Sprintf("duplicate workspace: %s", existingPkgRev.Name),
					))
				}
			*/
			// alternative since field selectors dont work
			for _, existingPkgRev := range existingPkgRevs.Items {
				existingPkgRev := existingPkgRev
				if existingPkgRev.Spec.PackageRevID.Target == pkgRev.Spec.PackageRevID.Target &&
					existingPkgRev.Spec.PackageRevID.Repository == pkgRev.Spec.PackageRevID.Repository &&
					existingPkgRev.Spec.PackageRevID.Realm == pkgRev.Spec.PackageRevID.Realm &&
					existingPkgRev.Spec.PackageRevID.Package == pkgRev.Spec.PackageRevID.Package &&
					existingPkgRev.Spec.PackageRevID.Workspace == pkgRev.Spec.PackageRevID.Workspace {
					log.Info("duplicate workspace", "pkgID", existingPkgRev.Spec.PackageRevID)
					allErrs = append(allErrs, field.Invalid(
						field.NewPath("spec.packageRevID.workspace"),
						pkgRev.Spec.PackageRevID.Workspace,
						fmt.Sprintf("duplicate workspace: %s", existingPkgRev.Name),
					))
				}
			}
		}
		if len(pkgRev.Status.Conditions) != 0 {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("status.conditions"),
				pkgRev.Status.GetCondition(condition.ConditionTypeReady),
				"condition cannot be set",
			))
		}

	default:
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.lifecycle"),
			pkgRev.Spec.PackageRevID.Repository,
			"cannot create a packagerevision with lifecycle other than draft",
		))
	}

	log.Debug("validate packageRevision done")
	return allErrs
}

func (packageRevisionStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (packageRevisionStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (packageRevisionStrategy) Canonicalize(obj runtime.Object) {
}

func (packageRevisionStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	log := log.FromContext(ctx)
	log.Debug("validate packageRevision update")
	var allErrs field.ErrorList
	newPkgRev, ok := obj.(*pkgv1alpha1.PackageRevision)
	if !ok {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath(""),
			obj,
			fmt.Errorf("expected new PackageRevision object, got %T", obj).Error(),
		))
		return allErrs
	}
	oldPkgRev, ok := old.(*pkgv1alpha1.PackageRevision)
	if !ok {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath(""),
			obj,
			fmt.Errorf("expected old PackageRevision object, got %T", obj).Error(),
		))
		return allErrs
	}

	if oldPkgRev.Spec.PackageRevID.Target != newPkgRev.Spec.PackageRevID.Target {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.packageRevID.target"),
			newPkgRev.Spec.PackageRevID.Repository,
			"spec.packageRevID.target is immutable",
		))
	}
	if oldPkgRev.Spec.PackageRevID.Repository != newPkgRev.Spec.PackageRevID.Repository {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.packageRevID.repository"),
			newPkgRev.Spec.PackageRevID.Repository,
			"spec.packageRevID.repository is immutable",
		))
	}
	if oldPkgRev.Spec.PackageRevID.Realm != newPkgRev.Spec.PackageRevID.Realm {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.packageRevID.realm"),
			newPkgRev.Spec.PackageRevID.Realm,
			"spec.packageRevID.realm is immutable",
		))
	}
	if oldPkgRev.Spec.PackageRevID.Package != newPkgRev.Spec.PackageRevID.Package {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.packageRevID.package"),
			newPkgRev.Spec.PackageRevID.Package,
			"spec.packageRevID.package is immutable",
		))
	}
	if oldPkgRev.Spec.PackageRevID.Workspace != newPkgRev.Spec.PackageRevID.Workspace {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.packageRevID.workspace"),
			newPkgRev.Spec.PackageRevID.Workspace,
			"spec.packageRevID.workspace is immutable",
		))
	}
	if newPkgRev.Spec.Lifecycle != pkgv1alpha1.PackageRevisionLifecyclePublished {
		if oldPkgRev.Spec.PackageRevID.Revision != newPkgRev.Spec.PackageRevID.Revision {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec.packageRevID.revision"),
				newPkgRev.Spec.PackageRevID.Revision,
				"spec.packageRevID.revision is immutable",
			))
		}
	}

	/*
		if len(newPkgRev.Spec.Tasks) != 0 {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec.tasks"),
				newPkgRev.Spec.Tasks,
				fmt.Sprint("tasks cannot be set during update"),
			))
		}
	*/
	if err := pkgv1alpha1.ValidateUpdateLifeCycle(newPkgRev, oldPkgRev); err != nil {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.lifecycle"),
			newPkgRev.Spec.Lifecycle,
			err.Error(),
		))
	}
	log.Debug("validate packageRevision done")
	return allErrs
}

func (packageRevisionStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (packageRevisionStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
