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

package pullcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"
	"github.com/kform-dev/kform/pkg/fsys"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgrevid"
	"github.com/pkgserver-dev/pkgserver/pkg/client"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags) *Runner {
	r := &Runner{}
	cmd := &cobra.Command{
		Use:  "pull PKGREV[<Target>.<REPO>.<REALM>.<PACKAGE>.<WORKSPACE>] [DIR] [flags]",
		Args: cobra.MinimumNArgs(1),
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

func (r *Runner) runE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	//log := log.FromContext(ctx)
	//log.Info("create packagerevision", "src", args[0], "dst", args[1])

	namespace := "default"
	if r.kubeflags.Namespace != nil && *r.kubeflags.Namespace != "" {
		namespace = *r.kubeflags.Namespace
	}

	pkgRevName := args[0]
	if _, err := pkgrevid.ParsePkgRev2PkgRevID(pkgRevName); err != nil {
		return err
	}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      pkgRevName,
	}

	prr := &pkgv1alpha1.PackageRevisionResources{}
	if err := r.client.Get(ctx, key, prr); err != nil {
		return err
	}
	if len(args) > 1 {
		if err := writeToDir(prr.Spec.Resources, args[1]); err != nil {
			return err
		}
	} else {
		if err := writeToWriter(prr.Spec.Resources, os.Stdout); err != nil {
			return err
		}
	}
	return nil
}

func writeToDir(resources map[string]string, dir string) error {
	if !fsys.IsDir(dir) {
		return fmt.Errorf("dir %s does not exist", dir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	for k, v := range resources {
		f := filepath.Join(dir, k)
		d := filepath.Dir(f)
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(f, []byte(v), 0644); err != nil {
			return err
		}
	}
	return nil
}

func writeToWriter(resources map[string]string, w io.Writer) error {
	for k, v := range resources {
		fmt.Fprintf(w, "path: %s\n%s", k, v)
	}
	return nil
}
