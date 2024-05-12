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

package packagescheduler

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/henderiw/apiserver-store/pkg/storebackend"
	memstore "github.com/henderiw/apiserver-store/pkg/storebackend/memory"
	"github.com/henderiw/logger/log"
	"github.com/henderiw/resource"
	"github.com/kform-dev/kform/pkg/dag"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/ctrlconfig"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/lease"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/packagediscovery/catalog"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/packagescheduler/pkg"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func init() {
	reconcilers.Register("packagescheduler", &reconciler{})
}

const (
	controllerName      = "PackageSchedulerController"
	controllerCondition = condition.ReadinessGate_PkgSchedule
	//controllerEventError = "PkgRevInstallerError"
	controllerEvent = "PackageScheduler"
	finalizer       = "packagescheduler.pkg.pkgserver.dev/finalizer"
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
	r.APIPatchingApplicator = resource.NewAPIPatchingApplicator(mgr.GetClient())
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)
	r.catalogStore = cfg.CatalogStore
	r.installedPkgStore = memstore.NewStore[*pkg.Package]()
	r.recorder = mgr.GetEventRecorderFor(controllerName)
	r.catalogStore = cfg.CatalogStore

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&pkgv1alpha1.PackageRevision{}).
		Complete(r)
}

type reconciler struct {
	resource.APIPatchingApplicator
	finalizer    *resource.APIFinalizer
	catalogStore *catalog.Store
	// key: target + Package(NS/Pkg) -> pkgRevName
	installedPkgStore storebackend.Storer[*pkg.Package]
	recorder          record.EventRecorder
	dag               dag.DAG[*configv1alpha1.PackageVariant]
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
	// if the pkgRev is a catalog packageRevision or
	// if the pkgrev does not have a pkg Dependency readiness gate
	// this event is not relevant
	if strings.HasPrefix(cr.GetName(), pkgid.PkgTarget_Catalog) ||
		!cr.HasReadinessGate(controllerCondition) {
		return ctrl.Result{}, nil
	}

	l := lease.New(r.Client, req.NamespacedName)
	if err := l.AcquireLease(ctx, controllerName); err != nil {
		log.Debug("cannot acquire lease", "targetKey", req.NamespacedName.String(), "error", err.Error())
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"lease", "error %s", err.Error())
		return ctrl.Result{Requeue: true, RequeueAfter: lease.RequeueInterval}, nil
	}
	r.recorder.Eventf(cr, corev1.EventTypeWarning,
		"lease", "acquired")

	if !cr.GetDeletionTimestamp().IsZero() {
		// delete the pkg from the installedPkgstore
		if err := r.installedPkgStore.UpdateWithFn(ctx, func(ctx context.Context, key storebackend.Key, pkg *pkg.Package) *pkg.Package {
			// delete the pkgRev from the installed pkg
			if key == getkeyFromPkgRev(cr) {
				pkg.DeletePackageRevision(&cr.Spec.PackageID)
			}
			// delete the pkgRev as owner reference from all packages
			pkg.DeleteOwnerRef(&cr.Spec.PackageID)
			return pkg
		}); err != nil {
			log.Error("cannot delete pkgrev as ownref", "pkgID", cr.Spec.PackageID.PkgRevString())
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
	// update pkgrev in installPkgStore
	if err := r.installedPkgStore.UpdateWithKeyFn(ctx, getkeyFromPkgRev(cr), func(ctx context.Context, p *pkg.Package) *pkg.Package {
		if p == nil {
			p = pkg.NewPackage()
		}
		log.Info("add pkgRev", "pkgID", cr.Spec.PackageID.String())
		p.AddPackageRevision(&cr.Spec.PackageID)
		return p
	}); err != nil {
		log.Error("cannot add pkgRev to installedPkgStore", "error", err.Error())
		r.recorder.Eventf(cr, corev1.EventTypeNormal,
			controllerEvent, "cannot add pkgRev %s to installedPkgStore %s", cr.Name)

		cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot add pkgRev to installedPkgStore", cr.Name))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	// reinitialize the dag
	r.dag = dag.New[*configv1alpha1.PackageVariant]()
	// add yourself to the dag
	if err := r.dag.AddVertex(ctx, cr.Spec.PackageID.PkgString(), nil); err != nil {
		log.Error("cannot add pkgRev to dag", "vertex", cr.Spec.PackageID.PkgString(), "error", err.Error())
		r.recorder.Eventf(cr, corev1.EventTypeNormal,
			controllerEvent, "cannot add pkgRev %s to dag %s", cr.Name)

		cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot add pkgRev to dag", cr.Name))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	recursivelyResolved, err := r.isPkgRevRecursivelyresolved(ctx, cr)
	if err != nil {
		log.Error("cannot recursively resolve dependencies", "error", err.Error())
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			controllerEvent, "cannot recursively resolve dependencies err: %s", err.Error())

		cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot recursively resolve dependencies", err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}
	if !recursivelyResolved {
		log.Info("pkg dependency resolution incomplete")
		r.recorder.Eventf(cr, corev1.EventTypeNormal,
			controllerEvent, "pkg dependency resolution incomplete")
		cr.SetConditions(condition.ConditionUpdate(controllerCondition, "pkg dependency resolution incomplete", ""))
		return ctrl.Result{RequeueAfter: 15 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	for vertex, pkgVar := range r.dag.GetVertices() {
		log.Debug("vertices", "vertex", vertex, "down dependencies", r.dag.GetDownVertexes(vertex), "up dependencies", r.dag.GetUpVertexes(vertex), "pvar present", pkgVar != nil)
		// if there are no downstream vertices it means the pkg can be installed
		// if the pkgVar is != nil the package variant can be installed
		// if the pkgVar is nil, the package is being installed -> dont do anything
		if len(r.dag.GetDownVertexes(vertex)) == 0 && pkgVar != nil {
			if err := r.Apply(ctx, pkgVar); err != nil {
				log.Error("cannot apply pkgVariant", "pkgVar", pkgVar.Name, "error", err.Error())
				r.recorder.Eventf(cr, corev1.EventTypeNormal,
					controllerEvent, "cannot apply pkgVariant %s", pkgVar.Name)

				cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot apply pkgVariant", pkgVar.Name))
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
			r.recorder.Eventf(cr, corev1.EventTypeNormal,
				controllerEvent, "pkgVar applied", pkgVar.Name)
		}
	}

	/*
		isResolved, upstreamPkgDeps := r.catalogStore.GetPkgRevDependencies(ctx, cr)
		// irespective if all packages are resolved we could already instantiate the packages
		log.Info("pkgRev dependencies", "isResolved", isResolved, "upstream", upstreamPkgDeps)
		for _, upstream := range upstreamPkgDeps {
			upstream := upstream
			// get the packageVariant
			pvar := buildPackageVariant(ctx, cr, &upstream)

			isInstalled, err := r.isPkgScheduled(ctx, &pvar.Spec.Downstream)
			if err != nil {
				log.Error("cannot get installed packages", "downstream", pvar.Spec.Downstream.String(), "error", err.Error())
				r.recorder.Eventf(cr, corev1.EventTypeNormal,
					controllerEvent, "cannot get install packages %s", pvar.Spec.Downstream.String())

				cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot get installed packages", pvar.Spec.Downstream.String()))
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
			if !isInstalled {
				// get latest revision
				latestCatalogPkgRev, err := r.getLatestCatalogPackageRevision(ctx, &upstream)
				if err != nil {
					log.Error("cannot get latest catalog package revision", "upstream", upstream.String(), "error", err.Error())
					r.recorder.Eventf(cr, corev1.EventTypeNormal,
						controllerEvent, "cannot get latest catalog package revision %s", upstream.String())

					cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot get latest catalog package revision", upstream.String()))
					return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}
				pvar.Spec.Upstream.Revision = latestCatalogPkgRev.Spec.PackageID.Revision

				if err := r.Apply(ctx, pvar); err != nil {
					log.Error("cannot apply package variant", "pvar", pvar.Name, "error", err.Error())
					r.recorder.Eventf(cr, corev1.EventTypeNormal,
						controllerEvent, "cannot apply package variant %s", pvar.Name)

					cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot apply package variant", pvar.Name))
					return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
				}
				r.recorder.Eventf(cr, corev1.EventTypeNormal,
					controllerEvent, "pvar applied", pvar.Name)
			}
			// update dependency in the install package store
			if err := r.installedPkgStore.UpdateWithKeyFn(ctx, getkeyFromDownStream(&pvar.Spec.Downstream), func(ctx context.Context, p *pkg.Package) *pkg.Package {
				if p == nil {
					p = pkg.NewPackage()
				}
				p.AddOwnerRef(&cr.Spec.PackageID)
				return p
			}); err != nil {
				log.Error("cannot add pkgRev to installedPkgStore", "error", err.Error())
				r.recorder.Eventf(cr, corev1.EventTypeNormal,
					controllerEvent, "cannot add pkgRev %s to installedPkgStore %s", cr.Name)

				cr.SetConditions(condition.ConditionUpdate(controllerCondition, "cannot add pkgRev to installedPkgStore", cr.Name))
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
		}
	*/
	if len(r.dag.GetVertices()) != 1 {
		log.Info("pkg dependencies being installed")
		r.recorder.Eventf(cr, corev1.EventTypeNormal,
			controllerEvent, "pkg dependencies being installed")
		cr.SetConditions(condition.ConditionUpdate(controllerCondition, "pkg dependencies being installed", ""))
		return ctrl.Result{RequeueAfter: 10 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	log.Debug("pkg dependencies installed")
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

func (r *reconciler) getPkgState(ctx context.Context, downstream *pkgid.Downstream) (*pkgv1alpha1.PackageRevision, pkgid.PkgState, error) {
	// TODO why does these client options not work
	opts := []client.ListOption{
		/*
			client.MatchingFields{
				"spec.packageID.target":     downstream.Target,
				"spec.packageID.repository": downstream.Repository,
				"spec.packageID.realm":      downstream.Realm,
				"spec.packageID.package":    downstream.Package,
			},
		*/
	}
	pkgRevs := pkgv1alpha1.PackageRevisionList{}
	if err := r.List(ctx, &pkgRevs, opts...); err != nil {
		return nil, pkgid.PkgState_NotAvailable, err
	}
	for _, pkgRev := range pkgRevs.Items {
		// check if the the package we are intersted in is scheduled
		if pkgRev.Spec.PackageID.Target == downstream.Target &&
			pkgRev.Spec.PackageID.Repository == downstream.Repository &&
			pkgRev.Spec.PackageID.Realm == downstream.Realm &&
			pkgRev.Spec.PackageID.Package == downstream.Package {
			if pkgRev.GetCondition(condition.ReadinessGate_PkgInstall).Status == metav1.ConditionTrue {
				return &pkgRev, pkgid.PkgState_Installed, nil
			}
			return &pkgRev, pkgid.PkgState_Scheduled, nil
		}
	}
	/*
		if len(installPkgs.Items) == 0 {
			return false, nil
		}
	*/
	return nil, pkgid.PkgState_NotAvailable, nil
}

func (r *reconciler) getLatestCatalogPackageRevision(ctx context.Context, upstream *pkgid.Upstream) (*pkgv1alpha1.PackageRevision, error) {
	log := log.FromContext(ctx)

	opts := []client.ListOption{
		/*
			client.MatchingFields{
				"spec.lifecycle":            "published",
				"spec.packageID.repository": upstream.Repository,
				"spec.packageID.realm":      upstream.Realm,
				"spec.packageID.package":    upstream.Package,
			},
		*/
	}
	log.Info("getLatestCatalogPackageRevision", "upstream", upstream.String())
	pkgRevList := pkgv1alpha1.PackageRevisionList{}
	if err := r.List(ctx, &pkgRevList, opts...); err != nil {
		return nil, err
	}
	pkgRevs := []pkgv1alpha1.PackageRevision{}
	for _, pkgRev := range pkgRevList.Items {
		if pkgRev.Spec.Lifecycle == pkgv1alpha1.PackageRevisionLifecyclePublished &&
			pkgRev.Spec.PackageID.Repository == upstream.Repository &&
			pkgRev.Spec.PackageID.Realm == upstream.Realm &&
			pkgRev.Spec.PackageID.Package == upstream.Package {
			pkgRevs = append(pkgRevs, pkgRev)
		}
	}

	latestRev := pkgid.NoRevision
	var latestPkgRev *pkgv1alpha1.PackageRevision
	for _, pkgRev := range pkgRevs {
		log.Debug("getLatestCatalogPackageRevision", "pkgID", pkgRev.Spec.PackageID, "latestRev", latestRev)
		if pkgid.IsRevLater(pkgRev.Spec.PackageID.Revision, latestRev) {
			latestRev = pkgRev.Spec.PackageID.Revision
			latestPkgRev = &pkgRev
		}
	}
	if latestPkgRev == nil {
		return nil, fmt.Errorf("cannot get latests ppkrev upstream: %v", upstream.String())
	}
	return latestPkgRev, nil
}

func (r *reconciler) isPkgRevRecursivelyresolved(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) (bool, error) {
	log := log.FromContext(ctx)
	isResolved, upstreamPkgDeps := r.catalogStore.GetPkgRevDependencies(ctx, pkgRev)
	if !isResolved {
		return isResolved, nil
	}
	for _, upstream := range upstreamPkgDeps {
		log.Info("upstream dependency", "pkgRev", pkgRev.Name, "upstream", upstream.String())
		upstream := upstream
		pkgVar := buildPackageVariant(ctx, pkgRev, &upstream)
		// TODO: could we use the memory store to find this info iso reaching out to the cache
		depPkgRev, pkgState, err := r.getPkgState(ctx, &pkgVar.Spec.Downstream)
		if err != nil {
			return false, fmt.Errorf("cannot get installed packages for downstream: %s, error: %s", pkgVar.Spec.Downstream.String(), err.Error())
		}
		switch pkgState {
		case pkgid.PkgState_NotAvailable:
			// continue recursively
			// get latest revision from the catalog
			latestCatalogPkgRev, err := r.getLatestCatalogPackageRevision(ctx, &upstream)
			if err != nil {
				return false, fmt.Errorf("cannot get latest catalog package revision for upstream: %s, error: %s", upstream.String(), err.Error())
			}
			// update the upstream with the selected revision
			pkgVar.Spec.Upstream.Revision = latestCatalogPkgRev.Spec.PackageID.Revision

			// build a new pkgrev that is dependent on the original pkgRev
			newDepPkgRev, err := buildPackageRevision(ctx, pkgVar)
			if err != nil {
				return false, fmt.Errorf("cannot build package revision, err: %s", err.Error())
			}

			// add new pkgrev to the dag with the pkgvar to be applied
			log.Debug("pkg not available add vertex", "vertex", newDepPkgRev.Spec.PackageID.PkgString())
			if err := r.addVertex(ctx, newDepPkgRev.Spec.PackageID.PkgString(), pkgVar); err != nil {
				return false, err
			}
			r.dag.Connect(ctx, pkgRev.Spec.PackageID.PkgString(), newDepPkgRev.Spec.PackageID.PkgString())

			isResolved, err = r.isPkgRevRecursivelyresolved(ctx, newDepPkgRev)
			if err != nil {
				return false, err
			}
			if !isResolved {
				return isResolved, nil
			}

		case pkgid.PkgState_Scheduled:
			// we have to wait -> add the pvar to the dag with a nil body
			// add already scheduled pkgRev to the dag without the pkgVar
			// as this indicated if the
			log.Debug("pkg scheduled", "vertex", depPkgRev.Spec.PackageID.PkgString())
			if err := r.addVertex(ctx, depPkgRev.Spec.PackageID.PkgString(), nil); err != nil {
				return false, err
			}
			r.dag.Connect(ctx, pkgRev.Spec.PackageID.PkgString(), depPkgRev.Spec.PackageID.PkgString())

		case pkgid.PkgState_Installed:
			// we can proceed, dont add the pkgRev to the dag
			// as such no down vertices will be visible
		default:
			return false, fmt.Errorf("unexpected pkgState: %s", pkgState.String())
		}
	}
	return isResolved, nil
}

func buildPackageVariant(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, upstream *pkgid.Upstream) *configv1alpha1.PackageVariant {
	log := log.FromContext(ctx)
	log.Debug("buildPackageVariant")
	return configv1alpha1.BuildPackageVariant(
		metav1.ObjectMeta{
			Namespace: pkgRev.Namespace,
			// DNS name can have - or .
			Name: fmt.Sprintf("%s-pkginstall-%s", pkgRev.Spec.PackageID.Target, strings.ReplaceAll(upstream.PkgString(), "/", ".")),
		},
		configv1alpha1.PackageVariantSpec{
			Upstream: *upstream,
			Downstream: pkgid.Downstream{
				Target:     pkgRev.Spec.PackageID.Target,
				Repository: pkgRev.Spec.PackageID.Repository,
				Realm:      upstream.Realm,
				Package:    upstream.Package,
			},
			PackageContext: configv1alpha1.PackageContext{
				// we could also copy the readiness gate from the original pkgRev
				// and exclude the schedule one
				ReadinessGates: []condition.ReadinessGate{
					// we dont have to set the scheduled condition as this is already done
					// by the scheduler
					{ConditionType: condition.ReadinessGate_PkgProcess},
					{ConditionType: condition.ReadinessGate_PkgApprove},
					{ConditionType: condition.ReadinessGate_PkgInstall},
				},
			},
		},
		configv1alpha1.PackageVariantStatus{},
	)
}

func buildPackageRevision(ctx context.Context, pkgVar *configv1alpha1.PackageVariant) (*pkgv1alpha1.PackageRevision, error) {
	log := log.FromContext(ctx)
	log.Debug("buildPackageRevision")
	hash, err := pkgVar.Spec.CalculateHash()
	if err != nil {
		return nil, err
	}

	pkgID := pkgid.PackageID{
		Target:     pkgVar.Spec.Downstream.Target,
		Repository: pkgVar.Spec.Downstream.Repository,
		Realm:      pkgVar.Spec.Downstream.Realm,
		Package:    pkgVar.Spec.Downstream.Package,
		Workspace:  pkgVar.GetWorkspaceName(hash),
		// the revision will be determined by the system
	}
	return pkgv1alpha1.BuildPackageRevision(
		metav1.ObjectMeta{
			Namespace:   pkgVar.Namespace,
			Name:        pkgID.PkgRevString(),
			Labels:      pkgVar.Spec.PackageContext.Labels,
			Annotations: pkgVar.Spec.PackageContext.Annotations,
		},
		pkgv1alpha1.PackageRevisionSpec{
			PackageID:      pkgID,
			Lifecycle:      pkgv1alpha1.PackageRevisionLifecycleDraft,
			Upstream:       pkgVar.Spec.Upstream.DeepCopy(),
			ReadinessGates: pkgVar.Spec.PackageContext.ReadinessGates,
			Inputs:         pkgVar.Spec.PackageContext.Inputs,
			Tasks: []pkgv1alpha1.Task{
				{
					Type: pkgv1alpha1.TaskTypeClone,
				},
			},
		},
		pkgv1alpha1.PackageRevisionStatus{},
	), nil
}

func (r *reconciler) addVertex(ctx context.Context, name string, pkgVar *configv1alpha1.PackageVariant) error {
	log := log.FromContext(ctx)
	if _, err := r.dag.GetVertex(name); err != nil {
		if err := r.dag.AddVertex(ctx, name, pkgVar); err != nil {
			log.Error("cannot add vertex to dag", "vertex", name, "error", err.Error())
			return err
		}
	} else {
		if err := r.dag.UpdateVertex(ctx, name, pkgVar); err != nil {
			log.Error("cannot update vertex to dag", "vertex", name, "error", err.Error())
			return err
		}
	}
	return nil
}
