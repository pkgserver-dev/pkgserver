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

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/pkgserver-dev/pkgserver/pkg/auth"
	"github.com/spf13/viper"
)

func NewAuthResolver() Resolver {
	return &ViperAuthResolver{}
}

var _ Resolver = &ViperAuthResolver{}

type ViperAuthResolver struct{}

func (b *ViperAuthResolver) Resolve(_ context.Context, name string) (auth.Credential, error) {
	var secret apis.Secret
	if err := viper.UnmarshalKey(fmt.Sprintf("secrets.%s", name), &secret); err != nil {
		return nil, err
	}

	if secret.Username == "" ||
		secret.Password == "" {
		return nil, fmt.Errorf("a token must be provided, use pkgctl auth add USERNAME TOKEN")
	}
	return &ViperAuthCredential{
		Username: secret.Username,
		Password: secret.Password,
	}, nil
}

type ViperAuthCredential struct {
	Username string
	Password string
}

var _ auth.Credential = &ViperAuthCredential{}

func (b *ViperAuthCredential) Valid() bool {
	return true
}

func (b *ViperAuthCredential) ToAuthMethod() transport.AuthMethod {
	return &http.BasicAuth{
		Username: string(b.Username),
		Password: string(b.Password),
	}
}
