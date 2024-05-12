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
	"strings"

	"github.com/henderiw/logger/log"
	"github.com/henderiw/resource"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/ctrlconfig"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/lease"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func init() {
	reconcilers.Register("packageapproval", &reconciler{})
}

const (
	controllerName = "PackageApprovalController"
	//controllerEventError = "PkgRevInstallerError"
	controllerEvent = "PackageApproval"
	finalizer       = "packageapproval.pkg.kform.dev/finalizer"
	// errors
	errGetCr            = "cannot get cr"
	errUpdateStatus     = "cannot update status"
	controllerCondition = condition.ReadinessGate_PkgApprove
)

//+kubebuilder:rbac:groups=pkg.kform.dev,resources=packagerevision,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pkg.kform.dev,resources=packagerevision/status,verbs=get;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	/*
		if err := configv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
			return nil, err
		}
	*/
	r.APIPatchingApplicator = resource.NewAPIPatchingApplicator(mgr.GetClient())
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)
	r.recorder = mgr.GetEventRecorderFor(controllerName)

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&pkgv1alpha1.PackageRevision{}).
		Complete(r)
}

type reconciler struct {
	resource.APIPatchingApplicator
	finalizer *resource.APIFinalizer
	recorder  record.EventRecorder
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

	cr = cr.DeepCopy()
	key := types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      cr.GetName(),
	}
	// if the pkgRev is a catalog packageRevision or
	// if the pkgrev does not have a condition to process the pkgRev
	// this event is not relevant
	if strings.HasPrefix(cr.GetName(), pkgid.PkgTarget_Catalog) ||
		!cr.HasReadinessGate(controllerCondition) {
		return ctrl.Result{}, nil
	}
	// the previous condition is not finished, so we need to wait for acting
	if cr.GetPreviousCondition(controllerCondition).Status == metav1.ConditionFalse {
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
		// TODO release resources that were allocated e.g. ipam?

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

	switch cr.Spec.Lifecycle {
	case pkgv1alpha1.PackageRevisionLifecycleDraft:
		log.Debug("update lifecycle draft -> proposed")
		cr.Spec.Lifecycle = pkgv1alpha1.PackageRevisionLifecycleProposed
		cr.Spec.Tasks = []pkgv1alpha1.Task{}
		r.recorder.Eventf(cr, corev1.EventTypeNormal,
			controllerEvent, "update lifecycle draft -> proposed")
		cr.SetConditions(condition.ConditionUpdate(controllerCondition, "update lifecycle draft -> proposed", ""))
		if err := r.Update(ctx, cr); err != nil {
			log.Error("cannot update cr", "error", err)
			r.recorder.Eventf(cr, corev1.EventTypeWarning,
				controllerEvent, "error %s", err.Error())
			cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot update cr", err.Error()))
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}
		return ctrl.Result{Requeue: true}, nil
	case pkgv1alpha1.PackageRevisionLifecycleProposed:
		log.Debug("update lifecycle proposed -> published")
		cr.Spec.Lifecycle = pkgv1alpha1.PackageRevisionLifecyclePublished
		cr.Spec.Tasks = []pkgv1alpha1.Task{}
		r.recorder.Eventf(cr, corev1.EventTypeNormal,
			controllerEvent, "update lifecycle proposed -> published")
		cr.SetConditions(condition.ConditionUpdate(controllerCondition, "update lifecycle proposed -> published", ""))
		if err := r.Update(ctx, cr); err != nil {
			log.Error("cannot update cr", "error", err)
			r.recorder.Eventf(cr, corev1.EventTypeWarning,
				controllerEvent, "error %s", err.Error())
			cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot update cr", err.Error()))
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}
		return ctrl.Result{Requeue: true}, nil
	case pkgv1alpha1.PackageRevisionLifecyclePublished:
		log.Debug("appproval complete")
		if cr.GetCondition(controllerCondition).Status == metav1.ConditionFalse {
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
	default:
		// dont act upon deletion proposed
		return ctrl.Result{}, nil
	}
}
