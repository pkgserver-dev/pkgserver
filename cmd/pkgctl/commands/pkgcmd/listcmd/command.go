package listcmd

import (
	"context"
	"fmt"
	"sort"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/pkgserver-dev/pkgserver/pkg/client"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *Runner {
	r := &Runner{}
	cmd := &cobra.Command{
		Use:  "list [flags]",
		Args: cobra.ExactArgs(0),
		//Short:   docs.InitShort,
		//Long:    docs.InitShort + "\n" + docs.InitLong,
		//Example: docs.InitExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}

	r.Command = cmd
	r.cfg = cfg
	r.local = *pkgctlcfg.Local
	/*
		r.Command.Flags().StringVar(
			&r.FnConfigDir, "fn-config-dir", "", "dir where the function config files are located")
	*/
	return r
}

func NewCommand(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *cobra.Command {
	return NewRunner(ctx, version, kubeflags, pkgctlcfg).Command
}

type Runner struct {
	Command *cobra.Command
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	local   bool
}

func (r *Runner) preRunE(_ *cobra.Command, _ []string) error {
	if !r.local {
		client, err := client.CreateClientWithFlags(r.cfg)
		if err != nil {
			return err
		}
		r.client = client
	} else {
		return fmt.Errorf("this command only excecutes on k8s, got: %t", r.local)
	}
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	ctx := c.Context()
	//log := log.FromContext(ctx)
	//log.Info("get packagerevision", "name", args[0])

	pkgRevList := &pkgv1alpha1.PackageRevisionList{}
	if err := r.client.List(ctx, pkgRevList); err != nil {
		return err
	}
	pkgRevs := make([]pkgv1alpha1.PackageRevision, len(pkgRevList.Items))
	copy(pkgRevs, pkgRevList.Items)
	sort.SliceStable(pkgRevs, func(i, j int) bool {
		return pkgRevs[i].Name < pkgRevs[j].Name
	})

	for _, pkgRev := range pkgRevs {
		fmt.Println(pkgRev.Name, pkgRev.Spec.PackageID.Package, pkgRev.Spec.PackageID.Workspace, pkgRev.Spec.PackageID.Revision, pkgRev.Spec.Lifecycle)
	}

	return nil
}
