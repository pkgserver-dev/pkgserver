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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/henderiw/logger/log"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"go.opentelemetry.io/otel/trace"
)

func (r *gitRepository) EnsurePackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) error {
	ctx, span := tracer.Start(ctx, "gitRepository::EnsurePackageRevision", trace.WithAttributes())
	defer span.End()
	r.m.Lock()
	defer r.m.Unlock()
	log := log.FromContext(ctx)
	log.Info("EnsurePackageRevision")

	// saftey sync with the repo
	if err := r.repo.FetchRemoteRepository(ctx); err != nil {
		return err
	}
	if pkgRev.Spec.Lifecycle == pkgv1alpha1.PackageRevisionLifecyclePublished && pkgRev.Spec.PackageID.Revision != "" {
		return r.commitToMain(ctx, pkgRev)
		//return r.pushTag(ctx, pkgRev)
	}
	return fmt.Errorf("EnsurePackageRevision should only be used for published packages with a revision")
}

func (r *gitRepository) UpsertPackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, resources map[string]string) error {
	ctx, span := tracer.Start(ctx, "gitRepository::UpsertPackageRevision", trace.WithAttributes())
	defer span.End()
	r.m.Lock()
	defer r.m.Unlock()
	log := log.FromContext(ctx)

	// saftey sync with the repo
	if err := r.repo.FetchRemoteRepository(ctx); err != nil {
		return err
	}

	if pkgRev.Spec.Lifecycle == pkgv1alpha1.PackageRevisionLifecyclePublished {
		// ignore resources

		log.Error("in published mode we should not perform an update of the repo, we should use only ensurePackageRevision")

		// This code is obsolete, since we never do an update when in published state
		/*
			if pkgRev.Spec.PackageID.Revision != "" {
				// validate if the pkg revision tag exists
				pkgTagRefName := packageTagRefName(pkgRev.Spec.PackageID, pkgRev.Spec.PackageID.Revision)
				if _, err := r.repo.Repo.Reference(pkgTagRefName, true); err != nil {
					log.Error("get main pkg revision tag", "error", err)
					return err
				}
			} else {
				// allocated a revision
				newRev, err := r.getNextRevision(pkgRev)
				if err != nil {
					return err
				}
				log.Info("next revision", "newRev", newRev)
				// we set the revision in the spec such that this is returned to the reconciler
				pkgRev.Spec.PackageID.Revision = newRev

				// tag the package
				//pkgTagName := packageTagName(pkgRev.Spec.Package, pkgRev.Spec.Revision)
				if err := r.pushTag(ctx, pkgRev); err != nil {
					return err
				}

			}
		*/
	} else {
		// create the workspace package branch if it does not exist
		// based on the strategy we can use various parent commits
		// strategy latest: uses the package resources of the prevision pkg revision if it exists
		// strategy main: uses the package resources of main if it exists
		if _, err := r.commit(ctx, pkgRev, resources); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (r *gitRepository) commit(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, resources map[string]string) (plumbing.Hash, error) {
	ctx, span := tracer.Start(ctx, "gitRepository::commit", trace.WithAttributes())
	defer span.End()
	log := log.FromContext(ctx)
	// get the parent commit
	// could be either the head of the main branch or the latest pkg revision
	var parentCommit *object.Commit
	var commitMsg string
	var err error
	if pkgRev.Spec.PackageID.Revision == "" {
		parentCommit, commitMsg, err = r.getParentCommit(ctx, pkgRev)
		if err != nil {
			return plumbing.ZeroHash, err
		}
	} else {
		parentCommit, commitMsg, err = r.getHeadCommit(ctx, pkgRev)
		if err != nil {
			return plumbing.ZeroHash, err
		}
	}

	// get the commit helper, the packageTree Hash allows to add the resources of the
	// parent package if this is the strategy that was adopted
	ch := newCommithelper(r.repo.Repo, parentCommit)

	// get the commit message
	commitMessage, err := getCommitMessage(pkgRev, commitMsg)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	if err := ch.Initialize(ctx, r.cr.GetDirectory(), pkgRev); err != nil {
		return plumbing.ZeroHash, err
	}
	if pkgRev.Spec.UpdatePolicy == pkgv1alpha1.UpdatePolicy_Strict {
		ch.initPackage()
	}

	// add the new resources to the commit helper
	for k, v := range resources {
		// append the file root package directory to the path
		ch.storeFile(k, v)
	}
	// commit the resources
	commitHash, _, err := ch.commit(ctx, commitMessage)
	if err != nil {
		log.Error("failed to commit", "error", err.Error())
		return plumbing.ZeroHash, fmt.Errorf("failed to commit package: %w", err)
	}

	refName := workspacePackageBranchRefName(pkgRev.Spec.PackageID) // normally the workspace name
	if pkgRev.Spec.PackageID.Revision != "" {
		refName = mainRefName(string(r.branch)) // when the package revision is
	}

	localRef := plumbing.NewHashReference(refName, commitHash)

	// push the commit to the remote
	refSpecs := newPushRefSpecBuilder()
	// build the refs to push to the remote reference
	refSpecs.AddRefToPush(localRef.Name(), commitHash)

	specs, require, err := refSpecs.BuildRefSpecs()
	if err != nil {
		log.Error("failed to build push commit ref specs", "error", err.Error())
		return plumbing.ZeroHash, err
	}
	if err := r.repo.PushAndCleanup(ctx, specs, require); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			log.Error("failed to push commit", "error", err.Error())
			return plumbing.ZeroHash, err
		}
	}
	return commitHash, nil
}

