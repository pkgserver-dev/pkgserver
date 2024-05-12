package pkg

import (
	"fmt"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

type pushRefSpecBuilder struct {
	pushRefs map[plumbing.ReferenceName]plumbing.Hash
	require  map[plumbing.ReferenceName]plumbing.Hash
}

func newPushRefSpecBuilder() *pushRefSpecBuilder {
	return &pushRefSpecBuilder{
		pushRefs: map[plumbing.ReferenceName]plumbing.Hash{},
		require:  map[plumbing.ReferenceName]plumbing.Hash{},
	}
}

func (b *pushRefSpecBuilder) AddRefToPush(to plumbing.ReferenceName, hash plumbing.Hash) {
	b.pushRefs[to] = hash
}

func (b *pushRefSpecBuilder) AddRefToDelete(ref *plumbing.Reference) {
	b.AddRefToPush(ref.Name(), plumbing.ZeroHash)
	b.RequireRef(ref)
}

func (b *pushRefSpecBuilder) RequireRef(ref *plumbing.Reference) {
	if ref != nil {
		b.require[ref.Name()] = ref.Hash()
	}
}

func (b *pushRefSpecBuilder) BuildRefSpecs() (push []config.RefSpec, require []config.RefSpec, err error) {
	for local, hash := range b.pushRefs {
		remote, err := refInRemoteFromRefInLocal(local)
		if err != nil {
			return nil, nil, err
		}

		if !hash.IsZero() {
			push = append(push, config.RefSpec(fmt.Sprintf("%s:%s", hash, remote)))
		} else {
			push = append(push, config.RefSpec(fmt.Sprintf(":%s", remote)))
		}
	}

	for local, hash := range b.require {
		remote, err := refInRemoteFromRefInLocal(local)
		if err != nil {
			return nil, nil, err
		}

		if !hash.IsZero() {
			require = append(require, config.RefSpec(fmt.Sprintf("%s:%s", hash, remote)))
		}
	}

	return push, require, nil
}
