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
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
)

const (
	//DefaultMainReferenceName plumbing.ReferenceName = "refs/heads/main"
	OriginName string     = "origin"
	MainBranch BranchName = "main"

	//branchPrefixInLocalRepo  = "refs/heads/"
	branchPrefixInLocalRepo  = "refs/remotes/" + OriginName + "/"
	branchPrefixInRemoteRepo = "refs/heads/"
	tagsPrefixInLocalRepo    = "refs/tags/"
	tagsPrefixInRemoteRepo   = "refs/tags/"

	branchRefSpec config.RefSpec = config.RefSpec("+" + branchPrefixInRemoteRepo + "*:" + branchPrefixInLocalRepo + "*")
	tagRefSpec    config.RefSpec = config.RefSpec("+" + tagsPrefixInRemoteRepo + "*:" + tagsPrefixInLocalRepo + "*")
)

var (
	DefaultMainReferenceName plumbing.ReferenceName = plumbing.ReferenceName(branchPrefixInLocalRepo + "/" + string(MainBranch))
	// The default fetch spec contains both branches and tags.
	// This enables push of a tag which will automatically update
	// its local reference, avoiding explicitly setting of refs.
	defaultFetchSpec []config.RefSpec = []config.RefSpec{
		branchRefSpec,
		tagRefSpec,
	}
	// DO NOT USE for fetches. Used for reverse reference mapping only.
	reverseFetchSpec []config.RefSpec = []config.RefSpec{
		config.RefSpec(branchPrefixInLocalRepo + "*:" + branchPrefixInRemoteRepo + "*"),
		config.RefSpec(tagsPrefixInLocalRepo + "*:" + tagsPrefixInRemoteRepo + "*"),
	}
)

// BranchName represents a relative branch name (i.e. 'main', 'drafts/bucket/v1')
// and supports transformation to the ReferenceName in local (cached) repository
// (those references are in the form 'refs/remotes/origin/...') or in the remote
// repository (those references are in the form 'refs/heads/...').
type BranchName string
type TagName string

func (b BranchName) BranchInLocal() plumbing.ReferenceName {
	return plumbing.ReferenceName(branchPrefixInLocalRepo + string(b))
}

func (b BranchName) BranchInRemote() plumbing.ReferenceName {
	return plumbing.ReferenceName(branchPrefixInRemoteRepo + string(b))
}

func (b TagName) TagInLocal() plumbing.ReferenceName {
	return plumbing.ReferenceName(tagsPrefixInLocalRepo + string(b))
}

func (b TagName) TagInRemote() plumbing.ReferenceName {
	return plumbing.ReferenceName(tagsPrefixInRemoteRepo + string(b))
}

func isMainBranch(n plumbing.ReferenceName, branch string) bool {
	fmt.Println("isRepoBranch ref", n.String())
	fmt.Println("mainBranch ref", fmt.Sprintf("%s%s", branchPrefixInLocalRepo, branch))
	return n.String() == fmt.Sprintf("%s%s", branchPrefixInLocalRepo, branch)
}

func isBranchInLocalRepo(n plumbing.ReferenceName) bool {
	return strings.HasPrefix(n.String(), branchPrefixInLocalRepo)
}

func isTagInLocalRepo(n plumbing.ReferenceName) bool {
	return strings.HasPrefix(n.String(), tagsPrefixInLocalRepo)
}

func getTagNameInLocalRepo(n plumbing.ReferenceName) (string, bool) {
	return trimOptionalPrefix(n.String(), tagsPrefixInLocalRepo)
}

func trimOptionalPrefix(s, prefix string) (string, bool) {
	if strings.HasPrefix(s, prefix) {
		return strings.TrimPrefix(s, prefix), true
	}
	return "", false
}

func mainRefName(branch string) plumbing.ReferenceName {
	return BranchName(branch).BranchInLocal()
}

func workspacePackageBranchRefName(pkgID pkgid.PackageID) plumbing.ReferenceName {
	return packageBranchName(pkgID).BranchInLocal()
}

func packageBranchName(pkgID pkgid.PackageID) BranchName {
	if pkgID.Target == pkgid.PkgTarget_Catalog {
		return BranchName(fmt.Sprintf("%s/%s/%s", pkgID.Realm, pkgID.Package, pkgID.Workspace))
	}
	return BranchName(fmt.Sprintf("%s/%s/%s/%s", pkgID.Target, pkgID.Realm, pkgID.Package, pkgID.Workspace))
}

func packageTagRefName(pkgID pkgid.PackageID, rev string) plumbing.ReferenceName {
	return packageTagName(pkgID, rev).TagInLocal()
}

func packageTagName(pkgID pkgid.PackageID, rev string) TagName {
	if pkgID.Target == pkgid.PkgTarget_Catalog {
		return TagName(fmt.Sprintf("%s/%s/%s", pkgID.Realm, pkgID.Package, rev))
	}
	return TagName(fmt.Sprintf("%s/%s/%s/%s", pkgID.Target, pkgID.Realm, pkgID.Package, rev))
}

func refInRemoteFromRefInLocal(n plumbing.ReferenceName) (plumbing.ReferenceName, error) {
	return translateReference(n, reverseFetchSpec)
}

func translateReference(n plumbing.ReferenceName, specs []config.RefSpec) (plumbing.ReferenceName, error) {
	for _, spec := range specs {
		if spec.Match(n) {
			return spec.Dst(n), nil
		}
	}
	return "", fmt.Errorf("cannot translate reference %s", n)
}
