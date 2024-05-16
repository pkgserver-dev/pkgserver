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

package pkg

import (
	"encoding/json"
	"fmt"

	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
)

// gitAnnotation is the structured data that we store with commits.
// Currently this is stored as a json-encoded blob in the commit message,
type gitAnnotation struct {
	// PackagePath is the path of the package the commit belings to.
	PackagePath string `json:"package,omitempty"`

	// WorkspaceName holds the workspaceName of the package revision the commit
	// belongs to.
	WorkspaceName string `json:"workspaceName,omitempty"`

	// Revision hold the revision of the package revision the commit
	// belongs to.
	Revision string `json:"revision,omitempty"`

	// Lifecycle holds the lifecycle the commit belongs to
	Lifecycle string `json:"lifecycle,omitempty"`

	//Task holds the task we performed, if a task caused the commit.
	Task string `json:"task,omitempty"`
}

// AnnotateCommitMessage adds the gitAnnotation to the commit message.
func AnnotateCommitMessage(message string, annotation *gitAnnotation) (string, error) {
	b, err := json.Marshal(annotation)
	if err != nil {
		return "", fmt.Errorf("error marshaling annotation: %w", err)
	}

	message += "\n\npkg-server:" + string(b) + "\n"

	return message, nil
}

func GetGitAnnotation(pkgRev *pkgv1alpha1.PackageRevision) *gitAnnotation {
	a := &gitAnnotation{
		PackagePath:   pkgRev.Spec.PackageRevID.Package,
		WorkspaceName: pkgRev.Spec.PackageRevID.Workspace,
		Revision:      pkgRev.Spec.PackageRevID.Revision,
		Lifecycle:     string(pkgRev.Spec.Lifecycle),
	}
	if len(pkgRev.Spec.Tasks) != 0 {
		a.Task = string(pkgRev.Spec.Tasks[0].Type)
	}
	return a
}
