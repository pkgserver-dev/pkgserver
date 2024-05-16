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

package createcmd

import (
	"context"
	"fmt"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	"github.com/fatih/color"
	"github.com/henderiw/logger/log"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags) *Runner {
	r := &Runner{}
	cmd := &cobra.Command{
		Use:  "create NAME USERNAME TOKEN",
		Args: cobra.ExactArgs(3),
		//Short:   docs.InitShort,
		//Long:    docs.InitShort + "\n" + docs.InitLong,
		//Example: docs.InitExamples,
		//PreRunE: r.preRunE,
		RunE: r.runE,
	}

	r.Command = cmd
	r.kubeflags = kubeflags

	return r
}

func NewCommand(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags) *cobra.Command {
	return NewRunner(ctx, version, kubeflags).Command
}

type Runner struct {
	Command   *cobra.Command
	kubeflags *genericclioptions.ConfigFlags
	local     bool
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	ctx := c.Context()
	log := log.FromContext(ctx)
	log.Debug("create secret")

	secretName := args[0]
	secret := apis.Secret{
		Username: args[1],
		Password: args[2],
	}
	if r.local {
		viper.Set(fmt.Sprintf("secrets.%s", secretName), secret)

		if err := viper.WriteConfig(); err != nil {
			color.Red("Error writing config file: %s", err.Error())
			return err
		}
	}

	return nil
}
