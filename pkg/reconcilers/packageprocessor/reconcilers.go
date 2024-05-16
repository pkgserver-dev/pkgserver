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

package packageprocessor

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/henderiw/logger/log"
	"github.com/henderiw/resource"
	"github.com/henderiw/store"
	"github.com/henderiw/store/memory"
	"github.com/kform-dev/kform/pkg/exec/kform/runner"
	"github.com/kform-dev/kform/pkg/inventory/config"
	"github.com/pkg/errors"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	"github.com/pkgserver-dev/pkgserver/apis/generated/clientset/versioned"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgrevid"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/ctrlconfig"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/lease"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubectl/pkg/cmd/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func init() {
	reconcilers.Register("packageprocessor", &reconciler{})
}

const (
	controllerName = "PackageProcessorController"
	//controllerEventError = "PkgRevInstallerError"
	controllerEvent = "PackageProcessor"
	finalizer       = "packageprocessor.pkg.pkgserver.dev/finalizer"
	// errors
	errGetCr            = "cannot get cr"
	errUpdateStatus     = "cannot update status"
	controllerCondition = condition.ReadinessGate_PkgProcess
)

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}
	/*
		if err := configv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
			return nil, err
		}
	*/
	//r.APIPatchingApplicator = resource.NewAPIPatchingApplicator(mgr.GetClient())
	r.Client = mgr.GetClient()
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)
	r.recorder = mgr.GetEventRecorderFor(controllerName)
	r.clientset = cfg.ClientSet

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&pkgv1alpha1.PackageRevision{}).
		Complete(r)
}

