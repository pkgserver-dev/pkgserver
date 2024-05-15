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

package clonecmd

import (
	"context"
	"fmt"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	"github.com/henderiw/logger/log"
	"github.com/kform-dev/kform/pkg/pkgio"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/pkgserver-dev/pkgserver/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *Runner {
	r := &Runner{}
	cmd := &cobra.Command{
		Use:  "clone PKGREV[<Target>.<REPO>.<REALM>.<PACKAGE>.<WORKSPACE>] [LOCAL_DST_DIRECTORY] [flags]",
		Args: cobra.MinimumNArgs(1),
		// pkgctl pkg clone catalog.repo-catalog.infra:workload-cluster.capi-kind.ws3 --revision v3
		//Short:   docs.InitShort,
		//Long:    docs.InitShort + "\n" + docs.InitLong,
		//Example: docs.InitExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}

	r.Command = cmd
	r.cfg = cfg
	r.local = *pkgctlcfg.Local
	r.Command.Flags().StringVar(
		&r.revision, "revision", "", "revision of the package to be cloned")

	return r
}

func NewCommand(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *cobra.Command {
	return NewRunner(ctx, version, kubeflags, pkgctlcfg).Command
}

type Runner struct {
	Command  *cobra.Command
	cfg      *genericclioptions.ConfigFlags
	client   client.Client
	revision string
	local      bool
}

func (r *Runner) preRunE(_ *cobra.Command, _ []string) error {
	if !r.local {
		client, err := client.CreateClientWithFlags(r.cfg)
		if err != nil {
			return err
		}
		r.client = client
	}
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	ctx := c.Context()
	log := log.FromContext(ctx)
	log.Debug("clone packagerevision", "name", args[0])

	pkgRevName := args[0]
	pkgID, err := pkgid.ParsePkgRev2PkgID(pkgRevName)
	if err != nil {
		return err
	}
	dir := pkgID.Package
	if len(args) > 2 {
		dir = args[2]
	}

	if r.local {
		repoName := pkgID.Repository
		var repo apis.Repo
		if err := viper.UnmarshalKey(fmt.Sprintf("repos.%s", repoName), &repo); err != nil {
			return err
		}
		if r.revision != "" {
			pkgID.Revision = r.revision
		}
		/*
			var tag string
			branch := pkgID.Branch(catalog)
			if r.revision != "" {
				pkgID.Revision = r.revision
				tag = pkgID.Tag(catalog)
			}
		*/

		reader := pkgio.GitReader{
			URL:        repo.URL,
			Secret:     repo.Secret,
			Deployment: repo.Deployment,
			Directory:  repo.Directory,
			PkgID:      pkgID,
			PkgPath:    dir,
		}
		datastore, err := reader.Read(ctx)
		if err != nil {
			return err
		}
		w := pkgio.ByteWriter{
			Type: pkgio.OutputSink_Dir,
			Path: dir,
		}

		return w.Write(ctx, datastore)
	} else {
		r.validateUpstream(ctx, pkgID)
		// map to the datastore and use the generic writer
	}
	return nil
}

func (r *Runner) validateUpstream(ctx context.Context, pkgID *pkgid.PackageID) error {

	pkgRevList := pkgv1alpha1.PackageRevisionList{}
	if err := r.client.List(ctx, &pkgRevList); err != nil {
		return err
	}

	for _, pkgRev := range pkgRevList.Items {
		if pkgRev.Spec.PackageID.Repository == pkgID.Repository &&
			pkgRev.Spec.PackageID.Package == pkgID.Package &&
			pkgRev.Spec.PackageID.Revision == pkgID.Revision {
			if pkgRev.GetCondition(condition.ConditionTypeReady).Status == metav1.ConditionTrue {
				return nil
			}
			return fmt.Errorf("pkg %s not ready", pkgRev.Name)
		}
	}
	return fmt.Errorf("upstream pkg %s not found", pkgID.PkgRevString())
}
