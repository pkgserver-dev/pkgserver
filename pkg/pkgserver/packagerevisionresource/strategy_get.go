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

	"github.com/henderiw/logger/log"
	"github.com/pkgserver-dev/pkgserver/apis/pkgrevid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func (r *strategy) Get(ctx context.Context, key types.NamespacedName) (runtime.Object, error) {
	log := log.FromContext(ctx)
	log.Debug("get...", "key", key)
	pkgRev, err := r.getPackageRevision(ctx, key)
	if err != nil {
		log.Error("pkgRevResources cannot get pkgRev in apiserver", "key", key.String(), "error", err.Error())
		return nil, apierrors.NewInternalError(err)
	}

	repo, err := r.getRepository(ctx, types.NamespacedName{
		Name:      pkgrevid.GetRepoNameFromPkgRevName(key.Name),
		Namespace: key.Namespace})
	if err != nil {
		log.Error("pkgRevResources cannot get repository in apiserver", "repo", pkgrevid.GetRepoNameFromPkgRevName(key.Name), "error", err.Error())
		return nil, apierrors.NewInternalError(err)
	}

	cachedRepo, err := r.cache.Open(ctx, repo)
	if err != nil {
		log.Error("pkgRevResources cannot get cached repo", "repo", repo, "error", err.Error())
		return nil, apierrors.NewInternalError(err)
	}
	log.Debug("get resources", "key", pkgRev.Name)
	resources, err := cachedRepo.GetResources(ctx, pkgRev)
	if err != nil {
		log.Error("pkgRevResources cannot get resource in apiserver", "pkgRev", pkgRev, "error", err.Error())
		return nil, apierrors.NewInternalError(err)
	}
	log.Debug("build resources", "key", pkgRev.Name)
	pkgRevResources := buildPackageRevisionResources(pkgRev, resources)
	//log.Info("pkgRevResources", "value", pkgRevResources)
	return pkgRevResources, nil
}
