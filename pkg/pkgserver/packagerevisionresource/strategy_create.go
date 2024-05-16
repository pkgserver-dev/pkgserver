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
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/watch"
)

func (r *strategy) BeginCreate(ctx context.Context) error {
	log := log.FromContext(ctx)
	log.Info("begincreate")
	return nil
	//return apierrors.NewMethodNotSupported(pkgv1alpha1.Resource(pkgv1alpha1.PackageRevisionResourcesPlural), "create")
}

func (r *strategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (r *strategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (r *strategy) Create(ctx context.Context, key types.NamespacedName, obj runtime.Object, dryrun bool) (runtime.Object, error) {
	log := log.FromContext(ctx)
	log.Info("create")

	/*
		obj, err := r.Get(ctx, key)
		if err != nil {
			return obj, err
		}
	*/

	pkgRevResources, ok := obj.(*pkgv1alpha1.PackageRevisionResources)
	if !ok {
		log.Error("pkgRevResources create unexpected object", "obj", obj)
		return obj, fmt.Errorf("unexpected object expecting: %s, got: %s", pkgv1alpha1.PackageRevisionResourcesKind, reflect.TypeOf(obj).Name())
	}
	deployment, cachedRepo, err := r.getRepo(ctx, pkgRevResources)
	if err != nil {
		log.Error("pkgRevResources update cannot get repo from cache", "error", err.Error())
		return obj, err
	}

	// for non deployments we dont have to update the git
	// we just want to please the client, with the watch notification
	if deployment {
		pkgRev := pkgv1alpha1.BuildPackageRevision(
			pkgRevResources.ObjectMeta,
			pkgv1alpha1.PackageRevisionSpec{
				PackageRevID: *pkgRevResources.Spec.PackageRevID.DeepCopy(),
			},
			pkgv1alpha1.PackageRevisionStatus{},
		)

		if err := cachedRepo.UpsertPackageRevision(ctx, pkgRev, pkgRevResources.Spec.Resources); err != nil {
			log.Error("pkgRevResources update cannot update resources", "error", err.Error())
			return obj, err
		}
	}
	obj, err = r.Get(ctx, key)
	if err != nil {
		return obj, err
	}
	log.Info("prr resource added", "name", pkgRevResources.Name, "resources", len(pkgRevResources.Spec.Resources))
	r.notifyWatcher(ctx, watch.Event{
		Type:   watch.Added,
		Object: obj,
	})
	return obj, nil
	//return obj, apierrors.NewMethodNotSupported(pkgv1alpha1.Resource(pkgv1alpha1.PackageRevisionResourcesPlural), "create")
}

func (r *strategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}
