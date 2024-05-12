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
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/henderiw/logger/log"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// packageList holds a list of packages in the git repository
type packageList struct {
	// parent is the gitRepository of which this is part
	parent *gitRepository

	// commit is the commit at which we scanned for packages
	commit *object.Commit

	// packages holds the packages we found
	packages map[string]*packageListEntry
}

// packageListEntry is a single package found in a git repository
type packageListEntry struct {
	// parent is the packageList of which we are part
	parent *packageList

	// path is the relative path to the root of the package
	path string

	// treeHash is the git-hash of the git tree corresponding to Path
	treeHash plumbing.Hash
}

// findPackage finds the packages in the git repository, under commit, if it is exists at path.
// If no package is found at that path, returns nil, nil
func (r *gitRepository) findPackage(ctx context.Context, commit *object.Commit, packagePath string) (*packageListEntry, error) {
	t, err := r.discoverPackagesInTree(
		ctx,
		commit,
		DiscoverPackagesOptions{
			PackagePath: packagePath,
			Recurse:     false,
		},
	)
	if err != nil {
		return nil, err
	}
	return t.packages[packagePath], nil
}

// DiscoveryPackagesOptions holds the configuration for walking a git tree
type DiscoverPackagesOptions struct {
	// PackagePath restricts package discovery to a particular subdirectory, which is the
	// PackagePath or PackageName or BranchName
	// The subdirectory is not required to exist (we will return an empty package).
	PackagePath string

	// Recurse enables recursive traversal of the git tree.
	Recurse bool
}

// discoverPackagesInTree finds the packages in the git repository, under commit.
// If PackagePath is non-empty, only packages with the specified packagePath will be returned.
// It is not an error if PackagePath matches no packages or even is not a real directory name;
// we will simply return an empty list of packages.
func (r *gitRepository) discoverPackagesInTree(ctx context.Context, commit *object.Commit, opt DiscoverPackagesOptions) (*packageList, error) {
	log := log.FromContext(ctx)
	pkgList := &packageList{
		parent:   r,
		commit:   commit,
		packages: make(map[string]*packageListEntry),
	}
	rootTree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve commit %v to tree (corrupted repository?): %w", commit.Hash, err)
	}
	if opt.PackagePath != "" {
		tree, err := rootTree.Tree(opt.PackagePath)
		if err != nil {
			if err == object.ErrDirectoryNotFound {
				// We treat the filter prefix as a filter, the path doesn't have to exist
				log.Info("could not find packageName/packagePath in commit; returning no packages", "packagePath", opt.PackagePath, "commit", commit.Hash.String())
				return pkgList, nil
			} else {
				return nil, fmt.Errorf("error getting tree %s: %w", opt.PackagePath, err)
			}
		}
		rootTree = tree
	}

	if err := pkgList.discoverPackages(ctx, rootTree, "", opt); err != nil {
		return nil, err
	}
	return pkgList, nil
}

// discoverPackages is the recursive function we use to traverse the tree and find packages.
// tree is the git-tree we are search, treePath is the repo-relative-path to tree.
func (r *packageList) discoverPackages(ctx context.Context, tree *object.Tree, curPath string, opt DiscoverPackagesOptions) error {
	log := log.FromContext(ctx)
	log.Debug("discoverPackages", "packagePath", opt.PackagePath, "curPath", curPath)

	// TODO: do we need specific logic to check if a package is a true package
	// e.g. check a specific file, etc
	r.packages[opt.PackagePath] = &packageListEntry{
		path:     opt.PackagePath,
		treeHash: tree.Hash,
		parent:   r,
	}
	// a package cannot have recursive entries
	return nil
}

// buildGitDiscoveredPackageRevision creates a gitPackageRevision for the packageListEntry
func (r *packageListEntry) buildPackageRevision(ctx context.Context, deployment bool, revision, ws string, commit *object.Commit) *pkgv1alpha1.PackageRevision {
	log := log.FromContext(ctx)
	log.Debug("buildPackageRevision", "deployment", deployment, "revision", revision, "workspace", ws, "packagepach", r.path)
	repo := r.parent.parent

	annotations := map[string]string{}
	var pkgID pkgid.PackageID
	if !deployment {
		pkgID = pkgid.PackageID{
			Target:     pkgid.PkgTarget_Catalog,
			Repository: repo.cr.Name,
			Realm:      filepath.Dir(r.path),
			Package:    filepath.Base(r.path),
			Revision:   revision,
			Workspace:  ws,
		}
		annotations[pkgv1alpha1.DiscoveredPkgRevKey] = "true"
	} else {
		parts := strings.Split(r.path, "/")
		if len(parts) < 2 {
			return nil
		}
		pkgID = pkgid.PackageID{
			Target:     parts[0],
			Repository: repo.cr.Name,
			Realm:      filepath.Join(parts[1 : len(parts)-2]...),
			Package:    parts[len(parts)-1],
			Revision:   revision,
			Workspace:  ws,
		}
	}

	pkgrev := &pkgv1alpha1.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			APIVersion: pkgv1alpha1.SchemeGroupVersion.Identifier(),
			Kind:       pkgv1alpha1.PackageRevisionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   repo.cr.Namespace,
			Name:        pkgID.PkgRevString(),
			Annotations: annotations,
		},
		Spec: pkgv1alpha1.PackageRevisionSpec{
			PackageID: pkgID,
			Lifecycle: pkgv1alpha1.PackageRevisionLifecyclePublished,
		},
		Status: pkgv1alpha1.PackageRevisionStatus{
			PublishedBy: commit.Author.Name,
			PublishedAt: metav1.Time{Time: commit.Author.When},
		},
	}
	pkgrev.SetConditions(condition.Ready())
	return pkgrev
}
