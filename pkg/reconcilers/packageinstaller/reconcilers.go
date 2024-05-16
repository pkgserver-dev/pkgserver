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

package packageinstaller

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/henderiw/logger/log"
	"github.com/henderiw/resource"
	"github.com/pkg/errors"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgrevid"
	"github.com/pkgserver-dev/pkgserver/pkg/client"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/ctrlconfig"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/lease"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	configsyncv1beta1 "kpt.dev/configsync/pkg/api/configsync/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func init() {
	reconcilers.Register("packageinstaller", &reconciler{})
}

const (
	controllerName = "PackageInstallerController"
	//controllerEventError = "PkgRevInstallerError"
	controllerEvent = "PackageInstaller"
	finalizer       = "packageinstaller.pkg.pkgserver.dev/finalizer"
	// errors
	errGetCr            = "cannot get cr"
	errUpdateStatus     = "cannot update status"
	controllerCondition = condition.ReadinessGate_PkgInstall
	gitopsNamespace     = "config-management-system"
)

//+kubebuilder:rbac:groups=pkg.pkgserver.dev,resources=packagerevision,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pkg.pkgserver.dev,resources=packagerevision/status,verbs=get;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	/*
		if err := configv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
			return nil, err
		}
	*/
	r.Client = mgr.GetClient()
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)
	r.recorder = mgr.GetEventRecorderFor(controllerName)

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&pkgv1alpha1.PackageRevision{}).
		Complete(r)
}

type reconciler struct {
	client.Client
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
	key := types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      cr.GetName(),
	}
	cr = cr.DeepCopy()
	log = log.With("pkgID", cr.Spec.PackageRevID)
	// if the pkgRev is a catalog packageRevision or
	// if the pkgrev does not have a condition to process the pkgRev
	// this event is not relevant
	if strings.HasPrefix(cr.GetName(), pkgrevid.PkgTarget_Catalog) ||
		!cr.HasReadinessGate(controllerCondition) {
		return ctrl.Result{}, nil
	}
	// the package revisioner first need to act e.g. to clone the repo
	if cr.GetCondition(condition.ConditionTypeReady).Status == metav1.ConditionFalse {
		return ctrl.Result{}, nil
	}
	// the previous condition is not finished, so we need to wait for acting
	if cr.GetPreviousCondition(controllerCondition).Status == metav1.ConditionFalse {
		return ctrl.Result{}, nil
	}
	if cr.Spec.PackageRevID.Revision == "" {
		log.Info("package not released")
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

	// the installer uses 2 conditions

	if cr.GetCondition(controllerCondition).Status == metav1.ConditionFalse {
		rootSyncKey := types.NamespacedName{
			Namespace: gitopsNamespace,
			Name:      cr.Spec.PackageRevID.DNSName(),
		}
		rootSync := &configsyncv1beta1.RootSync{}
		if err := r.Get(ctx, rootSyncKey, rootSync); err != nil {
			if resource.IgnoreNotFound(err) != nil {
				log.Error("cannot get rootSync", "error", err)
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					controllerEvent, "error %s", err.Error())
				cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot add finalizer", err.Error()))
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
			rootSync, err := r.buildRootSync(ctx, cr)
			if err != nil {
				log.Error("cannot build rootSync", "error", err)
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					controllerEvent, "error %s", err.Error())
				cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot build rootSync", err.Error()))
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
			if err := r.Create(ctx, rootSync); err != nil {
				log.Error("cannot create rootSync", "error", err)
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					controllerEvent, "error %s", err.Error())
				cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot create rootSync", err.Error()))
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}

		}
		// did the spec change
		if rootSync.Spec.Git == nil ||
			rootSync.Spec.Git.Revision != cr.Spec.PackageRevID.GitRevision() ||
			rootSync.Spec.Git.Branch != cr.Spec.PackageRevID.Branch(false) {
			rootSync, err := r.buildRootSync(ctx, cr)
			if err != nil {
				log.Error("cannot build rootSync", "error", err)
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					controllerEvent, "error %s", err.Error())
				cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot build rootSync", err.Error()))
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
			if err := r.Update(ctx, rootSync); err != nil {
				log.Error("cannot apply rootSync", "error", err)
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					controllerEvent, "error %s", err.Error())
				cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot update rootSync", err.Error()))
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
		}
		// determine if rootsync is in good shape
		// TBD determine what good is here
		if len(rootSync.Status.Source.Errors) != 0 &&
			len(rootSync.Status.Sync.Errors) != 0 {

			// sync errors
			log.Info("package install sync errors")
			r.recorder.Eventf(cr, corev1.EventTypeNormal,
				controllerEvent, "package install sync errors")
			cr.SetConditions(condition.ConditionUpdate(controllerCondition, "package install sync errors", fmt.Errorf("sync errors: %d, source errors %d", len(rootSync.Status.Sync.Errors), len(rootSync.Status.Source.Errors)).Error()))
			cType := cr.NextReadinessGate(controllerCondition)
			if cType != condition.ConditionTypeEnd {
				if !cr.HasCondition(cType) {
					cr.SetConditions(condition.ConditionUpdate(cType, "init", ""))
				}
			}
			return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)

		}
		// all good synced
		log.Info("package install complete")
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
	// nothing to do all good
	return ctrl.Result{}, nil
}

func (r *reconciler) buildRootSync(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) (*configsyncv1beta1.RootSync, error) {
	repo, err := r.getRepo(ctx, pkgRev)
	if err != nil {
		return nil, err
	}

	switch repo.Spec.Type {
	case configv1alpha1.RepositoryTypeGit:
		return &configsyncv1beta1.RootSync{
			TypeMeta: metav1.TypeMeta{
				Kind:       reflect.TypeOf(configsyncv1beta1.RootSync{}).Name(),
				APIVersion: configsyncv1beta1.SchemeGroupVersion.Identifier(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: gitopsNamespace,
				Name:      pkgRev.Spec.PackageRevID.DNSName(),
			},
			Spec: configsyncv1beta1.RootSyncSpec{
				SourceFormat: "unstructured",
				SourceType:   string(configv1alpha1.RepositoryTypeGit),
				Git: &configsyncv1beta1.Git{
					Repo:     repo.Spec.Git.URL,
					Branch:   pkgRev.Spec.PackageRevID.Branch(false),
					Revision: pkgRev.Spec.PackageRevID.GitRevision(),
					Dir:      pkgRev.Spec.PackageRevID.OutDir(),
					Auth:     "token",
					SecretRef: &configsyncv1beta1.SecretReference{
						Name: repo.Spec.Git.Credentials,
					},
				},
			},
		}, nil
	case configv1alpha1.RepositoryTypeOCI:
		// TODO code build rootsync
		return nil, fmt.Errorf("unsupported repo type, got: %s", string(repo.Spec.Type))
	default:
		return nil, fmt.Errorf("unsupported repo type, got: %s", string(repo.Spec.Type))
	}
}

func (r *reconciler) getRepo(ctx context.Context, cr *pkgv1alpha1.PackageRevision) (*configv1alpha1.Repository, error) {
	log := log.FromContext(ctx)
	repokey := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Spec.PackageRevID.Repository}

	repo := &configv1alpha1.Repository{}
	if err := r.Get(ctx, repokey, repo); err != nil {
		log.Error("cannot get repo", "error", err)
		return nil, err
	}
	return repo, nil
}
