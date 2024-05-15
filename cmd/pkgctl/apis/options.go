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

package apis

import (
	"github.com/spf13/pflag"
	"k8s.io/utils/ptr"
)

const (
	flagLocal = "local"
)

type ConfigFlags struct {
	Local               *bool
	usePersistentConfig bool
}

func NewConfigFlags(usePersistentConfig bool) *ConfigFlags {
	return &ConfigFlags{
		Local:                 ptr.To[bool](false),
		usePersistentConfig: usePersistentConfig,
	}
}

func (f *ConfigFlags) AddFlags(flags *pflag.FlagSet) {
	if f.Local != nil {
		flags.BoolVar(f.Local, flagLocal, *f.Local, "If true, execute the command locally, by default it is executed on the k8s cluster where the pkgserver is installed")
	}
}
