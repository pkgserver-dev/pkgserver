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

	"github.com/henderiw/apiserver-store/pkg/rest"
	"github.com/henderiw/logger/log"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	repocache "github.com/pkgserver-dev/pkgserver/pkg/cache"
	watchermanager "github.com/pkgserver-dev/pkgserver/pkg/watcher-manager"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewStrategy creates and returns a fischerStrategy instance
func NewStrategy(ctx context.Context, typer runtime.ObjectTyper, client client.Client, cache *repocache.Cache) *strategy {
	watcherManager := watchermanager.New(32)

	go watcherManager.Start(ctx)

	return &strategy{
		ObjectTyper:    typer,
		NameGenerator:  names.SimpleNameGenerator,
		client:         client,
		cache:          cache,
		watcherManager: watcherManager,
	}
}

// Match is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func Match(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// GetAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not a PackageRevisionResources
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	api, ok := obj.(*pkgv1alpha1.PackageRevisionResources)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a PackageRevisionResource")
	}
	return labels.Set(api.ObjectMeta.Labels), SelectableFields(api), nil
}

// SelectableFields returns a field set that represents the object.
func SelectableFields(obj *pkgv1alpha1.PackageRevisionResources) fields.Set {
	return fields.Set{
		"metadata.namespace":       obj.Namespace,
		"metadata.name":            obj.Name,
		"spec.packgeID.repository": obj.Spec.PackageID.Repository,
		"spec.packgeID.package":    obj.Spec.PackageID.Package,
		"spec.packgeID.revision":   obj.Spec.PackageID.Revision,
		"spec.packgeID.workspace":  obj.Spec.PackageID.Workspace,
	}
	//fieldSet := generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

func Indexers() *cache.Indexers {
	return &cache.Indexers{
		"f:spec.repository": func(obj interface{}) ([]string, error) {
			pkgRevRes, ok := obj.(*pkgv1alpha1.PackageRevisionResources)
			if !ok {
				return []string{}, nil
			}
			return []string{pkgRevRes.Spec.PackageID.Repository}, nil
		},
		"f:spec.package": func(obj interface{}) ([]string, error) {
			pkgRevRes, ok := obj.(*pkgv1alpha1.PackageRevisionResources)
			if !ok {
				return []string{}, nil
			}
			return []string{pkgRevRes.Spec.PackageID.Package}, nil
		},
		"f:spec.revision": func(obj interface{}) ([]string, error) {
			pkgRevRes, ok := obj.(*pkgv1alpha1.PackageRevisionResources)
			if !ok {
				return []string{}, nil
			}
			return []string{pkgRevRes.Spec.PackageID.Revision}, nil
		},
		"f:spec.workspace": func(obj interface{}) ([]string, error) {
			pkgRevRes, ok := obj.(*pkgv1alpha1.PackageRevisionResources)
			if !ok {
				return []string{}, nil
			}
			return []string{pkgRevRes.Spec.PackageID.Revision}, nil
		},
	}
}

var _ rest.RESTCreateStrategy = &strategy{}
var _ rest.RESTUpdateStrategy = &strategy{}
var _ rest.RESTDeleteStrategy = &strategy{}
var _ rest.RESTGetStrategy = &strategy{}
var _ rest.RESTListStrategy = &strategy{}
var _ rest.RESTWatchStrategy = &strategy{}

type strategy struct {
	runtime.ObjectTyper
	names.NameGenerator
	client         client.Client
	cache          *repocache.Cache
	watcherManager watchermanager.WatcherManager
}

func (r *strategy) NamespaceScoped() bool {
	return true
}

func (r *strategy) Canonicalize(obj runtime.Object) {}

func (r *strategy) notifyWatcher(ctx context.Context, event watch.Event) {
	log := log.FromContext(ctx).With("eventType", event.Type)
	log.Info("notify watcherManager")

	r.watcherManager.WatchChan() <- event
}

func (r *strategy) getRepository(ctx context.Context, nsn types.NamespacedName) (*configv1alpha1.Repository, error) {
	repo := configv1alpha1.Repository{}
	if err := r.client.Get(ctx, nsn, &repo); err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	return &repo, nil
}

func (r *strategy) getPackageRevision(ctx context.Context, nsn types.NamespacedName) (*pkgv1alpha1.PackageRevision, error) {
	pkgRev := pkgv1alpha1.PackageRevision{}
	if err := r.client.Get(ctx, nsn, &pkgRev); err != nil {
		return nil, apierrors.NewNotFound(pkgv1alpha1.Resource(pkgv1alpha1.PackageRevisionResourcesPlural), nsn.Name)
	}
	return &pkgRev, nil
}

func buildPackageRevisionResources(pkgRev *pkgv1alpha1.PackageRevision, resources map[string]string) *pkgv1alpha1.PackageRevisionResources {
	return pkgv1alpha1.BuildPackageRevisionResources(
		*pkgRev.ObjectMeta.DeepCopy(),
		pkgv1alpha1.PackageRevisionResourcesSpec{
			PackageID: *pkgRev.Spec.PackageID.DeepCopy(),
			Resources: resources,
		},
		pkgv1alpha1.PackageRevisionResourcesStatus{},
	)
}