type reconciler struct {
	client.Client
	finalizer *resource.APIFinalizer
	recorder  record.EventRecorder
	clientset *versioned.Clientset
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = ctrlconfig.InitContext(ctx, controllerName, req.NamespacedName)
	log := log.FromContext(ctx)
	log.Info("reconcile")

	cr := &pkgv1alpha1.PackageRevision{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			log.Error(errGetCr, "error", err)
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetCr)
		}
		return ctrl.Result{}, nil
	}
	key := types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      cr.GetName(),
	}
	cr = cr.DeepCopy()
	// if the pkgRev is a catalog packageRevision or
	// if the pkgrev does not have a condition to process the pkgRev
	// this event is not relevant
	if strings.HasPrefix(cr.GetName(), pkgrevid.PkgTarget_Catalog) ||
		!cr.HasReadinessGate(controllerCondition) {
		return ctrl.Result{}, nil
	}
	// the package revisioner first need to act e.g. to clone the repo
	// when the pkgRev is getting deleted this does not matter
	if cr.GetDeletionTimestamp().IsZero() && cr.GetCondition(condition.ConditionTypeReady).Status == metav1.ConditionFalse {
		return ctrl.Result{}, nil
	}
	// the previous condition is not finished, so we need to wait for acting
	// when the pkgRev is getting deleted this does not matter
	if cr.GetDeletionTimestamp().IsZero() && cr.GetPreviousCondition(controllerCondition).Status == metav1.ConditionFalse {
		return ctrl.Result{}, nil
	}

	l := lease.New(r.Client, key)
	if err := l.AcquireLease(ctx, controllerName); err != nil {
		log.Debug("cannot acquire lease", "key", key.String(), "error", err.Error())
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"lease", "error %s", err.Error())
		return ctrl.Result{Requeue: true, RequeueAfter: lease.RequeueInterval}, nil
	}
	r.recorder.Eventf(cr, corev1.EventTypeWarning,
		"lease", "acquired")

	if !cr.GetDeletionTimestamp().IsZero() {
		log.Info("deleting...")
		if cr.Spec.PackageRevID.Revision != "" && cr.Spec.Lifecycle != pkgv1alpha1.PackageRevisionLifecycleDeletionProposed {
			log.Info("cannot delete the kform resources since the package was released and the paackage is not in deletionProposed state")
			return ctrl.Result{}, nil
		}

		// When the packageRevision gets deleted we need to cleanup the resources
		// Kform created, the rest will be taken care of by gitops tooling e.g.
		// so no need to update the package resources and inventory
		resourceData, inventoryExists, localInventory, _, err := r.gatherPackageresources(ctx, cr)
		if err != nil {
			r.handleError(ctx, cr, "cannot get pkgRevResources", err)
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}
		if inventoryExists {
			// run kform with the destroy flag to true
			_, err := r.runkform(ctx, cr, localInventory, memory.NewStore[[]byte](), resourceData, true)
			if err != nil {
				r.handleError(ctx, cr, "kform run failed", err)
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
			// No need to update the output if the cleanup was successfull since the package reference will be deleted
		}

		// remove the finalizer
		if err := r.finalizer.RemoveFinalizer(ctx, cr); err != nil {
			//log.Error("cannot remove finalizer", "error", err)
			r.recorder.Eventf(cr, corev1.EventTypeWarning,
				controllerEvent, "error %s", err.Error())
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, nil
	}

	if err := r.finalizer.AddFinalizer(ctx, cr); err != nil {
		log.Error("cannot add finalizer", "error", err)
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			controllerEvent, "error %s", err.Error())
		cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot add finalizer", err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	// process resource data
	// run kform
	// determine error
	// create output
	if cr.GetCondition(controllerCondition).Status == metav1.ConditionFalse {
		resourceData, inventoryExists, localInventory, localInventoryString, err := r.gatherPackageresources(ctx, cr)
		if err != nil {
			r.handleError(ctx, cr, "cannot get pkgRevResources", err)
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}

		// process input
		inputData := gatherInput(ctx, cr)

		// run kform with destry flag to false
		outputData, err := r.runkform(ctx, cr, localInventory, inputData, resourceData, false)
		if err != nil {
			r.handleError(ctx, cr, "kform run failed", err)
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}

		if err := r.updatePackageRevisionResources(ctx, cr, outputData, inventoryExists, localInventoryString); err != nil {
			r.handleError(ctx, cr, "cannot update pkgRevResources", err)
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}

		log.Debug("kform processor complete")
		r.recorder.Eventf(cr, corev1.EventTypeNormal,
			controllerEvent, "ready")
		cr.SetConditions(condition.ConditionReady(controllerCondition))
		cType := cr.NextReadinessGate(controllerCondition)
		if cType != condition.ConditionTypeEnd {
			if !cr.HasCondition(cType) {
				cr.SetConditions(condition.ConditionUpdate(cType, "init", ""))
			}
		}
		return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}
	return ctrl.Result{}, nil
}

func (r *reconciler) handleError(ctx context.Context, cr *pkgv1alpha1.PackageRevision, msg string, err error) {
	log := log.FromContext(ctx)
	if err == nil {
		cr.SetConditions(condition.ConditionFailed(controllerCondition, msg))
		log.Error(msg)
		r.recorder.Eventf(cr, corev1.EventTypeWarning, controllerName, msg)
	} else {
		cr.SetConditions(condition.ConditionFailed(controllerCondition, err.Error()))
		log.Error(msg, "error", err)
		r.recorder.Eventf(cr, corev1.EventTypeWarning, controllerName, fmt.Sprintf("%s, err: %s", msg, err.Error()))
	}
}

func (r *reconciler) gatherPackageresources(ctx context.Context, cr *pkgv1alpha1.PackageRevision) (store.Storer[[]byte], bool, *unstructured.Unstructured, string, error) {
	resourceData := memory.NewStore[[]byte]()
	var inventoryResource string
	var inventoryExists bool

	pkgRevResources, err := r.clientset.PkgV1alpha1().PackageRevisionResourceses(cr.GetNamespace()).Get(ctx, cr.GetName(), metav1.GetOptions{})
	if err != nil {
		return resourceData, inventoryExists, nil, inventoryResource, err
	}

	for path, v := range pkgRevResources.Spec.Resources {
		// the path contains the files with their relative path within the package
		// Anything with a / is a file from a subdirectory and is not a business logic resource
		// for kform
		if !strings.Contains(path, "/") {
			// gather the kform resources which form the business logic for the kform fn run
			if err := resourceData.Create(ctx, store.ToKey(path), []byte(v)); err != nil {
				continue
			}
		}
		if config.InventoryPath("") == path {
			// this gathers the inventory file stored in .kform/kform-inventory.yaml to be able to
			// lookup the resources that were applied to the system
			inventoryResource = v
			inventoryExists = true
		}
	}

	if inventoryResource == "" {
		// no inventory found -> generate inventoryObject
		invConfig := config.New(genericiooptions.IOStreams{})
		invConfig.InventoryID, err = config.DefaultInventoryID()
		if err != nil {
			return resourceData, inventoryExists, nil, inventoryResource, err
		}
		inventoryResource = invConfig.GetInventoryFileString()
	}

	localInventory, err := config.ParseInventoryFile([]byte(inventoryResource))
	if err != nil {
		return resourceData, inventoryExists, nil, inventoryResource, err
	}
	return resourceData, inventoryExists, localInventory, inventoryResource, err
}

func gatherInput(ctx context.Context, cr *pkgv1alpha1.PackageRevision) store.Storer[[]byte] {
	// process input
	inputData := memory.NewStore[[]byte]()
	for i, input := range cr.Spec.Inputs {
		fmt.Println("kform processor gatherInput", fmt.Sprintf("pkgctx-%d.yaml", i), string(input.Raw))
		// ensure this is a yaml file, otherwise the reader will not pick it up
		inputData.Create(ctx, store.ToKey(fmt.Sprintf("pkgctx-%d.yaml", i)), input.Raw)
	}
	return inputData
}

func (r *reconciler) runkform(
	ctx context.Context,
	cr *pkgv1alpha1.PackageRevision,
	localInventory *unstructured.Unstructured,
	inputData store.Storer[[]byte],
	resourceData store.Storer[[]byte],
	destroy bool,
) (store.Storer[[]byte], error) {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := util.NewMatchVersionFlags(kubeConfigFlags)

	fmt.Println("kform processor inputdata", inputData.Len(ctx))

	outputData := memory.NewStore[[]byte]()
	kformRunnerConfig := &runner.Config{
		Factory:        util.NewFactory(util.NewMatchVersionFlags(matchVersionKubeConfigFlags)),
		PackageName:    cr.Spec.PackageRevID.Package,
		LocalInventory: localInventory,
		InputData:      inputData,
		ResourceData:   resourceData,
		OutputData:     outputData,
		DryRun:         false,
	}
	if destroy {
		kformRunnerConfig.Destroy = true
	} else {
		kformRunnerConfig.AutoApprove = true
	}

	kfrunner := runner.NewKformRunner(kformRunnerConfig)

	return outputData, kfrunner.Run(ctx)
}

func (r *reconciler) updatePackageRevisionResources(
	ctx context.Context,
	cr *pkgv1alpha1.PackageRevision,
	outputData store.Storer[[]byte],
	inventoryExists bool,
	localInventoryString string,
) error {
	resources := map[string]string{}
	outputData.List(ctx, func(ctx context.Context, key store.Key, data []byte) {
		resources[path.Join("out", key.Name)] = string(data)
	})

	// if the inventory resource was not provided we need to add it
	// to the resource so it will get committed
	if !inventoryExists {
		resources[config.InventoryPath("")] = localInventoryString
	}
	newPkgrevResources := pkgv1alpha1.BuildPackageRevisionResources(
		*cr.ObjectMeta.DeepCopy(),
		pkgv1alpha1.PackageRevisionResourcesSpec{
			PackageRevID: *cr.Spec.PackageRevID.DeepCopy(),
			Resources:    resources,
		},
		pkgv1alpha1.PackageRevisionResourcesStatus{},
	)
	return r.Update(ctx, newPkgrevResources)
}
