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

package pkg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/henderiw/logger/log"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/pkg/auth"
	pkggit "github.com/pkgserver-dev/pkgserver/pkg/git"
	"github.com/pkgserver-dev/pkgserver/pkg/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	LocalGitDirectory = ".git"
)

var tracer = otel.Tracer("gitpkg")

type gitRepository struct {
	cr     *configv1alpha1.Repository
	repo   *pkggit.Repository
	branch BranchName

	mu sync.RWMutex
}

type Options struct {
	CredentialResolver auth.CredentialResolver
	UserInfoProvider   auth.UserInfoProvider
}

func OpenRepository(ctx context.Context, cacheDir string, cr *configv1alpha1.Repository, opts *Options) (repository.Repository, error) {
	log := log.FromContext(ctx)
	ctx, span := tracer.Start(ctx, "OpenRepository", trace.WithAttributes())
	defer span.End()

	// there are 2 secenario's:
	// 1. client tool -> we need to open a single repo: we use <cacheDir>/.git
	// 2. porch server -> we need to open multiple repos: we use <cacheDir>/<repo-url with modifications>
	dir := cacheDir
	if filepath.Base(cacheDir) != LocalGitDirectory {
		replace := strings.NewReplacer("/", "-", ":", "-")
		dir = filepath.Join(cacheDir, replace.Replace(cr.GetURL()))
	}
	log.Debug("open repo", "dir", dir)

	// Cleanup the directory in case initialization fails.
	cleanup := dir
	defer func() {
		if cleanup != "" {
			os.RemoveAll(cleanup)
		}
	}()

	repo := &pkggit.Repository{
		Name:                     cr.Name,
		Namespace:                cr.Namespace,
		OriginName:               OriginName,
		DefaultMainReferenceName: DefaultMainReferenceName,
		DefaultFetchSpec:         defaultFetchSpec,
		URL:                      cr.GetURL(),
		Directory:                dir,
		CredentialSecret:         cr.GetCredentialSecret(),
		CredentialResolver:       opts.CredentialResolver,
		UserInfoProvider:         opts.UserInfoProvider,
	}

	// check if the directory exists (<init-dir>/<git>/<repo-url w/ replaced / and :>)
	if fi, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		r, err := repo.InitEmptyRepository()
		if err != nil {
			return nil, fmt.Errorf("error cloning git repository %q: %w", repo.URL, err)
		}

		repo.Repo = r

	} else if !fi.IsDir() {
		// file exists but is not a directory -> corruption
		return nil, fmt.Errorf("cannot clone git repository %q: %w", repo.URL, err)
	} else {
		// directory exists
		cleanup = "" // do no cleanup
		r, err := pkggit.OpenGitRepository(dir)
		if err != nil {
			return nil, err
		}
		repo.Repo = r
	}
	// Create Remote
	if err := repo.InitializeOrigin(repo.Repo); err != nil {
		return nil, fmt.Errorf("error cloning git repository %q, cannot create remote: %v", cr.GetURL(), err)
	}

	branch := MainBranch
	if cr.GetRef() != "" {
		branch = BranchName(cr.GetRef())
	}

	gitrepo := &gitRepository{
		cr:     cr.DeepCopy(),
		repo:   repo,
		branch: branch,
	}

	if err := repo.FetchRemoteRepository(ctx); err != nil {
		return nil, err
	}

	// validate if the branch exists
	if _, err := repo.Repo.Reference(branch.BranchInLocal(), false); err != nil {
		return nil, err
	}

	cleanup = "" // success we are good to go w/o removing the directory

	return gitrepo, nil
}

func Close(ctx context.Context, cacheDir string, cr *configv1alpha1.Repository) error {
	replace := strings.NewReplacer("/", "-", ":", "-")
	dir := filepath.Join(cacheDir, replace.Replace(cr.GetURL()))
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("error cleaning up local git cache for repo %s: %v", cr.Name, err)
	}
	return nil
}
