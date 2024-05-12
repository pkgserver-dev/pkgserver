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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/henderiw/logger/log"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"golang.org/x/mod/semver"
)

const (
	NoRevision = "v0"
)

func (r *gitRepository) getLatestRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) (string, error) {
	log := log.FromContext(ctx)
	tags, err := r.repo.Repo.Tags()
	if err != nil {
		return "", err
	}
	log.Info("getLatestRevision", "repo", r.cr.Name, "pkgID", pkgRev.Spec.PackageID)

	pkgTags := []string{}
	if err = tags.ForEach(func(t *plumbing.Reference) error {
		log.Info("getLatestRevision", "package", pkgRev.Spec.PackageID.Package, "tag", t.Name().String())
		if strings.Contains(t.Name().String(), pkgRev.Spec.PackageID.Package) {
			// add the revision (last part of the tag (syntax ns/pkg/rev)) to the pkg Tags
			pkgTags = append(pkgTags, filepath.Base(t.Name().String()))
		}
		return nil
	}); err != nil {
		return "", err
	}
	return LatestRevisionNumber(pkgTags), nil
}

func (r *gitRepository) getNextRevision(pkgrev *pkgv1alpha1.PackageRevision) (string, error) {
	tags, err := r.repo.Repo.Tags()
	if err != nil {
		return "", err
	}

	pkgTags := []string{}
	if err = tags.ForEach(func(t *plumbing.Reference) error {
		if strings.Contains(t.Name().String(), pkgrev.Spec.PackageID.Package) {
			// add the revision (last part of the tag) to the pkg Tags
			pkgTags = append(pkgTags, filepath.Base(t.Name().String()))
		}
		return nil
	}); err != nil {
		return "", err
	}
	return NextRevisionNumber(pkgTags)
}

// NextRevisionNumber computes the next revision number as the latest revision number + 1.
// This function only understands strict versioning format, e.g. v1, v2, etc. It will
// ignore all revision numbers it finds that do not adhere to this format.
// If there are no published revisions (in the recognized format), the revision
// number returned here will be "v1".
func NextRevisionNumber(revs []string) (string, error) {
	latestRev := NoRevision
	for _, currentRev := range revs {
		if !semver.IsValid(currentRev) {
			// ignore this revision
			continue
		}
		// collect the major version. i.e. if we find that the latest published
		// version is v3.1.1, we will end up returning v4
		currentRev = semver.Major(currentRev)

		switch cmp := semver.Compare(currentRev, latestRev); {
		case cmp == 0:
			// Same revision.
		case cmp < 0:
			// current < latest; no change
		case cmp > 0:
			// current > latest; update latest
			latestRev = currentRev
		}

	}
	i, err := strconv.Atoi(latestRev[1:])
	if err != nil {
		return "", err
	}
	i++
	next := "v" + strconv.Itoa(i)
	return next, nil
}

// LatestRevisionNumber computes the latest revision number of a given list.
// This function only understands strict versioning format, e.g. v1, v2, etc. It will
// ignore all revision numbers it finds that do not adhere to this format.
// If there are no published revisions (in the recognized format), the revision
// number returned here will be "v0".
func LatestRevisionNumber(revs []string) string {
	latestRev := NoRevision
	for _, currentRev := range revs {
		if !semver.IsValid(currentRev) {
			// ignore this revision
			continue
		}
		// collect the major version. i.e. if we find that the latest published
		// version is v3.1.1, we will end up returning v4
		currentRev = semver.Major(currentRev)

		switch cmp := semver.Compare(currentRev, latestRev); {
		case cmp == 0:
			// Same revision.
		case cmp < 0:
			// current < latest; no change
		case cmp > 0:
			// current > latest; update latest
			latestRev = currentRev
		}

	}
	return latestRev
}
