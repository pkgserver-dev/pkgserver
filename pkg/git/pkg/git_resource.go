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
	"io"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/henderiw/logger/log"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"go.opentelemetry.io/otel/trace"
)

func (r *gitRepository) GetResources(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, useWorkspaceBranch bool) (map[string]string, error) {
	ctx, span := tracer.Start(ctx, "gitRepository::GetResources", trace.WithAttributes())
	defer span.End()
	r.mu.RLock()
	defer r.mu.RUnlock()

	// saftey sync with the repo
	if err := r.repo.FetchRemoteRepository(ctx); err != nil {
		return nil, err
	}

	log := log.FromContext(ctx)
	commit, err := r.getCommit(ctx, pkgRev, useWorkspaceBranch)
	if err != nil {
		log.Error("get resources: cannot get commit", "err", err)
		return nil, err
	}
	return getResources(ctx, pkgRev.Spec.PackageID, commit)

}

func getResources(ctx context.Context, pkgID pkgid.PackageID, commit *object.Commit) (map[string]string, error) {
	log := log.FromContext(ctx)
	resources := map[string]string{}
	// get the root tree of the package
	pkgRootTree, err := getPackageTree(ctx, pkgID, commit)
	if err != nil {
		log.Error("cannot get package root tree", "error", err.Error())
		return resources, err
	}
	if pkgRootTree == nil {
		return resources, err
	}
	// Files() iterator iterates recursively over all files in the tree.
	fit := pkgRootTree.Files()
	defer fit.Close()
	for {
		file, err := fit.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to load package resources: %w", err)
		}
		content, err := file.Contents()
		if err != nil {
			return nil, fmt.Errorf("failed to read package file contents: %q, %w", file.Name, err)
		}
		resources[file.Name] = content
	}
	return resources, nil
}

func (r *gitRepository) getCommit(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, useWorkspaceBranch bool) (*object.Commit, error) {
	log := log.FromContext(ctx)
	log.Info("getCommit", "repo", r.cr.Name, "pkgID", pkgRev.Spec.PackageID)
	// when a revision is set there is a tag and we use the tag to lookup the commit reference
	// there is 2 types of tags annotated tags and regular tags, depending on the type we get the
	// commit by different means
	if !useWorkspaceBranch && pkgRev.Spec.PackageID.Revision != "" {
		tagRefName := packageTagRefName(pkgRev.Spec.PackageID, pkgRev.Spec.PackageID.Revision)
		tagRef, err := r.repo.Repo.Reference(tagRefName, true)
		if err != nil {
			log.Error("cannot get commit from published package, tag ref does not exist", "tagRefName", tagRefName, "error", err)
			return nil, err
		}
		if annotedTagObject, err := r.repo.Repo.TagObject(tagRef.Hash()); err != plumbing.ErrObjectNotFound {
			if annotedTagObject.TargetType == plumbing.CommitObject {
				return r.repo.Repo.CommitObject(annotedTagObject.Target)
			}
			return nil, fmt.Errorf("commit not found for ref: %s", tagRefName.String())
		}
		_, commit, err := r.getBranchAndCommitFromTagHash(ctx, pkgRev.Spec.PackageID.Package, tagRef.Hash(), tagRefName.String())
		return commit, err
	} else {
		branchRefName := workspacePackageBranchRefName(pkgRev.Spec.PackageID)
		branchRef, err := r.repo.Repo.Reference(branchRefName, true)
		if err != nil {
			log.Error("cannot get commit from package, branch does not exist", "branchRefName", branchRefName, "error", err)
			return nil, err
		}
		return r.repo.Repo.CommitObject(branchRef.Hash())
	}
}

func getPackageTree(ctx context.Context, pkgID pkgid.PackageID, commit *object.Commit) (*object.Tree, error) {
	log := log.FromContext(ctx)
	rootTree, err := commit.Tree()
	if err != nil {
		log.Error("cannot get root tree", "error", err.Error())
		return nil, fmt.Errorf("cannot resolve commit %v to tree (corrupted repository?): %w", commit.Hash, err)
	}
	tree, err := rootTree.Tree(pkgID.Path())
	if err != nil {
		if err == object.ErrDirectoryNotFound {
			// We treat the filter prefix as a filter, the path doesn't have to exist
			log.Info("could not find prefix in commit; returning no resources", "package", pkgID.Path(), "commit", commit.Hash.String())
			return nil, nil
		} else {
			return nil, fmt.Errorf("error getting tree %s: %w", pkgID.Path(), err)
		}
	}
	return tree, nil
}
