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

package packagescheduler

import (
	"github.com/henderiw/apiserver-store/pkg/storebackend"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

func getkeyFromPkgRev(cr *pkgv1alpha1.PackageRevision) storebackend.Key {
	return storebackend.KeyFromNSN(
		types.NamespacedName{
			Namespace: cr.Spec.PackageRevID.Target,
			Name:      cr.Spec.PackageRevID.PkgString(),
		},
	)
}

/*
func getkeyFromDownStream(downstream *pkgid.Downstream) storebackend.Key {
	return storebackend.KeyFromNSN(
		types.NamespacedName{
			Namespace: downstream.Target,
			Name:      downstream.PkgString(),
		},
	)
}
*/
