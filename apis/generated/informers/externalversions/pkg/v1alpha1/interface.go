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
// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "github.com/pkgserver-dev/pkgserver/apis/generated/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// PackageRevisions returns a PackageRevisionInformer.
	PackageRevisions() PackageRevisionInformer
	// PackageRevisionResourceses returns a PackageRevisionResourcesInformer.
	PackageRevisionResourceses() PackageRevisionResourcesInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// PackageRevisions returns a PackageRevisionInformer.
func (v *version) PackageRevisions() PackageRevisionInformer {
	return &packageRevisionInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// PackageRevisionResourceses returns a PackageRevisionResourcesInformer.
func (v *version) PackageRevisionResourceses() PackageRevisionResourcesInformer {
	return &packageRevisionResourcesInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
