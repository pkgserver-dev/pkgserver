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

package pkgcmd

import (
	"context"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd/approvecmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd/clonecmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd/createcmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd/deletecmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd/draftcmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd/getcmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd/listcmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd/proposecmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd/proposedeletecmd"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands/pkgcmd/pushcmd"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func GetCommand(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use: "pkg",
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
		approvecmd.NewCommand(ctx, version, cfg, pkgctlcfg),
		clonecmd.NewCommand(ctx, version, cfg, pkgctlcfg),
		createcmd.NewCommand(ctx, version, cfg, pkgctlcfg),
		deletecmd.NewCommand(ctx, version, cfg, pkgctlcfg),
		draftcmd.NewCommand(ctx, version, cfg, pkgctlcfg),
		getcmd.NewCommand(ctx, version, cfg, pkgctlcfg),
		listcmd.NewCommand(ctx, version, cfg, pkgctlcfg),
		proposecmd.NewCommand(ctx, version, cfg, pkgctlcfg),
		proposedeletecmd.NewCommand(ctx, version, cfg, pkgctlcfg),
		pushcmd.NewCommand(ctx, version, cfg, pkgctlcfg),
	)
	return cmd
}
