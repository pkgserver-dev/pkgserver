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

package v1alpha1

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkgserver-dev/pkgserver/apis/condition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetCondition returns the condition based on the condition kind
func (r *PackageVariant) GetCondition(t condition.ConditionType) condition.Condition {
	return r.Status.GetCondition(t)
}

// SetConditions sets the conditions on the resource. it allows for 0, 1 or more conditions
// to be set at once
func (r *PackageVariant) SetConditions(c ...condition.Condition) {
	r.Status.SetConditions(c...)
}

func GetPackageRevisionName(repo, pkg, ws string) string {
	return fmt.Sprintf("%s.%s.%s", repo, PackageToName(pkg), ws)
}

func PackageToName(pkg string) string {
	return strings.ReplaceAll(pkg, "/", ":")
}

func (r *PackageVariant) GetWorkspaceName(hash [sha1.Size]byte) string {
	//return fmt.Sprintf("pv-%s-%s", r.Name, hex.EncodeToString(hash[:8]))
	return fmt.Sprintf("pv-%s", hex.EncodeToString(hash[:8]))
}

func (r PackageVariantSpec) CalculateHash() ([sha1.Size]byte, error) {
	// Convert the struct to JSON
	jsonData, err := json.Marshal(r)
	if err != nil {
		return [sha1.Size]byte{}, err
	}

	// Calculate SHA-1 hash
	return sha1.Sum(jsonData), nil
}

// BuildPackageVariant returns an PackageRevision from a client Object a Spec/Status
func BuildPackageVariant(meta metav1.ObjectMeta, spec PackageVariantSpec, status PackageVariantStatus) *PackageVariant {
	return &PackageVariant{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.Identifier(),
			Kind:       PackageVariantKind,
		},
		ObjectMeta: meta,
		Spec:       spec,
		Status:     status,
	}
}
