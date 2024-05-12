package deletecmd

import (
	"context"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	"github.com/fatih/color"
	"github.com/henderiw/logger/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags, k8s bool) *Runner {
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
	r.k8s = k8s

	return r
}

func NewCommand(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags, k8s bool) *cobra.Command {
	return NewRunner(ctx, version, kubeflags, k8s).Command
}

type Runner struct {
	Command *cobra.Command
	cfg     *genericclioptions.ConfigFlags
	k8s     bool
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	ctx := c.Context()
	log := log.FromContext(ctx)
	log.Debug("delete secret")

	secretName := args[0]
	if !r.k8s {

		delete(viper.Get("secrets").(map[string]interface{}), secretName)

		if err := viper.WriteConfig(); err != nil {
			color.Red("Error writing config file: %s", err.Error())
			return err
		}
	}

	return nil
}
