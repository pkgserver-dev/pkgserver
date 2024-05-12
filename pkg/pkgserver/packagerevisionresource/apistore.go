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

	builderrest "github.com/henderiw/apiserver-builder/pkg/builder/rest"
	"github.com/henderiw/apiserver-store/pkg/generic/registry"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/pkg/cache"
	"go.opentelemetry.io/otel"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewProvider(ctx context.Context, client client.Client, cache *cache.Cache) builderrest.ResourceHandlerProvider {
	return func(ctx context.Context, scheme *runtime.Scheme, getter generic.RESTOptionsGetter) (rest.Storage, error) {
		return NewREST(ctx, scheme, getter, client, cache)
	}
}

// NewPackageRevisionREST returns a RESTStorage object that will work against API services.
func NewREST(ctx context.Context, scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter, client client.Client, cache *cache.Cache) (rest.Storage, error) {
	scheme.AddFieldLabelConversionFunc(
		schema.GroupVersionKind{
			Group:   pkgv1alpha1.Group,
			Version: pkgv1alpha1.Version,
			Kind:    pkgv1alpha1.PackageRevisionResourcesKind,
		},
		pkgv1alpha1.ConvertPackageRevisionResourcesFieldSelector,
	)

	strategy := NewStrategy(ctx, scheme, client, cache)

	// This is the etcd store
	store := &registry.Store{
		Tracer:                    otel.Tracer("pkg-server"),
		NewFunc:                   func() runtime.Object { return &pkgv1alpha1.PackageRevisionResources{} },
		NewListFunc:               func() runtime.Object { return &pkgv1alpha1.PackageRevisionResourcesList{} },
		PredicateFunc:             Match,
		DefaultQualifiedResource:  pkgv1alpha1.Resource(pkgv1alpha1.PackageRevisionResourcesPlural),
		SingularQualifiedResource: pkgv1alpha1.Resource(pkgv1alpha1.PackageRevisionResourcesSingular),
		GetStrategy:               strategy,
		ListStrategy:              strategy,
		CreateStrategy:            strategy,
		UpdateStrategy:            strategy,
		DeleteStrategy:            strategy,
		WatchStrategy:             strategy,

		TableConvertor: NewTableConvertor(pkgv1alpha1.Resource(pkgv1alpha1.PackageRevisionResourcesPlural)),
	}
	options := &generic.StoreOptions{
		RESTOptions: optsGetter,
		AttrFunc:    GetAttrs,
	}

	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return store, nil
}
