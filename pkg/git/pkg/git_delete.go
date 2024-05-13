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
	"strings"

	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
)

func (r *gitRepository) DeletePackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) error {
	//log := log.FromContext(ctx)

	// saftey sync with the repo
	if err := r.repo.FetchRemoteRepository(ctx); err != nil {
		return err
	}
	if pkgRev.Spec.PackageID.Revision != "" {
		// delete tag
		pkgTagRefName := packageTagRefName(pkgRev.Spec.PackageID, pkgRev.Spec.PackageID.Revision)
		if tagRef, err := r.repo.Repo.Reference(pkgTagRefName, true); err == nil {
			if err := r.deleteRef(ctx, tagRef); err != nil {
				if !strings.Contains(err.Error(), "reference not found") {
					return err
				}
			}
		}
		// TBD: do we need an option to delete the pkgWorkspace branch ? Prefer not to do this.
		// NOT needed when we commit to main
		/*
			wsPkgRefName := workspacePackageBranchRefName(pkgRev.Spec.PackageID)
			if wsPkgBranchRef, err := r.repo.Repo.Reference(wsPkgRefName, true); err != nil {
				return err
			} else {
				if err := r.deleteRef(ctx, wsPkgBranchRef); err != nil {
					if !strings.Contains(err.Error(), "reference not found") {
						return err
					}
				}
			}
		*/
		return nil
	} else {
		// TBD: do we need an option to retain the pkgWorkspace branch ?
		wsPkgRefName := workspacePackageBranchRefName(pkgRev.Spec.PackageID)
		if wsPkgBranchRef, err := r.repo.Repo.Reference(wsPkgRefName, true); err != nil {
			return err
		} else {
			if err := r.deleteRef(ctx, wsPkgBranchRef); err != nil {
				return err
			}
		}
	}
	return nil
}
