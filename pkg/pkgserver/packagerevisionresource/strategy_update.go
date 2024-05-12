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
	"context"
	"fmt"
	"reflect"

	"github.com/henderiw/logger/log"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/pkg/cache"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/watch"
)

func (r *strategy) BeginUpdate(ctx context.Context) error {
	return nil
}

func (r *strategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (r *strategy) AllowCreateOnUpdate() bool { return false }

func (r *strategy) AllowUnconditionalUpdate() bool { return false }

func (r *strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	log := log.FromContext(ctx)
	log.Info("validate packageRevisionResources update")
	var allErrs field.ErrorList
	newPkgRevResources, ok := obj.(*pkgv1alpha1.PackageRevisionResources)
	if !ok {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath(""),
			obj,
			fmt.Errorf("expected new pkgRevResources object, got %T", obj).Error(),
		))
		return allErrs
	}
	oldPkgRevResources, ok := old.(*pkgv1alpha1.PackageRevisionResources)
	if !ok {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath(""),
			obj,
			fmt.Errorf("expected old pkgRevResources object, got %T", obj).Error(),
		))
		return allErrs
	}

	if oldPkgRevResources.Spec.PackageID.Target != newPkgRevResources.Spec.PackageID.Target {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.packageID.target"),
			newPkgRevResources.Spec.PackageID.Target,
			fmt.Sprint("spec.packageID.target is immutable"),
		))
	}
	if oldPkgRevResources.Spec.PackageID.Repository != newPkgRevResources.Spec.PackageID.Repository {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.packageID.repository"),
			newPkgRevResources.Spec.PackageID.Repository,
			fmt.Sprint("spec.packageID.repository is immutable"),
		))
	}
	if oldPkgRevResources.Spec.PackageID.Package != newPkgRevResources.Spec.PackageID.Package {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.packageID.realm"),
			newPkgRevResources.Spec.PackageID.Realm,
			fmt.Sprint("spec.packageID.realm is immutable"),
		))
	}
	if oldPkgRevResources.Spec.PackageID.Package != newPkgRevResources.Spec.PackageID.Package {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.packageID.package"),
			newPkgRevResources.Spec.PackageID.Package,
			fmt.Sprint("spec.packageID.package is immutable"),
		))
	}
	if oldPkgRevResources.Spec.PackageID.Workspace != newPkgRevResources.Spec.PackageID.Workspace {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec.workspace"),
			newPkgRevResources.Spec.PackageID.Workspace,
			fmt.Sprint("spec.workspace is immutable"),
		))
	}

	log.Debug("validate packageRevisionResources done")
	return allErrs
}

func (r *strategy) Update(ctx context.Context, key types.NamespacedName, obj, old runtime.Object, dryrun bool) (runtime.Object, error) {
	// TODO
	//		obj, create, err := r.updatePkgRevResources(ctx, name, objInfo, createValidation, updateValidation, forceAllowCreate, options)
	//		if err != nil {
	//			return obj, create, err
	//		}
	//	return obj, create, nil
	/*
		obj, err := r.Get(ctx, key)
		if err != nil {
			return obj, err
		}
	*/
	log := log.FromContext(ctx)
	pkgRevResources, ok := obj.(*pkgv1alpha1.PackageRevisionResources)
	if !ok {
		log.Error("pkgRevResources update unexpected object", "obj", obj)
		return obj, fmt.Errorf("unexpected object expecting: %s, got: %s", pkgv1alpha1.PackageRevisionResourcesKind, reflect.TypeOf(obj).Name())
	}
	_, cachedRepo, err := r.getRepo(ctx, pkgRevResources)
	if err != nil {
		log.Error("pkgRevResources update cannot get repo from cache", "error", err.Error())
		return obj, err
	}

	pkgRev := pkgv1alpha1.BuildPackageRevision(
		pkgRevResources.ObjectMeta,
		pkgv1alpha1.PackageRevisionSpec{
			PackageID: *pkgRevResources.Spec.PackageID.DeepCopy(),
		},
		pkgv1alpha1.PackageRevisionStatus{},
	)

	if err := cachedRepo.UpsertPackageRevision(ctx, pkgRev, pkgRevResources.Spec.Resources); err != nil {
		log.Error("pkgRevResources update cannot update resources", "error", err.Error())
		return obj, err
	}

	r.notifyWatcher(ctx, watch.Event{
		Type:   watch.Modified,
		Object: obj,
	})
	return obj, nil
}

func (r *strategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

/*
	cachedRepo, err := r.getRepo(ctx, cr)
	if err != nil {
		//log.Error("cannot approve packagerevision", "error", err)
		cr.SetConditions(condition.Failed(err.Error()))
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"Error", "error %s", err.Error())
		return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}


	if err := cachedRepo.UpsertPackageRevision(ctx, cr, resources); err != nil {
		//log.Error("cannot update packagerevision", "error", err)
		cr.SetConditions(condition.Failed(err.Error()))
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"Error", "error %s", err.Error())
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

*/

func (r *strategy) getRepo(ctx context.Context, cr *pkgv1alpha1.PackageRevisionResources) (bool, *cache.CachedRepository, error) {
	log := log.FromContext(ctx)
	repokey := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Spec.PackageID.Repository}

	repo := &configv1alpha1.Repository{}
	if err := r.client.Get(ctx, repokey, repo); err != nil {
		log.Error("cannot get repo", "error", err)
		return false, nil, err
	}

	cachedRepo, err := r.cache.Open(ctx, repo)
	return repo.Spec.Deployment, cachedRepo, err
}
