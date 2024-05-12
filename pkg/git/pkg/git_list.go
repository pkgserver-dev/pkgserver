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
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/henderiw/logger/log"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/pkg/repository"
	"go.opentelemetry.io/otel/trace"
)

func (r *gitRepository) ListPackageRevisions(ctx context.Context, opt *repository.ListOption) ([]*pkgv1alpha1.PackageRevision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.listPackageRevisions(ctx, opt)
}

func (r *gitRepository) listPackageRevisions(ctx context.Context, opt *repository.ListOption) ([]*pkgv1alpha1.PackageRevision, error) {
	log := log.FromContext(ctx)

	if err := r.repo.FetchRemoteRepository(ctx); err != nil {
		return nil, err
	}

	pkgRevs := []*pkgv1alpha1.PackageRevision{}
	if err := r.repo.ListRefs(ctx, func(ctx context.Context, ref *plumbing.Reference) error {
		switch refName := ref.Name(); {
		case isTagInLocalRepo(refName):
			if !IsFiltered(refName, opt) {
				pkgRev := r.getTaggedPackage(ctx, ref)
				if pkgRev == nil {
					log.Info("ignore tagged pkg nil", "ref", ref)
				} else {
					pkgRevs = append(pkgRevs, pkgRev)
				}
			}
		default:
			// do nothing as we only look at released packages when we list packageRevisions
		}
		return nil
	}); err != nil {
		return pkgRevs, err
	}
	return pkgRevs, nil
}

// packageFilter filters
func IsFiltered(refName plumbing.ReferenceName, opt *repository.ListOption) bool {
	if opt == nil {
		return false
	}
	filter := true
	if opt.PackageID != nil {
		if strings.Contains(refName.String(), opt.PackageID.Path()) {
			filter = false
		} else {
			filter = true
		}
	}
	return filter
}

func (r *gitRepository) getTaggedPackage(ctx context.Context, tagRef *plumbing.Reference) *pkgv1alpha1.PackageRevision {
	log := log.FromContext(ctx).With("tagRef", tagRef.String())
	ctx, span := tracer.Start(ctx, "gitRepository::loadTaggedPackage", trace.WithAttributes())
	defer span.End()

	name, ok := getTagNameInLocalRepo(tagRef.Name())
	if !ok {
		return nil
	}
	slash := strings.LastIndex(name, "/")
	if slash < 0 {
		// tag=<version>
		// could be a release tag or something else, we ignore these types of tags
		log.Debug("ignored tagged package", "name", name, "tag", tagRef)
		return nil
	}
	// tag=<packagePath=packagename=branchName>/version
	packageName, revision := name[:slash], name[slash+1:]
	if !packageInDirectory(packageName, r.cr.GetDirectory()) {
		log.Debug("ignored package not in directory", "packageName", packageName, "dir", r.cr.GetDirectory())
		return nil
	}
	log.Debug("getTaggedPackage", "packageName", packageName, "revision", revision, "tagRef name", tagRef.String())

	ws, commit, err := r.getBranchAndCommitFromTag(ctx, packageName, tagRef)
	if err != nil {
		log.Error("cannot get branch and commits from tag", "error", err)
		return nil
	}

	krmPackage, err := r.findPackage(ctx, commit, packageName)
	if err != nil {
		log.Error("cannot find package", "error", err.Error())
		return nil
	}
	if krmPackage == nil {
		log.Info("package not found", "name", name)
		return nil
	}
	
	return krmPackage.buildPackageRevision(ctx, r.cr.Spec.Deployment, revision, ws, commit)
}

func (r *gitRepository) getBranchAndCommitFromTag(ctx context.Context, packageName string, tagRef *plumbing.Reference) (string, *object.Commit, error) {
	if annotedTagObject, err := r.repo.Repo.TagObject(tagRef.Hash()); err != plumbing.ErrObjectNotFound {
		if annotedTagObject.TargetType == plumbing.CommitObject {
			return r.getBranchAndCommitFromTagHash(ctx, packageName, annotedTagObject.Target)
		}
		return "", nil, fmt.Errorf("commit not found for ref: %s", tagRef.Name().String())
	}
	return r.getBranchAndCommitFromTagHash(ctx, packageName, tagRef.Hash())
}

func (r *gitRepository) getBranchAndCommitFromTagHash(ctx context.Context, packageName string, hash plumbing.Hash) (string, *object.Commit, error) {
	log := log.FromContext(ctx)
	log.Debug("getBranchAndCommitFromHash")

	refs, err := r.repo.Repo.References()
	if err != nil {
		log.Error("getBranchAndCommitFromHash cannot get references", "err", err.Error())
		return "", nil, err
	}
	var tagCommit *object.Commit
	var ws string
	if err = refs.ForEach(func(ref *plumbing.Reference) error {
		if isBranchInLocalRepo(ref.Name()) {
			if !strings.Contains(ref.Name().String(), packageName) {
				// the tag and branch dont match
				return nil
			}
			// check the commits for this branch
			commits, err := r.repo.Repo.Log(&git.LogOptions{From: ref.Hash()})
			if err != nil {
				return err
			}
			for {
				commit, err := commits.Next()
				if err != nil {
					break
				}
				log.Debug("getBranchAndCommitFromHash does branches match", "tagHash", hash.String(), "commitHash", commit.Hash.String())
				if commit.Hash.String() == hash.String() {
					parts := strings.Split(ref.Name().String(), "/")
					ws = parts[len(parts)-1]
					tagCommit = commit
					return storer.ErrStop // stops the iterator
				}
			}
		}
		return nil
	}); err != nil {
		return "", nil, err
	}
	if tagCommit == nil {
		return "", nil, fmt.Errorf("commit not found")
	}
	return ws, tagCommit, nil
}
