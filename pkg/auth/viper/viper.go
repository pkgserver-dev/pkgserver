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

package viper

import (
	"context"
	"fmt"

	"github.com/pkgserver-dev/pkgserver/pkg/auth"
)

func NewCredentialResolver() auth.CredentialResolver {
	return &viperResolver{}
}

type viperResolver struct {
}

var _ auth.CredentialResolver = &viperResolver{}

func (r *viperResolver) ResolveCredential(ctx context.Context, namespace, name string) (auth.Credential, error) {
	resolver := NewAuthResolver()
	cred, err := resolver.Resolve(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("error resolving credential: %w", err)
	}
	return cred, nil
}

type NoMatchingResolverError struct {
	Type string
}

func (e *NoMatchingResolverError) Error() string {
	return fmt.Sprintf("no resolver for secret with type %s", e.Type)
}

func (e *NoMatchingResolverError) Is(err error) bool {
	nmre, ok := err.(*NoMatchingResolverError)
	if !ok {
		return false
	}
	return nmre.Type == e.Type
}
