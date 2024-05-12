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

package packagediscovery

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/henderiw/logger/log"
	"github.com/henderiw/store"
	kformv1alpha1 "github.com/kform-dev/kform/apis/pkg/v1alpha1"
	"github.com/kform-dev/kform/pkg/pkgio"
	"github.com/kform-dev/kform/pkg/recorder"
	"github.com/kform-dev/kform/pkg/recorder/diag"
	"github.com/kform-dev/kform/pkg/syntax/parser/pkgparser"
	kformtypes "github.com/kform-dev/kform/pkg/syntax/types"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *reconciler) getPackageResources(ctx context.Context, cr *pkgv1alpha1.PackageRevision) (
	[]any,
	[]any,
	[]any,
	[]any,
	error,
) {
	log := log.FromContext(ctx)
	packages := []any{}
	resources := []any{}
	inputs := []any{}
	outputs := []any{}

	/*
		pkgRevResources, err := r.clientset.PkgV1alpha1().PackageRevisionResourceses(cr.Namespace).Get(ctx, cr.Name, v1.GetOptions{})
		if err != nil {
			log.Error("cannot list package resources", "error", err.Error())
			return packages, resources, inputs, outputs, err
		}
		if len(pkgRevResources.Spec.Resources) != 0 {
			log.Info("package resources", "total", len(pkgRevResources.Spec.Resources))
		}
	*/
	//log.Info("package resources listing for", "pkgRev", cr.Name, "namespace", cr.Namespace)
	/*
		opts := []client.ListOption{}
		pkgRevResourcesList := &pkgv1alpha1.PackageRevisionResourcesList{}
		if err := r.List(ctx, pkgRevResourcesList, opts...); err != nil {
			log.Error("cannot list package resources", "error", err.Error())
			return packages, resources, inputs, outputs, err
		}
		var pkgRevResources *pkgv1alpha1.PackageRevisionResources
		for _, pkgRevRes := range pkgRevResourcesList.Items {
			pkgRevRes := pkgRevRes
			//log.Info("package resources from list", "pkgRev", pkgRevRes.Name)
			if pkgRevRes.Name == cr.Name && pkgRevRes.Namespace == cr.Namespace {
				pkgRevResources = &pkgRevRes
				break
			}
		}
		if pkgRevResources == nil || pkgRevResources.Spec.Resources == nil {
			log.Error("cannot get package resources", "key", cr.Name)
			return packages, resources, inputs, outputs, fmt.Errorf("cannot get package revision resources for: %s", cr.Name)
		}
	*/

	key := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Name,
	}

	pkgRevResources, err := r.clientset.PkgV1alpha1().PackageRevisionResourceses(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
	if err != nil {
		log.Error("cannot get package resources", "key", key.String(), "error", err.Error())
		return packages, resources, inputs, outputs, err
	}
	/*
		pkgRevResources := &pkgv1alpha1.PackageRevisionResources{}
		if err := r.Get(ctx, pkgRevKey, pkgRevResources, &client.GetOptions{Raw: &v1.GetOptions{ResourceVersion: "0"}}); err != nil {
			log.Error("cannot get package resources", "key", pkgRevKey.String(), "error", err.Error())
			return packages, resources, inputs, outputs, err
		}
	*/

	pkgRecorder := recorder.New[diag.Diagnostic]()
	ctx = context.WithValue(ctx, kformtypes.CtxKeyRecorder, pkgRecorder)
	p, err := pkgparser.New(ctx, filepath.Base(cr.Spec.PackageID.Package))
	if err != nil {
		log.Error("cannot get package parser", "error", err.Error())
		return packages, resources, inputs, outputs, err
	}

	/*
		kf, kforms, err := loadkforms(ctx, pkgRevResources.Spec.Resources)
		if err != nil {
			log.Error("cannot load kforms", "error", err.Error())
			return packages, resources, inputs, outputs, err
		}
	*/

	reader := pkgio.KformMemReader{
		Resources: pkgRevResources.Spec.Resources,
	}
	kforms, err := reader.Read(ctx)
	if err != nil {
		log.Error("cannot reader package resource content")
	}

	pkg := p.Parse(ctx, kforms)
	if pkgRecorder.Get().HasError() {
		log.Error("kform parsin failed", "error", pkgRecorder.Get().Error().Error())
		// if an error is found we stop processing
		return packages, resources, inputs, outputs, pkgRecorder.Get().Error()
	}

	pkg.Blocks.List(ctx, func(ctx context.Context, key store.Key, block kformtypes.Block) {
		if strings.HasPrefix(key.Name, kformv1alpha1.BlockTYPE_OUTPUT.String()) {
			for _, rn := range block.GetData() {
				outputs = append(outputs, rn)
			}
		}
		if strings.HasPrefix(key.Name, kformv1alpha1.BlockTYPE_OUTPUT.String()) ||
			strings.HasPrefix(key.Name, kformv1alpha1.BlockTYPE_RESOURCE.String()) ||
			strings.HasPrefix(key.Name, kformv1alpha1.BlockTYPE_LIST.String()) ||
			strings.HasPrefix(key.Name, kformv1alpha1.BlockTYPE_DATA.String()) {
			for _, rn := range block.GetData() {
				resources = append(resources, rn)
			}

		}
		if strings.HasPrefix(key.Name, kformv1alpha1.BlockTYPE_PACKAGE.String()) {
			for _, rn := range block.GetData() {
				packages = append(packages, rn)
			}
		}

		if strings.HasPrefix(key.Name, kformv1alpha1.BlockTYPE_INPUT.String()) {
			for _, rn := range block.GetData() {
				inputs = append(inputs, rn)
			}
		}
	})
	return packages, resources, inputs, outputs, nil
}

