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

package packagerevision

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/henderiw/logger/log"
	"github.com/henderiw/resource"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"github.com/pkgserver-dev/pkgserver/pkg/cache"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/ctrlconfig"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/lease"
	"github.com/pkgserver-dev/pkgserver/pkg/repository"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func init() {
	reconcilers.Register("packagerevision", &reconciler{})
}

const (
	controllerName = "PackageRevisionController"
	finalizer      = "packagerevision.pkg.pkgserver.dev/finalizer"
	// errors
	errGetCr        = "cannot get cr"
	errUpdateStatus = "cannot update status"
)

//+kubebuilder:rbac:groups=pkg.pkgserver.dev,resources=packagerevision,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pkg.pkgserver.dev,resources=packagerevision/status,verbs=get;update;patch

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

	r.Client = mgr.GetClient()
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)
	r.repoCache = cfg.RepoCache
	r.recorder = mgr.GetEventRecorderFor(controllerName)

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&pkgv1alpha1.PackageRevision{}).
		Complete(r)
}

type reconciler struct {
	client.Client
	finalizer *resource.APIFinalizer
	repoCache *cache.Cache
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

	l := lease.New(r.Client, req.NamespacedName)
	if err := l.AcquireLease(ctx, controllerName); err != nil {
		log.Debug("cannot acquire lease", "targetKey", req.NamespacedName.String(), "error", err.Error())
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"lease", "error %s", err.Error())
		return ctrl.Result{Requeue: true, RequeueAfter: lease.RequeueInterval}, nil
	}
	r.recorder.Eventf(cr, corev1.EventTypeWarning,
		"lease", "acquired")
	// get the repo key
	repokey := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Spec.PackageID.Repository}

	if !cr.GetDeletionTimestamp().IsZero() {
		if cr.Spec.Lifecycle == pkgv1alpha1.PackageRevisionLifecyclePublished {
			msg := fmt.Sprintf("cannot delete a resource which lifecycle is %s, move to %s first", string(pkgv1alpha1.PackageRevisionLifecyclePublished), string(pkgv1alpha1.PackageRevisionLifecycleDeletionProposed))
			log.Debug(msg)
			cr.SetConditions(condition.Failed(msg))
			r.recorder.Eventf(cr, corev1.EventTypeWarning,
				"Error", msg)
			// someone need to update the status appropriatly
			return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}

		// Current Strategy: if repo does not exist we delete the pkgRev
		// The repo will not be updated, otherwise the resource hangs
		repo := &configv1alpha1.Repository{}
		if err := r.Get(ctx, repokey, repo); err != nil {
			log.Debug("cannot get repo -> delete anyway", "error", err)

			// remove the finalizer
			if err := r.finalizer.RemoveFinalizer(ctx, cr); err != nil {
				log.Error("cannot remove finalizer", "error", err)
				cr.SetConditions(condition.Failed(err.Error()))
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					"Error", "error %s", err.Error())
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}

			//log.Info("Successfully deleted resource")
			return ctrl.Result{}, nil
		}

		cachedRepo, err := r.repoCache.Open(ctx, repo)
		if err != nil {
			log.Error("cannot open cache", "error", err)
			cr.SetConditions(condition.Failed(err.Error()))
			r.recorder.Eventf(cr, corev1.EventTypeWarning,
				"Error", "error %s", err.Error())
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}

		// delete the package revision from git
		if err := cachedRepo.DeletePackageRevision(ctx, cr); err != nil {
			log.Error("cannot delete packagerevision", "error", err)
			// continue if error is not found
			if !strings.Contains(err.Error(), "reference not found") {
				cr.SetConditions(condition.Failed(err.Error()))
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					"Error", "error %s", err.Error())
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
		}

		// remove the finalizer
		if err := r.finalizer.RemoveFinalizer(ctx, cr); err != nil {
			log.Error("cannot remove finalizer", "error", err)
			cr.SetConditions(condition.Failed(err.Error()))
			r.recorder.Eventf(cr, corev1.EventTypeWarning,
				"Error", "error %s", err.Error())
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}

		log.Debug("Successfully deleted resource")
		return ctrl.Result{}, nil
	}

	if err := r.finalizer.AddFinalizer(ctx, cr); err != nil {
		log.Error("cannot add finalizer", "error", err)
		cr.SetConditions(condition.Failed(err.Error()))
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"Error", "error %s", err.Error())
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	if len(cr.Spec.Tasks) == 0 {
		if cr.Spec.Lifecycle == pkgv1alpha1.PackageRevisionLifecyclePublished {
			deployment, cachedRepo, err := r.getRepo(ctx, cr)
			if err != nil {
				//log.Error("cannot get repository", "error", err)
				cr.SetConditions(condition.Failed(err.Error()))
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					"Error", "error %s", err.Error())
				return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
			if cr.Spec.PackageID.Revision == "" {
				// allocate a revision

				repoPkgRevs, err := cachedRepo.ListPackageRevisions(ctx, &repository.ListOption{PackageID: &cr.Spec.PackageID})
				if err != nil {
					log.Error("cannot list repo pkgrevs", "error", err.Error())
					cr.SetConditions(condition.Failed(err.Error()))
					r.recorder.Eventf(cr, corev1.EventTypeWarning,
						"Error", "error %s", err.Error())
					return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}
				log.Info("published repo packages", "total", len(repoPkgRevs))

				opts := []client.ListOption{
					client.InNamespace(cr.Namespace),
					//client.MatchingFields{"spec.packageID.package": cr.Spec.PackageID.Package},
				}
				storedPkgRevs := &pkgv1alpha1.PackageRevisionList{}
				if err := r.List(ctx, storedPkgRevs, opts...); err != nil {
					log.Error("cannot list stored pkgrevs", "error", err.Error())
					cr.SetConditions(condition.Failed(err.Error()))
					r.recorder.Eventf(cr, corev1.EventTypeWarning,
						"Error", "error %s", err.Error())
					return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}
				log.Info("published stored packages", "total", len(storedPkgRevs.Items))

				allocatedRevisions := sets.New[string]()
				for _, pkgRev := range repoPkgRevs {
					pkgRev := pkgRev
					if pkgRev.Spec.PackageID.Repository == cr.Spec.PackageID.Repository &&
						pkgRev.Spec.PackageID.Target == cr.Spec.PackageID.Target &&
						pkgRev.Spec.PackageID.Realm == cr.Spec.PackageID.Realm &&
						pkgRev.Spec.PackageID.Package == cr.Spec.PackageID.Package {
						if pkgRev.Spec.PackageID.Revision != "" {
							allocatedRevisions.Insert(pkgRev.Spec.PackageID.Revision)
							log.Info("published repo packages", "name", pkgRev.Name, "revision", pkgRev.Spec.PackageID.Revision)
						}

					}
				}
				for _, pkgRev := range storedPkgRevs.Items {
					pkgRev := pkgRev
					if pkgRev.Spec.PackageID.Repository == cr.Spec.PackageID.Repository &&
						pkgRev.Spec.PackageID.Target == cr.Spec.PackageID.Target &&
						pkgRev.Spec.PackageID.Realm == cr.Spec.PackageID.Realm &&
						pkgRev.Spec.PackageID.Package == cr.Spec.PackageID.Package {
						if pkgRev.Spec.PackageID.Revision != "" {
							allocatedRevisions.Insert(pkgRev.Spec.PackageID.Revision)
							log.Info("published stored packages", "name", pkgRev.Name, "revision", pkgRev.Spec.PackageID.Revision)
						}
					}
				}
				nextRev, err := pkgid.NextRevisionNumber(allocatedRevisions.UnsortedList())
				if err != nil {
					cr.SetConditions(condition.Failed(err.Error()))
					r.recorder.Eventf(cr, corev1.EventTypeWarning,
						"Error", "error %s", err.Error())
					return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}
				log.Info("published package", "nextRev", nextRev)
				cr.Spec.PackageID.Revision = nextRev
				if err := r.Update(ctx, cr); err != nil {
					//log.Error("cannot update packagerevision", "error", err)
					cr.SetConditions(condition.Failed(err.Error()))
					r.recorder.Eventf(cr, corev1.EventTypeWarning,
						"Error", "error %s", err.Error())
					return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}
				// we requeue and ensure the pkg tag
				return ctrl.Result{Requeue: true}, nil
			}
			/* attempt 2
			// approval of the pkgRev
			log.Debug("approval pkgrev", "before", cr.Spec.PackageID.Revision)
			if err := cachedRepo.UpsertPackageRevision(ctx, cr, map[string]string{}); err != nil {
				//log.Error("cannot approve packagerevision", "error", err)
				cr.SetConditions(condition.Failed(err.Error()))
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					"Error", "error %s", err.Error())
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
			// TODO do an update because of the revision that got allocated
			log.Debug("approval pkgrev", "after", cr.Spec.PackageID.Revision)
			cr = cr.DeepCopy()
			if err := r.Update(ctx, cr); err != nil {
				//log.Error("cannot update packagerevision", "error", err)
				cr.SetConditions(condition.Failed(err.Error()))
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					"Error", "error %s", err.Error())
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
			*/
			/*
				pkgRevResources := pkgv1alpha1.BuildPackageRevisionResources(
					cr.ObjectMeta,
					pkgv1alpha1.PackageRevisionResourcesSpec{
						PackageID: *cr.Spec.PackageID.DeepCopy(),
					}
					pkgv1alpha1.PackageRevisionResourcesStatus{},
				)
				if err := r.Update(ctx, pkgRevResources); err != nil {
					//log.Error("cannot update packagerevision", "error", err)
					cr.SetConditions(condition.Failed(err.Error()))
					r.recorder.Eventf(cr, corev1.EventTypeWarning,
						"Error", "error %s", err.Error())
					return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}
			*/
			// ensure Tag for pkgRev in deployments
			if deployment {
				if err := cachedRepo.EnsurePackageRevision(ctx, cr); err != nil {
					cr.SetConditions(condition.Failed(err.Error()))
					r.recorder.Eventf(cr, corev1.EventTypeWarning,
						"Error", "error %s", err.Error())
					return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}
			}
		}
	} else {
		// only act if the ready condition was not true
		if cr.GetCondition(condition.ConditionTypeReady).Status != v1.ConditionTrue {
			switch cr.Spec.Tasks[0].Type {
			case pkgv1alpha1.TaskTypeClone:

				_, cachedRepo, err := r.getRepo(ctx, cr)
				if err != nil {
					//log.Error("cannot approve packagerevision", "error", err)
					cr.SetConditions(condition.Failed(err.Error()))
					r.recorder.Eventf(cr, corev1.EventTypeWarning,
						"Error", "error %s", err.Error())
					return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}

				resources, err := r.getResources(ctx, cr, cr.Spec.Upstream)
				if err != nil {
					log.Error("cannot get resources from src repo", "error", err)
					cr.SetConditions(condition.Failed(err.Error()))
					r.recorder.Eventf(cr, corev1.EventTypeWarning,
						"Error", "error %s", err.Error())
					return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}

				if err := cachedRepo.UpsertPackageRevision(ctx, cr, resources); err != nil {
					log.Error("cannot update packagerevision", "error", err)
					cr.SetConditions(condition.Failed(err.Error()))
					r.recorder.Eventf(cr, corev1.EventTypeWarning,
						"Error", "error %s", err.Error())
					return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}
				/*
					pkgRevResources := pkgv1alpha1.BuildPackageRevisionResources(
						cr.ObjectMeta,
						pkgv1alpha1.PackageRevisionResourcesSpec{
							PackageID: *cr.Spec.PackageID.DeepCopy(),
							Resources: resources,
						},
						pkgv1alpha1.PackageRevisionResourcesStatus{},
					)
					if err := r.Create(ctx, pkgRevResources); err != nil {
						//log.Error("cannot update packagerevision resources", "error", err)
						cr.SetConditions(condition.Failed(err.Error()))
						r.recorder.Eventf(cr, corev1.EventTypeWarning,
							"Error", "error %s", err.Error())
						return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
					}
				*/
			case pkgv1alpha1.TaskTypeInit:
				msg := fmt.Sprintf("unsupported taskType: %s", string(cr.Spec.Tasks[0].Type))
				cr.SetConditions(condition.Failed(msg))
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					"Error", msg)
				return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			default:
				msg := fmt.Sprintf("unsupported taskType: %s", string(cr.Spec.Tasks[0].Type))
				cr.SetConditions(condition.Failed(msg))
				r.recorder.Eventf(cr, corev1.EventTypeWarning,
					"Error", msg)
				return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
		}
	}
	cr.SetConditions(condition.Ready())
	r.recorder.Eventf(cr, corev1.EventTypeNormal,
		"packageRevision", "ready")
	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
}

