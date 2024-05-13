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

package packagerevisionresource

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/henderiw/logger/log"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type prr struct {
	prr *pkgv1alpha1.PackageRevisionResources
	err error
}

func (r *strategy) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	log := log.FromContext(ctx)
	log.Info("list")
	filter, err := parsePackageFieldSelector(ctx, options.FieldSelector)
	if err != nil {
		return nil, err
	}

	opts := []client.ListOption{}
	if options.LabelSelector != nil {
		opts = append(opts, client.MatchingLabelsSelector{Selector: options.LabelSelector})
	}

	//	if options.FieldSelector != nil {
	//		requirements := options.FieldSelector.Requirements()
	//		for _, req := range requirements {
	//			log.Info("list field selector requirements", "operator", req.Operator, "field", req.Field, "value", req.Value)
	//			opts = append(opts, client.MatchingFields{req.Field: req.Value})
	//		}
	//	}

	pkgRevs := pkgv1alpha1.PackageRevisionList{}
	if err := r.client.List(ctx, &pkgRevs, opts...); err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	newListObj := &pkgv1alpha1.PackageRevisionResourcesList{}
	v, err := getListPrt(newListObj)
	if err != nil {
		return nil, err
	}

	prrChan := make(chan prr)
	var wg sync.WaitGroup
	for _, pkgRev := range pkgRevs.Items {
		pkgRev := pkgRev
		if pkgRev.GetCondition(condition.ConditionTypeReady).Status == metav1.ConditionFalse {
			// the package might not be cloned yet
			continue
		}
		if filter != nil {
			if filter.Name != "" && filter.Name != pkgRev.Name {
				continue
			}
			if filter.Namespace != "" && filter.Namespace != pkgRev.Namespace {
				continue
			}
			if filter.Target != "" && filter.Target != pkgRev.Spec.PackageID.Target {
				continue
			}
			if filter.Repository != "" && filter.Repository != pkgRev.Spec.PackageID.Repository {
				continue
			}
			if filter.Realm != "" && filter.Realm != pkgRev.Spec.PackageID.Realm {
				continue
			}
			if filter.Package != "" && filter.Package != pkgRev.Spec.PackageID.Package {
				continue
			}
			if filter.Revision != "" && filter.Revision != pkgRev.Spec.PackageID.Revision {
				continue
			}
			if filter.Workspace != "" && filter.Workspace != pkgRev.Spec.PackageID.Workspace {
				continue
			}
			if filter.Lifecycle != "" && filter.Lifecycle != string(pkgRev.Spec.Lifecycle) {
				continue
			}
		}
		wg.Add(1)
		go func() error {
			defer wg.Done()
			repo, err := r.getRepository(ctx, types.NamespacedName{Name: pkgid.GetRepoNameFromPkgRevName(pkgRev.Name), Namespace: pkgRev.Namespace})
			if err != nil {
				prrChan <- prr{
					prr: nil,
					err: err,
				}
			}
			cachedRepo, err := r.cache.Open(ctx, repo)
			if err != nil {
				prrChan <- prr{
					prr: nil,
					err: err,
				}
				//return apierrors.NewInternalError(err)
			}
			resources, err := cachedRepo.GetResources(ctx, &pkgRev, false)
			if err != nil {
				prrChan <- prr{
					prr: nil,
					err: err,
				}
				//return apierrors.NewInternalError(err)
			}
			prrChan <- prr{
				prr: buildPackageRevisionResources(&pkgRev, resources),
				err: nil,
			}
			return nil
		}()
	}
	go func() {
		// Wait for all goroutines to finish
		wg.Wait()

		// Close the error channel to indicate that no more errors will be sent
		close(prrChan)
	}()
	var errm error
	for prr := range prrChan {
		prr := prr
		if prr.err != nil {
			errm  = errors.Join(errm, err)
			continue
		}
		appendItem(v, prr.prr)
		
	}
	if errm != nil {
		return nil, apierrors.NewInternalError(errm)
	}

	// sort the list
	sort.SliceStable(newListObj.Items, func(i, j int) bool {
		return newListObj.Items[i].Name < newListObj.Items[j].Name
	})
	return newListObj, nil
}

func getListPrt(listObj runtime.Object) (reflect.Value, error) {
	listPtr, err := meta.GetItemsPtr(listObj)
	if err != nil {
		return reflect.Value{}, err
	}
	v, err := conversion.EnforcePtr(listPtr)
	if err != nil || v.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("need ptr to slice: %v", err)
	}
	return v, nil
}

func appendItem(v reflect.Value, obj runtime.Object) {
	v.Set(reflect.Append(v, reflect.ValueOf(obj).Elem()))
}