/*
func loadkforms(ctx context.Context, resources map[string]string) (*kformv1alpha1.KformFile, map[string]*fn.KubeObject, error) {
	recorder := cctx.GetContextValue[recorder.Recorder[diag.Diagnostic]](ctx, kformtypes.CtxKeyRecorder)
	if recorder == nil {
		return nil, nil, fmt.Errorf("cannot load files w/o a recorder")
	}
	log := log.FromContext(ctx)
	var kfile *kformv1alpha1.KformFile
	kforms := map[string]*fn.KubeObject{}

	splitResources := map[string]string{}
	for fileName, data := range resources {
		// Replace the ending \r\n (line ending used in windows) with \n and then split it into multiple YAML documents
		// if it contains document separators (---)
		values, err := splitDocuments(strings.ReplaceAll(string(data), "\r\n", "\n"))
		if err != nil {
			return nil, nil, err
		}
		for i := range values {
			// the Split used above will eat the tail '\n' from each resource. This may affect the
			// literal string value since '\n' is meaningful in it.
			if i != len(values)-1 {
				values[i] += "\n"
			}
			splitResources[fmt.Sprintf("%s.%d", fileName, i)] = string(values[i])
		}
	}

	for fileName, data := range splitResources {
		ko, err := fn.ParseKubeObject([]byte(data))
		if err != nil {
			recorder.Record(diag.DiagErrorf("kubeObject parsing failed, path: %s, err: %s", fileName, err.Error()))
			continue
		}
		if ko.GetKind() == reflect.TypeOf(kformv1alpha1.KformFile{}).Name() {
			if kfile != nil {
				recorder.Record(diag.DiagErrorf("cannot have 2 kform file resource in the package"))
				continue
			}
			kfKOE, err := koe.NewFromKubeObject[kformv1alpha1.KformFile](ko)
			if err != nil {
				recorder.Record(diag.DiagFromErr(err))
				continue
			}
			kfile, err = kfKOE.GetGoStruct()
			if err != nil {
				recorder.Record(diag.DiagFromErr(err))
				continue
			}
		} else {
			//ko.GetAnnotations() TBD do we need to look at annotations

			kforms[fileName] = ko
			log.Debug("read kubeObject", "fileName", fileName, "kind", ko.GetKind(), "name", ko.GetName())
		}
	}
	return kfile, kforms, nil
}
*/

/*
// splitDocuments returns a slice of all documents contained in a YAML string. Multiple documents can be divided by the
// YAML document separator (---). It allows for white space and comments to be after the separator on the same line,
// but will return an error if anything else is on the line.
func splitDocuments(s string) ([]string, error) {
	docs := make([]string, 0)
	if len(s) > 0 {
		// The YAML document separator is any line that starts with ---
		yamlSeparatorRegexp := regexp.MustCompile(`\n---.*\n`)

		// Find all separators, check them for invalid content, and append each document to docs
		separatorLocations := yamlSeparatorRegexp.FindAllStringIndex(s, -1)
		prev := 0
		for i := range separatorLocations {
			loc := separatorLocations[i]
			separator := s[loc[0]:loc[1]]

			// If the next non-whitespace character on the line following the separator is not a comment, return an error
			trimmedContentAfterSeparator := strings.TrimSpace(separator[4:])
			if len(trimmedContentAfterSeparator) > 0 && trimmedContentAfterSeparator[0] != '#' {
				return nil, errors.Errorf("invalid document separator: %s", strings.TrimSpace(separator))
			}

			docs = append(docs, s[prev:loc[0]])
			prev = loc[1]
		}
		docs = append(docs, s[prev:])
	}

	return docs, nil
}
*/
