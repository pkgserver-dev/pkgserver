package deletecmd

import (
	"context"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	"github.com/fatih/color"
	"github.com/henderiw/logger/log"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *Runner {
	r := &Runner{}
	cmd := &cobra.Command{
		Use:  "delete NAME",
		Args: cobra.ExactArgs(1),
		//Short:   docs.InitShort,
		//Long:    docs.InitShort + "\n" + docs.InitLong,
		//Example: docs.InitExamples,
		//PreRunE: r.preRunE,
		RunE: r.runE,
	}

	r.Command = cmd
	r.cfg = cfg
	r.local = *pkgctlcfg.Local

	return r
}

func NewCommand(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *cobra.Command {
	return NewRunner(ctx, version, kubeflags, pkgctlcfg).Command
}

type Runner struct {
	Command *cobra.Command
	cfg     *genericclioptions.ConfigFlags
	local   bool
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	ctx := c.Context()
	log := log.FromContext(ctx)
	log.Debug("delete secret")

	secretName := args[0]
	if r.local {

		delete(viper.Get("secrets").(map[string]interface{}), secretName)

		if err := viper.WriteConfig(); err != nil {
			color.Red("Error writing config file: %s", err.Error())
			return err
		}
	}

	return nil
}