func (r *gitRepository) getHeadCommit(_ context.Context, pkgRev *pkgv1alpha1.PackageRevision) (*object.Commit, string, error) {
	//log := log.FromContext(ctx)
	ref, err := r.repo.Repo.Reference(mainRefName(string(r.branch)), true)
	if err != nil {
		// TODO: handle empty repositories - NotFound error
		return nil, "", err
	}
	parentCommit, err := r.repo.Repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, "", err
	}

	commitString := fmt.Sprintf("approved commit workspace %s", pkgRev.Spec.PackageID.Workspace)

	return parentCommit, commitString, nil
}

func (r *gitRepository) getParentCommit(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) (*object.Commit, string, error) {
	log := log.FromContext(ctx)
	var parentCommit *object.Commit
	var commitString string
	log.Info("getParentCommit", "repo", r.cr.Name, "pkgID", pkgRev.Spec.PackageID)

	// if the workspace package reference exists we return this package reference
	// if not we can get the parentCommit from either: latest pkgRev, main, a specific Version
	wsPkgRefName := workspacePackageBranchRefName(pkgRev.Spec.PackageID)
	if ref, err := r.repo.Repo.Reference(wsPkgRefName, false); err != nil {
		// TBD: we did not implement the full strategy now; we just do latest or fallback
		latestRev, err := r.getLatestRevision(ctx, pkgRev)
		if err != nil {
			return nil, "", err
		}
		var ref *plumbing.Reference
		if latestRev == NoRevision {
			log.Info("getParentCommit use main", "workspaceBranch does not exists", wsPkgRefName.String(), "revision", "noRevision", "ref", r.branch.BranchInLocal().String())
			// if there is no revision we take latest
			// get the main ref of the repo -> typically main
			ref, err = r.repo.Repo.Reference(r.branch.BranchInLocal(), false)
			if err != nil {
				log.Error("get reference", "error", err)
				return nil, "", err
			}
			commitString = "initial commit from main"
			log.Debug("getParentCommit", "reference", ref.Hash().String())
			// get the head commit of the ref in the repo
			parentCommit, err = r.repo.Repo.CommitObject(ref.Hash())
			if err != nil {
				// We dont support empty repositories
				return nil, "", err
			}
		} else {
			tagRefName := packageTagRefName(pkgRev.Spec.PackageID, latestRev)
			log.Info("getParentCommit use revision tag", "workspaceBranch", "does not exist", "revision", latestRev, "ref", tagRefName.String())
			ref, err = r.repo.Repo.Reference(tagRefName, true)
			if err != nil {
				log.Error("get reference", "error", err)
				return nil, "", err
			}
			commitString = fmt.Sprintf("initial conmit from package revision: %s", latestRev)
			_, parentCommit, err = r.getBranchAndCommitFromTag(ctx, pkgRev.Spec.PackageID.Package, ref)
			if err != nil {
				log.Error("get reference", "error", err)
				return nil, "", err
			}
		}
	} else {
		log.Info("getParentCommit use workspace branch", "workspaceBranch", wsPkgRefName.String())
		// the workspacePackage branch already exists -> take the parent commit
		parentCommit, err = r.repo.Repo.CommitObject(ref.Hash())
		if err != nil {
			// Strange
			return nil, "", err
		}
		commitString = "intermediate commit"
	}
	return parentCommit, commitString, nil
}

// commitToMain takes the data from the workspace branch and commits this to main
// steps:
// 1. check if the workspacebranch exists
// 2a: workspace branch exists

