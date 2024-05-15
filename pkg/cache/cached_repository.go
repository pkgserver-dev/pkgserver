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

package cache

import (
	"context"
	"time"

	"github.com/henderiw/logger/log"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/pkg/repository"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CachedRepository struct {
	client client.Client
	cr     *configv1alpha1.Repository
	repo   repository.Repository // interface to access git resources
	// Error encountered on repository refresh by the refresh goroutine.
	// Reported back by the repository reconciler
	refreshError error

	cancel context.CancelFunc
}

func newCachedRepository(ctx context.Context, client client.Client, cr *configv1alpha1.Repository, repo repository.Repository, syncFreq time.Duration) *CachedRepository {
	ctx, cancel := context.WithCancel(ctx)

	r := &CachedRepository{
		client: client,
		cr:     cr,
		repo:   repo,
		cancel: cancel,
	}

	// only provide the repo scan for repositories which do not have deployment enabled
	if !cr.Spec.Deployment {
		go r.repoScanner(ctx, syncFreq)
	}

	return r
}

func (r *CachedRepository) getRefreshError() error {
	return r.refreshError
}

func (r *CachedRepository) Close() error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

// repoScanner will continue scanning the repository until signal channel is closed or ctx is done.
func (r *CachedRepository) repoScanner(ctx context.Context, repoSyncFrequency time.Duration) {
	log := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			log.Debug("stop repo scanner", "name", r.cr.Name, "error", ctx.Err())
			return
		default:
			if err := r.reconcileCache(ctx); err != nil {
				log.Error("reconcile repo packages failed", "name", r.cr.Name, "error", err.Error())
			}
			time.Sleep(repoSyncFrequency)
		}
	}
}

func (r *CachedRepository) reconcileCache(ctx context.Context) error {
	log := log.FromContext(ctx).With("repoName", r.cr.Name)
	ctx, span := tracer.Start(ctx, "Repository::reconcileCache", trace.WithAttributes())
	defer span.End()

	start := time.Now()
	defer func() {
		log.Debug("reconcile repo", "finished", time.Since(start).Seconds())
	}()

	discoveredPkgRevs, err := r.repo.ListPackageRevisions(ctx, nil)
	if err != nil {
		return err
	}
	for _, discoveredPkgRev := range discoveredPkgRevs {
		log.Info("discovered package revision", "name", discoveredPkgRev)

		key := types.NamespacedName{
			Namespace: discoveredPkgRev.Namespace,
			Name:      discoveredPkgRev.Name,
		}

		if err := r.client.Get(ctx, key, &pkgv1alpha1.PackageRevision{}); err != nil {
			if err := r.client.Create(ctx, discoveredPkgRev); err != nil {
				log.Error("cannot create package revision", "key", key.String(), "error", err)
				continue
			}
		}
	}

	return nil
}

func (r *CachedRepository) GetResources(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) (map[string]string, error) {
	ctx, span := tracer.Start(ctx, "cache::GetResources", trace.WithAttributes())
	defer span.End()

	return r.repo.GetResources(ctx, pkgRev)
}

func (r *CachedRepository) UpsertPackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, resources map[string]string) error {
	ctx, span := tracer.Start(ctx, "cache::UpsertPackageRevision", trace.WithAttributes())
	defer span.End()

	return r.repo.UpsertPackageRevision(ctx, pkgRev, resources)
}

func (r *CachedRepository) DeletePackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) error {
	ctx, span := tracer.Start(ctx, "cache::DeletePackageRevision", trace.WithAttributes())
	defer span.End()

	return r.repo.DeletePackageRevision(ctx, pkgRev)
}

func (r *CachedRepository) ListPackageRevisions(ctx context.Context, opts *repository.ListOption) ([]*pkgv1alpha1.PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "cache::ListPackageRevisions", trace.WithAttributes())
	defer span.End()

	return r.repo.ListPackageRevisions(ctx, opts)
}

func (r *CachedRepository) EnsurePackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) error {
	ctx, span := tracer.Start(ctx, "cache::ListPackageRevisions", trace.WithAttributes())
	defer span.End()

	return r.repo.EnsurePackageRevision(ctx, pkgRev)
}