func (r *reconciler) getResources(ctx context.Context, cr *pkgv1alpha1.PackageRevision, upstream *pkgid.Upstream) (map[string]string, error) {
	log := log.FromContext(ctx)

	/*
		opts := []client.ListOption{
			client.InNamespace(cr.Namespace),
			client.MatchingFields{
				"spec.packageID.repository": upstream.Repository,
				"spec.packageID.realm":      upstream.Realm,
				"spec.packageID.package":    upstream.Package,
				"spec.packageID.revision":   upstream.Revision,
			},
		}
		pkgRevList := &pkgv1alpha1.PackageRevisionResourcesList{}
		if err := r.List(ctx, pkgRevList, opts...); err != nil {
			log.Error("cannot get pkgRev, list failed", "error", err.Error())
			return nil, err
		}
		if len(pkgRevList.Items) != 1 {
			log.Error("cannot find upstream packageresources", "error", pkgRevList.Items)
			return nil, fmt.Errorf("cannot find upstream packageresources: %s", upstream.String())
		}
		pkgRevResources := &pkgRevList.Items[0]
		for key := range pkgRevResources.Spec.Resources {
			log.Info("getResources", "key", key)
		}
		return pkgRevResources.Spec.Resources, nil
	*/

	opts := []client.ListOption{
		client.InNamespace(cr.Namespace),
	}
	pkgRevList := &pkgv1alpha1.PackageRevisionList{}
	if err := r.List(ctx, pkgRevList, opts...); err != nil {
		log.Error("cannot get pkgRev, list failed", "error", err.Error())
		return nil, err
	}
	var upstreamPkgRev *pkgv1alpha1.PackageRevision
	for _, pkgRev := range pkgRevList.Items {
		pkgRev := pkgRev
		if pkgRev.Spec.PackageID.Target == pkgid.PkgTarget_Catalog &&
			pkgRev.Spec.PackageID.Repository == upstream.Repository &&
			pkgRev.Spec.PackageID.Realm == upstream.Realm &&
			pkgRev.Spec.PackageID.Package == upstream.Package &&
			pkgRev.Spec.PackageID.Revision == upstream.Revision {
			// found
			upstreamPkgRev = &pkgRev
			break
		}
	}
	if upstreamPkgRev == nil {
		return nil, fmt.Errorf("cannot find upstream packageresources: %s", upstream.String())
	}
	key := types.NamespacedName{
		Namespace: upstreamPkgRev.Namespace,
		Name:      upstreamPkgRev.Name,
	}
	log.Info("getResources", "pkgRevResoureKey", key.String())
	pkgRevResources := &pkgv1alpha1.PackageRevisionResources{}
	if err := r.Get(ctx, key, pkgRevResources); err != nil {
		log.Error("cannot get pkgRevResources, get failed", "error", err.Error())
		return nil, err
	}
	for key := range pkgRevResources.Spec.Resources {
		log.Info("getResources", "key", key)
	}
	return pkgRevResources.Spec.Resources, nil

	// attempt 2
	/*
		srcrepoKey := types.NamespacedName{
			Name:      upstream.Repository,
			Namespace: cr.Namespace,
		}
		srcRepo := &configv1alpha1.Repository{}
		if err := r.Get(ctx, srcrepoKey, srcRepo); err != nil {
			log.Error("cannot get src repo", "error", err)
			return nil, err
		}

		srcCachedRepo, err := r.repoCache.Open(ctx, srcRepo)
		if err != nil {
			log.Error("cannot open src repo cache", "error", err)
			return nil, err
		}

		return srcCachedRepo.GetResources(ctx, &pkgv1alpha1.PackageRevision{
			Spec: pkgv1alpha1.PackageRevisionSpec{
				PackageID: pkgid.PackageID{
					Target:     pkgid.PkgTarget_Catalog,
					Repository: upstream.Repository,
					Realm:      upstream.Realm,
					Package:    upstream.Package,
					Revision:   upstream.Revision,
					//Workspace:  upstream.Revision,
				},
				// lifecycle must be set to ensure we only take resources from the
				// published repo
				Lifecycle: pkgv1alpha1.PackageRevisionLifecyclePublished,
			},
		})
	*/
}

func (r *reconciler) getRepo(ctx context.Context, cr *pkgv1alpha1.PackageRevision) (bool, *cache.CachedRepository, error) {
	log := log.FromContext(ctx)
	repokey := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Spec.PackageID.Repository}

	repo := &configv1alpha1.Repository{}
	if err := r.Get(ctx, repokey, repo); err != nil {
		log.Error("cannot get repo", "error", err)
		return false, nil, err
	}
	cachedRepo, err := r.repoCache.Open(ctx, repo)
	return repo.Spec.Deployment, cachedRepo, err
}
