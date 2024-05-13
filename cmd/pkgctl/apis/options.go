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
