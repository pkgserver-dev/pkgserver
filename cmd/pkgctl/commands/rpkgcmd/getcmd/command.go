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

package getcmd

import (
	"context"
	"fmt"
	"strings"

	//docs "github.com/pkgserver-dev/pkgserver/internal/docs/generated/initdocs"

	"github.com/henderiw/logger/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/get"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, version string, cfg *genericclioptions.ConfigFlags) *Runner {
	r := &Runner{
		printFlags: get.NewGetPrintFlags(),
	}
	cmd := &cobra.Command{
		Use:     "get [flags]",
		Aliases: []string{"list"},
		//Args: cobra.ExactArgs(1),
		//Short:   docs.InitShort,
		//Long:    docs.InitShort + "\n" + docs.InitLong,
		//Example: docs.InitExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}

	r.Command = cmd
	r.cfg = cfg

	cmd.Flags().StringVar(&r.target, "target", "", "Target of the packages to get, Any package whose target matches this value will be included in the result")
	cmd.Flags().StringVar(&r.realm, "realm", "", "Realm of the packages to get, Any package whose realm matches this value will be included in the result")
	cmd.Flags().StringVar(&r.packageName, "name", "", "Name of the packages to get, Any package whose name matches this value will be included in the result")
	cmd.Flags().StringVar(&r.revision, "revision", "", "Revision of the package to get, Any package whose revision matches this value will be included in the result")
	cmd.Flags().StringVar(&r.workspace, "workspace", "", "WorkspaceName of the packages to get, Any package whose workspace matches this value will be included in the result")

	r.printFlags.AddFlags(cmd)
	return r
}

func NewCommand(ctx context.Context, version string, kubeflags *genericclioptions.ConfigFlags) *cobra.Command {
	return NewRunner(ctx, version, kubeflags).Command
}

type Runner struct {
	Command *cobra.Command
	cfg     *genericclioptions.ConfigFlags
	//client  client.Client
	local bool

	// flags
	target      string
	realm       string
	packageName string
	revision    string
	workspace   string
	printFlags  *get.PrintFlags

	requestTable bool
}

func (r *Runner) preRunE(cmd *cobra.Command, _ []string) error {
	outputOption := cmd.Flags().Lookup("output").Value.String()
	if strings.Contains(outputOption, "custom-columns") || outputOption == "yaml" || strings.Contains(outputOption, "json") {
		r.requestTable = false
	} else {
		r.requestTable = true
	}
	return nil
}

func (r *Runner) runE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	log := log.FromContext(ctx)
	//log.Info("get packagerevision", "name", args[0])

	var objs []runtime.Object

	namespace := "default"
	if r.cfg.Namespace != nil && *r.cfg.Namespace != "" {
		namespace = *r.cfg.Namespace
	}

	fmt.Println("reguestTable", r.requestTable)

	b := resource.NewBuilder(r.cfg).NamespaceParam(namespace)
	if r.requestTable {
		scheme := runtime.NewScheme()
		// Accept PartialObjectMetadata and Table
		if err := metav1.AddMetaToScheme(scheme); err != nil {
			return fmt.Errorf("error building runtime.Scheme: %w", err)
		}
		b = b.WithScheme(scheme, schema.GroupVersion{Version: "v1"})
	} else {
		b = b.Unstructured()
	}

	useSelectors := true
	if len(args) > 0 {
		b = b.ResourceNames("packagerevisions", args...)
		// We can't pass selectors here, get an error "Error: selectors and the all flag cannot be used when passing resource/name arguments"
		// TODO: cli-utils bug?  I think there is a metadata.name field selector (used for single object watch)
		useSelectors = false
	} else {
		b = b.ResourceTypes("packagerevisions")
	}
	if useSelectors {
		fieldSelector := fields.Everything()
		if r.target != "" {
			fieldSelector = fields.OneTermEqualSelector("spec.packageID.target", r.target)
		}
		if r.realm != "" {
			fieldSelector = fields.OneTermEqualSelector("spec.packageID.realm", r.realm)
		}
		if r.revision != "" {
			fieldSelector = fields.OneTermEqualSelector("spec.packageID.revision", r.revision)
		}
		if r.workspace != "" {
			fieldSelector = fields.OneTermEqualSelector("spec.packageID.workspaceName", r.workspace)
		}
		if r.packageName != "" {
			fieldSelector = fields.OneTermEqualSelector("spec.packageID.packageName", r.packageName)
		}
		if s := fieldSelector.String(); s != "" {
			b = b.FieldSelectorParam(s)
		} else {
			b = b.SelectAllParam(true)
		}
	}

	b = b.ContinueOnError().
		Latest().
		Flatten()

	if r.requestTable {
		b = b.TransformRequests(func(req *rest.Request) {
			req.SetHeader("Accept", strings.Join([]string{
				"application/json;as=Table;g=meta.k8s.io;v=v1",
				"application/json",
			}, ","))
		})
	}

	res := b.Do()
	if err := res.Err(); err != nil {
		return err
	}

	infos, err := res.Infos()
	if err != nil {
		return err
	}

	for _, i := range infos {
		if table, ok := i.Object.(*metav1.Table); ok {
			for i := range table.Rows {
				row := &table.Rows[i]
				if row.Object.Object == nil && row.Object.Raw != nil {
					u := &unstructured.Unstructured{}
					if err := u.UnmarshalJSON(row.Object.Raw); err != nil {
						log.Error("error parsing raw object", "error", err)
					}
					row.Object.Object = u
				}
			}
		}
	}

	for _, i := range infos {
		switch obj := i.Object.(type) {
		case *unstructured.Unstructured:
			objs = append(objs, obj)
		case *metav1.Table:
			objs = append(objs, obj)
		default:
			return fmt.Errorf("unrecognized response %T", obj)
		}
	}

	printer, err := r.printFlags.ToPrinter()
	if err != nil {
		return err
	}

	w := printers.GetNewTabWriter(cmd.OutOrStdout())
	for _, obj := range objs {
		if err := printer.PrintObj(obj, w); err != nil {
			fmt.Println("test")
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil

	/*
		pkgRev := &pkgv1alpha1.PackageRevision{}
		if err := r.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: args[0]}, pkgRev); err != nil {
			return err
		}
		b, err := yaml.Marshal(pkgRev)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	*/
}
