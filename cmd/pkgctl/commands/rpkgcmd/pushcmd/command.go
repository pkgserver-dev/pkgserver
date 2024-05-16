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

package pushcmd

import (
	"context"
	"fmt"
	"os"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	"github.com/kform-dev/kform/pkg/fsys"
	"github.com/kform-dev/kform/pkg/pkgio"
	"github.com/kform-dev/kform/pkg/pkgio/ignore"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/apis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags) *Runner {
	r := &Runner{}
	cmd := &cobra.Command{
		Use:  "push PKGREV[<Target>.<REPO>.<REALM>.<PACKAGE>.<WORKSPACE>] [LOCAL_SRC_DIRECTORY] [flags]",
		Args: cobra.MinimumNArgs(1),
		//Short:   docs.InitShort,
		//Long:    docs.InitShort + "\n" + docs.InitLong,
		//Example: docs.InitExamples,
		//PreRunE: r.preRunE,
		RunE: r.runE,
	}

	r.Command = cmd
	r.cfg = cfg

	return r
}

func NewCommand(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags) *cobra.Command {
	return NewRunner(ctx, version, kubeflags).Command
}

type Runner struct {
	Command *cobra.Command
	cfg     *genericclioptions.ConfigFlags
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	ctx := c.Context()
	//log := log.FromContext(ctx)
	//log.Info("approve packagerevision", "name", args[0])

	pkgRevName := args[0]
	pkgID, err := pkgid.ParsePkgRev2PkgID(pkgRevName)
	if err != nil {
		return err
	}

	repoName := pkgID.Repository
	var repo apis.Repo
	if err := viper.UnmarshalKey(fmt.Sprintf("repos.%s", repoName), &repo); err != nil {
		return err
	}

	dir := pkgID.Package
	if len(args) > 2 {
		dir = args[2]
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("cannot push package, directory %s does not exist", dir)
	}
	if !fi.IsDir() {
		return fmt.Errorf("cannot push package, the path %s must be a directory", dir)
	}
	fsys := fsys.NewDiskFS(dir)
	ignoreRules := ignore.Empty(".git")
	f, err := fsys.Open(pkgio.IgnoreFileMatch[0])
	if err == nil {
		// if an error is return the rules is empty, so we dont have to worry about the error
		ignoreRules, _ = ignore.Parse(f)
	}
	reader := pkgio.DirReader{
		Path:           ".", // relative path to fsys
		Fsys:           fsys,
		MatchFilesGlob: pkgio.MatchAll,
		IgnoreRules:    ignoreRules,
		SkipDir:        false,
	}
	datastore, err := reader.Read(ctx)
	if err != nil {
		return err
	}

	writer := pkgio.GitWriter{
		URL:        repo.URL,
		Secret:     repo.Secret,
		Deployment: repo.Deployment,
		Directory:  repo.Directory,
		PkgID:      pkgID,
		PkgPath:    dir,
	}
	if err := writer.Write(ctx, datastore); err != nil {
		return err
	}

	// TODO push via k8s
	// create revision w/o resources
	// update package revisions resources with the resources

	return nil
}
