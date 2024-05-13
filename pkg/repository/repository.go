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

package repository

import (
	"context"

	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
)

type ListOption struct {
	PackageID *pkgid.PackageID
}

type Repository interface {
	// used for discovery
	ListPackageRevisions(ctx context.Context, opts *ListOption) ([]*pkgv1alpha1.PackageRevision, error)
	// UpsertPackageRevision updates the package revision in the revision backend
	UpsertPackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, resources map[string]string) error
	// DeletePackageRevision deletes the package revision in the revision backend
	DeletePackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) error
	// used for List or Gte PackageRevisionResources
	GetResources(ctx context.Context, pr *pkgv1alpha1.PackageRevision, useWorkspaceBranch bool) (map[string]string, error)
	// ensure packageRevision ensure the Packagerevision tag exists on the package revision
	EnsurePackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) error
}
