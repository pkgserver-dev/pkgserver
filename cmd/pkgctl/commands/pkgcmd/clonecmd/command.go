package clonecmd

import (
	"context"
	"fmt"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	"github.com/henderiw/logger/log"
	"github.com/kform-dev/kform/pkg/pkgio"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags, k8s bool) *Runner {
	r := &Runner{}
	cmd := &cobra.Command{
		Use:  "clone PKGREV[<Target>.<REPO>.<REALM>.<PACKAGE>.<WORKSPACE>] [LOCAL_DST_DIRECTORY] [flags]",
		Args: cobra.MinimumNArgs(1),
		// pkgctl pkg clone catalog.repo-catalog.infra:workload-cluster.capi-kind.ws3 --revision v3
		//Short:   docs.InitShort,
		//Long:    docs.InitShort + "\n" + docs.InitLong,
		//Example: docs.InitExamples,
		//PreRunE: r.preRunE,
		RunE: r.runE,
	}

	r.Command = cmd
	r.cfg = cfg
	r.k8s = k8s
	r.Command.Flags().StringVar(
		&r.revision, "revision", "", "revision of the package to be cloned")

	return r
}

func NewCommand(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags, k8s bool) *cobra.Command {
	return NewRunner(ctx, version, kubeflags, k8s).Command
}

type Runner struct {
	Command  *cobra.Command
	cfg      *genericclioptions.ConfigFlags
	revision string
	k8s      bool
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

	if !r.k8s {
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
		// get packagerevision resources
		// map to the datastore and use the generic writer
	}
	return nil
}
