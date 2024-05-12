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

package packagediscovery

import (
	"context"
	"time"

	"github.com/henderiw/logger/log"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/packagediscovery/catalog"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type APIDiscovery interface {
	Start(ctx context.Context)
	Stop(ctx context.Context)
}

type apiDiscoverer struct {
	client          client.Client
	discoveryClient *discovery.DiscoveryClient
	catalogStore    *catalog.Store

	cancel context.CancelFunc
}

func NewAPIDiscoverer(
	client client.Client,
	config *rest.Config,
	catalogStore *catalog.Store,
) APIDiscovery {
	return &apiDiscoverer{
		client:          client,
		discoveryClient: discovery.NewDiscoveryClientForConfigOrDie(config),
		catalogStore:    catalogStore,
	}
}

func (r *apiDiscoverer) Stop(ctx context.Context) {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *apiDiscoverer) Start(ctx context.Context) {
	ctx, r.cancel = context.WithCancel(ctx)
	log := log.FromContext(ctx).With("name", "apiDiscoverer")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			apiResources, err := r.discoveryClient.ServerPreferredResources()
			if err != nil {
				log.Error("cannot get server resources")
			} else {
				r.catalogStore.UpdateAPIfromAPIResources(ctx, apiResources)
			}
			r.catalogStore.Print(ctx)
			time.Sleep(10 * time.Second)
		}
	}
}
