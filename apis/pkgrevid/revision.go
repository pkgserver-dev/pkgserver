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

package pkgrevid

import (
	"strconv"

	"golang.org/x/mod/semver"
)

const (
	NoRevision  = "v0"
	Anyrevision = "*"
)

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

func IsRevLater(currentRev, latestRev string) bool {
	if !semver.IsValid(currentRev) {
		// ignore this revision
		return false
	}
	// collect the major version. i.e. if we find that the latest published
	// version is v3.1.1, we will end up returning v4
	currentRev = semver.Major(currentRev)

	switch cmp := semver.Compare(currentRev, latestRev); {
	case cmp == 0:
		// Same revision.
		return false
	case cmp < 0:
		// current < latest; no change
		return false
	case cmp > 0:
		// current > latest; update latest
		return true
	}
	return false
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
