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
	"fmt"
	"path/filepath"
	"time"

	"github.com/henderiw/logger/log"
	"github.com/henderiw/store"
	"github.com/henderiw/store/memory"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/pkg/auth"
	"github.com/pkgserver-dev/pkgserver/pkg/git/pkg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var tracer = otel.Tracer("cache")

type Cache struct {
	cacheDir           string
	syncFreq           time.Duration
	client             client.Client
	repoStore          store.Storer[*CachedRepository]
	credentialResolver auth.CredentialResolver
	userInfoProvider   auth.UserInfoProvider
}

type Options struct {
	Client             client.Client
	CredentialResolver auth.CredentialResolver
	UserInfoProvider   auth.UserInfoProvider
}

func NewCache(cacheDir string, syncFreq time.Duration, opts Options) *Cache {
	return &Cache{
		cacheDir:           cacheDir,
		syncFreq:           syncFreq,
		repoStore:          memory.NewStore[*CachedRepository](),
		client:             opts.Client,
		credentialResolver: opts.CredentialResolver,
		userInfoProvider:   opts.UserInfoProvider,
	}
}

func (r *Cache) Open(ctx context.Context, cr *configv1alpha1.Repository) (*CachedRepository, error) {
	log := log.FromContext(ctx)
	ctx, span := tracer.Start(ctx, "Cache::OpenRepository", trace.WithAttributes())
	defer span.End()

	// validate the CR content
	if err := cr.Validate(); err != nil {
		return nil, err
	}

	// key is a specific key for git and or oci
	repoKey := store.ToKey(cr.GetKey())

	var exists bool
	cachedRepo, err := r.repoStore.Get(ctx, repoKey)
	if err == nil {
		exists = true
	}

	switch repositoryType := cr.Spec.Type; repositoryType {
	case configv1alpha1.RepositoryTypeOCI:
		// TODO implement OCI
		return nil, fmt.Errorf("oci type not implemented")
	case configv1alpha1.RepositoryTypeGit:
		if !exists {
			// repo was not stored -> open the git repository
			log.Info("cache dir", "dir", filepath.Join(r.cacheDir, "git"))
			gitRepo, err := pkg.OpenRepository(
				ctx,
				filepath.Join(r.cacheDir, "git"),
				cr,
				&pkg.Options{
					CredentialResolver: r.credentialResolver,
					UserInfoProvider:   r.userInfoProvider,
				},
			)
			if err != nil {
				return nil, err
			}
			log.Info("new repository", "gitRepo", gitRepo)
			cachedRepo = newCachedRepository(ctx, r.client, cr, gitRepo, r.syncFreq)
			if err := r.repoStore.Create(ctx, repoKey, cachedRepo); err != nil {
				return nil, err
			}
			return cachedRepo, nil
		} else {
			// repo was stored -> check if the poller retrieved an error or not
			if err := cachedRepo.getRefreshError(); err != nil {
				return nil, err
			}
		}
		return cachedRepo, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", repositoryType)
	}
}

func (r *Cache) Close(ctx context.Context, cr *configv1alpha1.Repository) error {
	ctx, span := tracer.Start(ctx, "Cache::CloseRepository", trace.WithAttributes())
	defer span.End()

	if err := cr.Validate(); err != nil {
		return err
	}

	if err  := pkg.Close(ctx, filepath.Join(r.cacheDir, "git"), cr); err != nil {
		return err
	}

	key := store.ToKey(cr.GetKey())
	cachedRepo, err := r.repoStore.Get(ctx, key)
	if err != nil {
		return nil // does not exist
	}
	if cachedRepo != nil {
		if err := cachedRepo.Close(); err != nil {
			return err
		}
	}
	return nil
}
