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

package git

import (
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// initEmptyRepository initializes an empty bare repository
func (r *Repository) InitEmptyRepository() (*git.Repository, error) {
	isBare := true
	repo, err := git.PlainInit(r.Directory, isBare)
	if err != nil {
		return nil, err
	}
	if err := r.InitializeDefaultBranches(repo); err != nil {
		return nil, err
	}
	return repo, nil
}

// initializeDefaultBranches
func (r *Repository) InitializeDefaultBranches(repo *git.Repository) error {
	// Adjust Repository references
	if err := repo.Storer.RemoveReference(plumbing.Master); err != nil {
		return err
	}
	// gogit points HEAD at a wrong branch; point it at main
	main := plumbing.NewSymbolicReference(plumbing.HEAD, r.DefaultMainReferenceName)
	if err := repo.Storer.SetReference(main); err != nil {
		return err
	}
	return nil
}

func OpenGitRepository(path string) (*git.Repository, error) {
	dot := osfs.New(path)
	storage := filesystem.NewStorage(dot, cache.NewObjectLRUDefault())
	return git.Open(storage, dot)
}

func (r *Repository) InitializeOrigin(repo *git.Repository) error {
	cfg, err := repo.Config()
	if err != nil {
		return err
	}

	cfg.Remotes[r.OriginName] = &config.RemoteConfig{
		Name:  r.OriginName,
		URLs:  []string{r.URL},
		Fetch: r.DefaultFetchSpec,
	}

	if err := repo.SetConfig(cfg); err != nil {
		return err
	}

	return nil
}
