package createcmd

import (
	"context"
	"fmt"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/pkgserver-dev/pkgserver/pkg/client"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *Runner {
	r := &Runner{}
	cmd := &cobra.Command{
		Use:  "create PKGREV[<Target>.<REPO>.<REALM>.<PACKAGE>.<WORKSPACE>] [flags]",
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

	r.Command.Flags().StringVar(
		&r.source, "src", "", "source from which the pkg should be cloned")

	return r
}

func NewCommand(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags, pkgctlcfg *apis.ConfigFlags) *cobra.Command {
	return NewRunner(ctx, version, kubeflags, pkgctlcfg).Command
}

type Runner struct {
	Command *cobra.Command
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	source  string
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
	//log.Info("create packagerevision", "src", args[0], "dst", args[1])

	namespace := "default"
	if r.cfg.Namespace != nil && *r.cfg.Namespace != "" {
		namespace = *r.cfg.Namespace
	}

	var err error
	var srcPkgID *pkgid.PackageID
	if r.source != "" {
		srcPkgID, err = pkgid.ParsePkgRev2PkgID(r.source)
		if err != nil {
			return err
		}
	}

	pkgRevName := args[0]
	dstPkgID, err := pkgid.ParsePkgRev2PkgID(pkgRevName)
	if err != nil {
		return err
	}

	pkgRev := pkgv1alpha1.BuildPackageRevision(
		metav1.ObjectMeta{
			Name:      pkgRevName,
			Namespace: namespace,
		},
		pkgv1alpha1.PackageRevisionSpec{
			PackageID: *dstPkgID,
			Lifecycle: pkgv1alpha1.PackageRevisionLifecycleDraft,
			Tasks: []pkgv1alpha1.Task{
				{
					Type: pkgv1alpha1.TaskTypeInit,
				},
			},
		},
		pkgv1alpha1.PackageRevisionStatus{},
	)

	if srcPkgID != nil {
		pkgRev = pkgv1alpha1.BuildPackageRevision(
			metav1.ObjectMeta{
				Name:      pkgRevName,
				Namespace: namespace,
			},
			pkgv1alpha1.PackageRevisionSpec{
				PackageID: *dstPkgID,
				Lifecycle: pkgv1alpha1.PackageRevisionLifecycleDraft,
				Upstream: &pkgid.Upstream{
					Repository: srcPkgID.Repository,
					Realm:      srcPkgID.Realm,
					Package:    srcPkgID.Package,
					Revision:   srcPkgID.Revision,
				},
				Tasks: []pkgv1alpha1.Task{
					{
						Type: pkgv1alpha1.TaskTypeClone,
					},
				},
			},
			pkgv1alpha1.PackageRevisionStatus{},
		)
	}

	if err := r.client.Create(ctx, pkgRev); err != nil {
		return err
	}
	fmt.Println(pkgRev.Name)
	return nil
}
