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

package git

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/henderiw/logger/log"
	"github.com/pkgserver-dev/pkgserver/pkg/auth"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("git")

type Repository struct {
	Name                     string
	Namespace                string
	OriginName               string
	DefaultMainReferenceName plumbing.ReferenceName
	DefaultFetchSpec         []config.RefSpec
	URL                      string
	Directory                string
	CredentialSecret         string
	CredentialResolver       auth.CredentialResolver
	UserInfoProvider         auth.UserInfoProvider

	Repo *git.Repository
	// credential contains the information needed to authenticate against
	// a git repository.
	credential auth.Credential
}

func (r *Repository) FetchRemoteRepository(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "gitRepository::fetchRemoteRepository", trace.WithAttributes())
	defer span.End()

	// Fetch
	switch err := r.doGitWithAuth(ctx, func(auth transport.AuthMethod) error {
		return r.Repo.Fetch(&git.FetchOptions{
			RemoteName: r.OriginName,
			Auth:       auth,
		})
	}); err {
	case nil: // OK
	case git.NoErrAlreadyUpToDate:
	case transport.ErrEmptyRemoteRepository:

	default:
		return fmt.Errorf("cannot fetch repository %q: %w", r.URL, err)
	}

	return nil
}

// pushes the local reference to the remote repository
func (r *Repository) PushAndCleanup(ctx context.Context, specs, require []config.RefSpec) error {
	if err := r.doGitWithAuth(ctx, func(auth transport.AuthMethod) error {
		return r.Repo.Push(&git.PushOptions{
			RemoteName:        r.OriginName, // origin
			RefSpecs:          specs,        // e.g. [d48aaa68deca311768be2bb5dd0cd97b8da13971:refs/heads/test-package/test-workspace]
			Auth:              auth,
			RequireRemoteRefs: require, // empty for push
			Force:             true,
		})
	}); err != nil {
		return err
	}
	return nil
}

// doGitWithAuth fetches auth information for git and provides it
// to the provided function which performs the operation against a git repo.
func (r *Repository) doGitWithAuth(ctx context.Context, op func(transport.AuthMethod) error) error {
	log := log.FromContext(ctx)
	auth, err := r.getAuthMethod(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to obtain git credentials: %w", err)
	}
	err = op(auth)
	if err != nil {
		if !errors.Is(err, transport.ErrAuthenticationRequired) {
			return err
		}
		log.Info("Authentication failed. Trying to refresh credentials")
		// TODO: Consider having some kind of backoff here.
		auth, err := r.getAuthMethod(ctx, true)
		if err != nil {
			return fmt.Errorf("failed to obtain git credentials: %w", err)
		}
		return op(auth)
	}
	return nil
}

// getAuthMethod fetches the credentials for authenticating to git. It caches the
// credentials between calls and refresh credentials when the tokens have expired.
func (r *Repository) getAuthMethod(ctx context.Context, forceRefresh bool) (transport.AuthMethod, error) {
	// If no secret is provided, we try without any auth.
	log := log.FromContext(ctx)
	log.Debug("getAuthMethod", "credentialSecret", r.CredentialSecret, "credential", r.credential)
	if r.CredentialSecret == "" {
		return nil, nil
	}

	if r.credential == nil || !r.credential.Valid() || forceRefresh {
		if cred, err := r.CredentialResolver.ResolveCredential(ctx, r.Namespace, r.CredentialSecret); err != nil {
			return nil, fmt.Errorf("failed to obtain credential from secret %s/%s: %w", "", "", err)
		} else {
			r.credential = cred
		}
	}

	return r.credential.ToAuthMethod(), nil
}

type ListRefsFunc func(ctx context.Context, ref *plumbing.Reference) error

func (r *Repository) ListRefs(ctx context.Context, listRefsFunc ListRefsFunc) error {
	if err := r.FetchRemoteRepository(ctx); err != nil {
		return err
	}

	refs, err := r.Repo.References()
	if err != nil {
		return err
	}

	var errall error
	for {
		ref, err := refs.Next()
		if err == io.EOF {
			break
		}

		if listRefsFunc != nil {
			if err := listRefsFunc(ctx, ref); err != nil {
				errall = errors.Join(errall, err)
			}
		}
	}
	return errall
}
