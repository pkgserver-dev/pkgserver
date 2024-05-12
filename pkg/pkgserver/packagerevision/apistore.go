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

	builderrest "github.com/henderiw/apiserver-builder/pkg/builder/rest"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/pkg/cache"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewProvider(ctx context.Context, client client.Client, cache *cache.Cache) builderrest.ResourceHandlerProvider {
	return func(ctx context.Context, scheme *runtime.Scheme, getter generic.RESTOptionsGetter) (rest.Storage, error) {
		return NewREST(scheme, getter, client, cache)
	}
}

// NewPackageRevisionREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter, client client.Client, cache *cache.Cache) (rest.Storage, error) {
	scheme.AddFieldLabelConversionFunc(
		schema.GroupVersionKind{
			Group:   pkgv1alpha1.Group,
			Version: pkgv1alpha1.Version,
			Kind:    pkgv1alpha1.PackageRevisionKind,
		},
		pkgv1alpha1.ConvertPackageRevisionsFieldSelector,
	)

	strategy := NewStrategy(scheme, client)

	// This is the etcd store
	store := &genericregistry.Store{
		NewFunc:                   func() runtime.Object { return &pkgv1alpha1.PackageRevision{} },
		NewListFunc:               func() runtime.Object { return &pkgv1alpha1.PackageRevisionList{} },
		PredicateFunc:             MatchPackageRevision,
		DefaultQualifiedResource:  pkgv1alpha1.Resource(pkgv1alpha1.PackageRevisionPlural),
		SingularQualifiedResource: pkgv1alpha1.Resource(pkgv1alpha1.PackageRevisionSingular),
		CreateStrategy:            strategy,
		UpdateStrategy:            strategy,
		DeleteStrategy:            strategy,

		TableConvertor: NewTableConvertor(pkgv1alpha1.Resource(pkgv1alpha1.PackageRevisionPlural)),
	}
	options := &generic.StoreOptions{
		RESTOptions: optsGetter,
		AttrFunc:    GetAttrs,
		Indexers:    Indexers(),
	}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return store, nil
}
