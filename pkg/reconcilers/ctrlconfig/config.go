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

package ctrlconfig

import (
	"context"

	"github.com/henderiw/logger/log"
	"github.com/pkgserver-dev/pkgserver/apis/generated/clientset/versioned"
	"github.com/pkgserver-dev/pkgserver/pkg/cache"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/packagediscovery/catalog"
	"k8s.io/apimachinery/pkg/types"
)

type ControllerConfig struct {
	RepoCache    *cache.Cache
	CatalogStore *catalog.Store
	ClientSet    *versioned.Clientset
}

func InitContext(ctx context.Context, controllerName string, req types.NamespacedName) context.Context {
	l := log.FromContext(ctx).With("controller", controllerName, "req", req)
	return log.IntoContext(ctx, l)
}
