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

package catalog

import (
	"fmt"
	"sort"
	"sync"

	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Dependency is per packageRevision
type Dependency struct {
	m                  *sync.RWMutex
	resolutionError    error
	resolutionErrors   map[schema.GroupVersionKind]error
	resolutionWarnings map[schema.GroupVersionKind]error
	pkgId              pkgid.PackageID
	pkgDependencies    map[schema.GroupVersionKind]sets.Set[pkgid.Upstream]
	resDependencies   sets.Set[schema.GroupVersionKind]
}

func NewDependency(pkgID pkgid.PackageID) *Dependency {
	return &Dependency{
		m:                  new(sync.RWMutex),
		resolutionError:    nil,
		resolutionErrors:   map[schema.GroupVersionKind]error{},
		resolutionWarnings: map[schema.GroupVersionKind]error{},
		pkgId:              pkgID,
		pkgDependencies:    map[schema.GroupVersionKind]sets.Set[pkgid.Upstream]{},
		resDependencies:   sets.Set[schema.GroupVersionKind]{},
	}
}

func (r *Dependency) PkgString() string {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.pkgId.PkgString()
}

func (r *Dependency) HasResolutionError() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.resolutionError != nil || len(r.resolutionErrors) != 0
}

func (r *Dependency) GetResolutionError() error {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.resolutionError
}

func (r *Dependency) ListResolutionErrors() map[schema.GroupVersionKind]error {
	r.m.RLock()
	defer r.m.RUnlock()
	resolutionErrors := map[schema.GroupVersionKind]error{}
	for gvk, err := range r.resolutionErrors {
		resolutionErrors[gvk] = err
	}
	return resolutionErrors
}

func (r *Dependency) ListResolutionWarnings() map[schema.GroupVersionKind]error {
	r.m.RLock()
	defer r.m.RUnlock()
	resolutionWarnings := map[schema.GroupVersionKind]error{}
	for gvk, err := range r.resolutionWarnings {
		resolutionWarnings[gvk] = err
	}
	return resolutionWarnings
}

func (r *Dependency) AddResolutionError(err error) {
	r.m.Lock()
	defer r.m.Unlock()
	r.resolutionError = err
}

func (r *Dependency) AddGVKResolutionError(gvk schema.GroupVersionKind, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	r.resolutionErrors[gvk] = err
}

func (r *Dependency) AddGVKResolutionWarning(gvk schema.GroupVersionKind, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	r.resolutionWarnings[gvk] = err
}

func (r *Dependency) AddPkgDependency(gvk schema.GroupVersionKind, upstream pkgid.Upstream) {
	r.m.Lock()
	defer r.m.Unlock()

	if _, ok := r.pkgDependencies[gvk]; !ok {
		r.pkgDependencies[gvk] = sets.New[pkgid.Upstream]()
	}
	r.pkgDependencies[gvk].Insert(*upstream.DeepCopy())
}

// ListPkgDependencies lists a unique set of upstream packages
func (r *Dependency) ListPkgDependencies() []pkgid.Upstream {
	r.m.RLock()
	defer r.m.RUnlock()
	upstreamSet := sets.New[pkgid.Upstream]()
	for _, upstreams := range r.pkgDependencies {
		for _, upstream := range upstreams.UnsortedList() {
			upstreamSet.Insert(upstream)
		}
	}
	upstreams := upstreamSet.UnsortedList()
	sort.SliceStable(upstreams, func(i, j int) bool {
		return upstreams[i].PkgString() < upstreams[j].PkgString()
	})
	return upstreams
}

func (r *Dependency) ListGVKPkgDependencies() map[schema.GroupVersionKind][]pkgid.Upstream {
	r.m.RLock()
	defer r.m.RUnlock()
	gvkPkgDeps := map[schema.GroupVersionKind][]pkgid.Upstream{}
	for gvk, upstreamSets := range r.pkgDependencies {
		upstreams := upstreamSets.UnsortedList()
		sort.SliceStable(upstreams, func(i, j int) bool {
			return upstreams[i].PkgString() < upstreams[j].PkgString()
		})
		gvkPkgDeps[gvk] = upstreams
	}
	return gvkPkgDeps
}

func (r *Dependency) AddCoreDependency(gvk schema.GroupVersionKind) {
	r.m.Lock()
	defer r.m.Unlock()

	r.resDependencies.Insert(gvk)
}

func (r *Dependency) ListCoreDependencies() []schema.GroupVersionKind {
	r.m.RLock()
	defer r.m.RUnlock()
	gvks := r.resDependencies.UnsortedList()
	sort.SliceStable(gvks, func(i, j int) bool {
		return gvks[i].Group < gvks[j].Group
	})
	return gvks
}

func (r *Dependency) PrintResolutionErrors() {
	if r.GetResolutionError() != nil {
		fmt.Printf("  resolution error: %s\n", r.GetResolutionError().Error())
	}

	for gvk, error := range r.ListResolutionErrors() {
		fmt.Printf("  gvk %s, resolution error %s\n", gvk.String(), error.Error())
	}

	for gvk, error := range r.ListResolutionWarnings() {
		fmt.Printf("  gvk %s, resolution warning %s\n", gvk.String(), error.Error())
	}
}

func (r *Dependency) PrintCoreDependencies() {
	gvks := r.ListCoreDependencies()
	for _, gvk := range gvks {
		fmt.Println("  resource gvk", gvk.String())
	}
}

func (r *Dependency) PrintPkgDependencies() {
	for gvk, upstreams := range r.ListGVKPkgDependencies() {
		upstreamStrings := []string{}
		for _, upstream := range upstreams {
			upstreamStrings = append(upstreamStrings, fmt.Sprintf("%s:%s:%s", upstream.Repository, upstream.Package, upstream.Revision))
		}
		fmt.Printf("  pkg gvk %s, upstreams %v\n", gvk.String(), upstreamStrings)
	}
}
