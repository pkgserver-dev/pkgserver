package repocmd

import (
	"context"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/repocmd/createcmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/repocmd/deletecmd"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func GetCommand(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use: "repo",
		//Short:   docs.InitShort,
		//Long:    docs.InitShort + "\n" + docs.InitLong,
		//Example: docs.InitExamples,
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := cmd.Flags().GetBool("help")
			if err != nil {
				return err
			}
			if h {
				return cmd.Help()
			}
			return cmd.Usage()
		},
	}

	cmd.AddCommand(
		createcmd.NewCommand(ctx, version, cfg, pkgctlcfg),
		deletecmd.NewCommand(ctx, version, cfg, pkgctlcfg),
	)
	return cmd
}