// 2b: workspace branch does not exists -> tag should exist
func (r *gitRepository) commitToMain(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) error {
	log := log.FromContext(ctx)
	log.Info("commitToMain")
	// check if the workspaceBranch still exists
	branchRefName := workspacePackageBranchRefName(pkgRev.Spec.PackageID)
	refName, err := r.repo.Repo.Reference(branchRefName, true)
	if err != nil {
		return err
	}
	log.Info("commitToMain", "branchref", refName.Name().String())
	// the branch exists we execute the following things
	// get Resources from workspace branch -> we use the true flag to indicate to use
	// the workspacebranch iso the package revision
	resources, err := r.getResources(ctx, pkgRev, true)
	if err != nil {
		return err
	}
	log.Info("commitToMain", "resources", resources)
	commitHash, err := r.commit(ctx, pkgRev, resources)
	if err != nil {
		return err
	}
	pkgTagName := packageTagName(pkgRev.Spec.PackageID, pkgRev.Spec.PackageID.Revision)
	log.Info("commitToMain", "pkgTagName", pkgTagName)
	// Get the commit object from the reference.
	commitObj, err := r.repo.Repo.CommitObject(commitHash)
	if err != nil {
		return err
	}

	tagRef, err := r.repo.Repo.CreateTag(string(pkgTagName), commitObj.Hash, &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  commitSignatureName,
			Email: commitSignatureEmail,
			When:  time.Now(),
		},
		Message: string(pkgTagName),
	})
	if err != nil {
		if !strings.Contains(err.Error(), "tag already exists") {
			log.Error("cannot create tag", "error", err)
			return err
		}
		log.Info("commitToMain", "tag already exists", pkgTagName)
		return nil
	}

	log.Info("create tag local", "tagRef", pkgTagName.TagInLocal().String(), "tagRef", string(pkgTagName))
	// push the tag
	refSpecs := newPushRefSpecBuilder()
	// build the refs to push to the remote reference
	refSpecs.AddRefToPush(tagRef.Name(), commitObj.Hash)

	specs, require, err := refSpecs.BuildRefSpecs()
	if err != nil {
		return err
	}
	if err := r.repo.PushAndCleanup(ctx, specs, require); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return err
		}
	}

	if err := r.deleteRef(ctx, refName); err != nil {
		if !strings.Contains(err.Error(), "reference not found") {
			return err
		}
	}

	return nil

}

// DO NOT DELETE
// pushTag was used when not committing to main, but just tagging the branch
/*
func (r *gitRepository) pushTag(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) error {
	log := log.FromContext(ctx)
	pkgTagName := packageTagName(pkgRev.Spec.PackageID, pkgRev.Spec.PackageID.Revision)
	log.Info("push tag", "name", pkgTagName.TagInLocal().String())

	wsPkgRefName := workspacePackageBranchRefName(pkgRev.Spec.PackageID)
	ref, err := r.repo.Repo.Reference(wsPkgRefName, true)
	if err != nil {
		return err
	}
	// Get the commit object from the reference.
	commitObj, err := r.repo.Repo.CommitObject(ref.Hash())
	if err != nil {
		return err
	}

	tagRef, err := r.repo.Repo.CreateTag(string(pkgTagName), commitObj.Hash, &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  commitSignatureName,
			Email: commitSignatureEmail,
			When:  time.Now(),
		},
		Message: string(pkgTagName),
	})
	if err != nil {
		if !strings.Contains(err.Error(), "tag already exists") {
			log.Error("cannot create tag", "error", err)
			return err
		}
		return nil
	}

	log.Info("create tag local", "tagRef", pkgTagName.TagInLocal().String(), "tagRef", string(pkgTagName))
	// push the tag
	refSpecs := newPushRefSpecBuilder()
	// build the refs to push to the remote reference
	refSpecs.AddRefToPush(tagRef.Name(), commitObj.Hash)

	specs, require, err := refSpecs.BuildRefSpecs()
	if err != nil {
		return err
	}
	if err := r.repo.PushAndCleanup(ctx, specs, require); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return err
		}
	}
	return nil
}
*/

// deleteRef deletes branches or tags in git
func (r *gitRepository) deleteRef(ctx context.Context, ref *plumbing.Reference) error {
	log := log.FromContext(ctx)
	log.Info("delete ref", "name", ref.Name().String())
	// prepare refSpecs
	refSpecs := newPushRefSpecBuilder()
	// build the refs to push to the remote reference
	refSpecs.AddRefToDelete(ref)

	specs, require, err := refSpecs.BuildRefSpecs()
	if err != nil {
		return err
	}
	if err := r.repo.PushAndCleanup(ctx, specs, require); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return err
		}
	}
	return nil
}
