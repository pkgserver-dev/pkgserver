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
	"sync"

	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Package struct {
	m         *sync.RWMutex
	// installed revisions of this package
	Revisions sets.Set[pkgid.PackageID] 
	// packages that have a dependency with this package, we store per revision
	// now we can argue if this is right but it is probably the easiest and
	// we can optimize cornercases afterwards
	OwnerRefs sets.Set[pkgid.PackageID]
}

func NewPackage() *Package {
	return &Package{
		m:         new(sync.RWMutex),
		Revisions: sets.New[pkgid.PackageID](),
		OwnerRefs: sets.New[pkgid.PackageID](),
	}
}

func (r *Package) AddPackageRevision(pkgID *pkgid.PackageID) {
	r.m.Lock()
	defer r.m.Unlock()

	r.Revisions.Insert(*pkgID)
}

func (r *Package) DeletePackageRevision(pkgID *pkgid.PackageID) {
	r.m.Lock()
	defer r.m.Unlock()

	r.Revisions.Delete(*pkgID)
}

func (r *Package) ListPackageRevisions(pkgID *pkgid.PackageID) []pkgid.PackageID {
	r.m.RLock()
	defer r.m.RUnlock()

	return r.Revisions.UnsortedList()
}

func (r *Package) AddOwnerRef(pkgID *pkgid.PackageID) {
	r.m.Lock()
	defer r.m.Unlock()

	r.OwnerRefs.Insert(*pkgID)
}

func (r *Package) DeleteOwnerRef(pkgID *pkgid.PackageID) {
	r.m.Lock()
	defer r.m.Unlock()

	r.OwnerRefs.Delete(*pkgID)
}

func (r *Package) ListOwnerrefs(pkgID *pkgid.PackageID) []pkgid.PackageID {
	r.m.RLock()
	defer r.m.RUnlock()

	return r.OwnerRefs.UnsortedList()
}
