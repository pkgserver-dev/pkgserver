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

package updatestatuscmd

import (
	"context"
	"fmt"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/pkg/client"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags) *Runner {
	r := &Runner{}
	cmd := &cobra.Command{
		Use:  "updatestatus PKGREV LIFECYCLE [flags]",
		Args: cobra.ExactArgs(2),
		//Short:   docs.InitShort,
		//Long:    docs.InitShort + "\n" + docs.InitLong,
		//Example: docs.InitExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
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
	client    client.Client
}

func (r *Runner) preRunE(_ *cobra.Command, _ []string) error {
	client, err := client.CreateClientWithFlags(r.kubeflags)
	if err != nil {
		return err
	}
	r.client = client

	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	ctx := c.Context()
	//log := log.FromContext(ctx)
	//log.Info("draft packagerevision", "name", args[0])

	lifecycle := args[1]
	if !matchLifecycle(args[1]) {
		return fmt.Errorf("please provide a valid lifecycle, got %s, supported lifecycles are %v", lifecycle, pkgv1alpha1.PackageRevisionLifecycles)
	}

	namespace := "default"
	if r.kubeflags.Namespace != nil && *r.kubeflags.Namespace != "" {
		namespace = *r.kubeflags.Namespace
	}

	pkgRev := &pkgv1alpha1.PackageRevision{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: args[0]}, pkgRev); err != nil {
		return err
	}

	pkgRev.Spec.Tasks = []pkgv1alpha1.Task{}
	pkgRev.Spec.Lifecycle = pkgv1alpha1.PackageRevisionLifecycle(lifecycle)

	return r.client.Update(ctx, pkgRev)
}

func matchLifecycle(input string) bool {
	for _, lifecycle := range pkgv1alpha1.PackageRevisionLifecycles {
		if string(lifecycle) == input {
			return true
		}
	}
	return false
}
