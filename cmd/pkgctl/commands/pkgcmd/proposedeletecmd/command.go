package proposedeletecmd

import (
	"context"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/pkgserver-dev/pkgserver/pkg/client"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *Runner {
	r := &Runner{}
	cmd := &cobra.Command{
		Use:  "propose-delete PKGREV [flags]",
		Args: cobra.ExactArgs(1),
		//Short:   docs.InitShort,
		//Long:    docs.InitShort + "\n" + docs.InitLong,
		//Example: docs.InitExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
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
	client  client.Client
	local   bool
}

func (r *Runner) preRunE(_ *cobra.Command, _ []string) error {
	client, err := client.CreateClientWithFlags(r.cfg)
	if err != nil {
		return err
	}
	r.client = client
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	ctx := c.Context()
	//log := log.FromContext(ctx)
	//log.Info("deletepropose packagerevision", "name", args[0])

	namespace := "default"
	if r.cfg.Namespace != nil && *r.cfg.Namespace != "" {
		namespace = *r.cfg.Namespace
	}

	pkgRev := &pkgv1alpha1.PackageRevision{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: args[0]}, pkgRev); err != nil {
		return err
	}

	pkgRev.Spec.Tasks = []pkgv1alpha1.Task{}
	pkgRev.Spec.Lifecycle = pkgv1alpha1.PackageRevisionLifecycleDeletionProposed

	return r.client.Update(ctx, pkgRev)
}
