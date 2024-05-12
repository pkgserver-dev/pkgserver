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

package condition

const (
	ReadinessGate_PkgSchedule ConditionType = "pkg.pkgserver.dev/schedule"
	ReadinessGate_PkgProcess  ConditionType = "pkg.pkgserver.dev/process"
	ReadinessGate_PkgPolicy   ConditionType = "pkg.pkgserver.dev/policy"
	ReadinessGate_PkgApprove  ConditionType = "pkg.pkgserver.dev/approve"
	ReadinessGate_PkgInstall  ConditionType = "pkg.pkgserver.dev/install"
)

// +k8s:openapi-gen=true
// ReadinessGate allows for specifying conditions for when a PackageRevision is considered ready.
type ReadinessGate struct {
	// ConditionType refers to the condition type whose status is used to determine readiness.
	ConditionType ConditionType `json:"conditionType" protobuf:"bytes,1,opt,name=conditionType"`
}

func (r ReadinessGate) GetCondition() Condition {
	return ConditionUpdate(r.ConditionType, "", "")
}
