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

package pkgrevid

type PkgState int64

const (
	PkgState_NotAvailable PkgState = iota
	PkgState_Scheduled
	PkgState_Processed
	PkgState_Validated // policy
	PkgState_Approved
	PkgState_Installed
)

func (r PkgState) String() string {
	switch r {
	case PkgState_NotAvailable:
		return "notAvailable"
	case PkgState_Scheduled:
		return "scheduled"
	case PkgState_Processed:
		return "processed"
	case PkgState_Approved:
		return "approved"
	case PkgState_Installed:
		return "installed"
	}
	return "unknown"
}
