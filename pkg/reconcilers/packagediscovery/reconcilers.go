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
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/henderiw/logger/log"
	"github.com/henderiw/resource"
	"github.com/kform-dev/kform/pkg/recorder"
	"github.com/kform-dev/kform/pkg/recorder/diag"
	"github.com/pkgserver-dev/pkgserver/apis/generated/clientset/versioned"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/ctrlconfig"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/lease"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/packagediscovery/catalog"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func init() {
	reconcilers.Register("packagediscovery", &reconciler{})
}

const (
	controllerName = "PackageDiscoveryController"
	finalizer      = "packagediscovery.pkg.kform.dev/finalizer"
	// errors
	errGetCr        = "cannot get cr"
	errUpdateStatus = "cannot update status"
)

//+kubebuilder:rbac:groups=pkg.kform.dev,resources=packagerevision,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pkg.kform.dev,resources=packagerevision/status,verbs=get;update;patch

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
	r.clientset = cfg.ClientSet
	//r.clientset = cfg.ClientSet
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)
	r.catalogStore = cfg.CatalogStore
	//r.installedPkgStore = memstore.NewStore[*api.PackageRevision]()
	r.recorder = mgr.GetEventRecorderFor(controllerName)

	d := NewAPIDiscoverer(mgr.GetClient(), mgr.GetConfig(), r.catalogStore)
	go d.Start(ctx)

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&pkgv1alpha1.PackageRevision{}).
		Complete(r)
}

type reconciler struct {
	client.Client
	clientset *versioned.Clientset
	//clientset    *versioned.Clientset
	finalizer    *resource.APIFinalizer
	catalogStore *catalog.Store
	// key: cluster + Package(NS/Pkg) -> pkgRevName
	//installedPkgStore storebackend.Storer[*api.PackageRevision] // we take the latest revision
	recorder record.EventRecorder
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = ctrlconfig.InitContext(ctx, controllerName, req.NamespacedName)
	log := log.FromContext(ctx)
	log.Debug("reconcile")

	cr := &pkgv1alpha1.PackageRevision{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			log.Error(errGetCr, "error", err)
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetCr)
		}
		return ctrl.Result{}, nil
	}

	// only look at catalog packages
	if !strings.HasPrefix(cr.GetName(), pkgid.PkgTarget_Catalog) {
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

	if !cr.GetDeletionTimestamp().IsZero() {
		// update the catalog stores takes care of deleting the pkgRev from the relevant stores
		// G/K -> API (versions, etc)
		// G/R -> K
		// PKG/REV -> DEP
		if err := r.updateCatalogStores(ctx, cr); err != nil {
			log.Error("cannot update catalog store")
			r.recorder.Eventf(cr, corev1.EventTypeWarning,
				"PkgRevDiscoveryError", "catalog stores: cannot delete pkgRev: %s", err.Error())
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}

		// remove the finalizer
		if err := r.finalizer.RemoveFinalizer(ctx, cr); err != nil {
			//log.Error("cannot remove finalizer", "error", err)
			r.recorder.Eventf(cr, corev1.EventTypeWarning,
				"PkgRevDiscoveryError", "error %s", err.Error())
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	if err := r.finalizer.AddFinalizer(ctx, cr); err != nil {
		log.Error("cannot add finalizer", "error", err)
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"PkgRevDiscoveryError", "error %s", err.Error())
		return ctrl.Result{Requeue: true}, nil
	}

	// update the catalog stores takes updates the catalog store when a pkg is in published state
	// otherwise the pkgrev is deleted from the catalog stores
	// G/K -> API (versions, etc)
	// G/R -> K
	// PKG/REV -> DEP
	if err := r.updateCatalogStores(ctx, cr); err != nil {
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"PkgRevDiscoveryError", "catalog stores: cannot update pkgRev: %s", err.Error())
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	r.recorder.Eventf(cr, corev1.EventTypeNormal,
		"PkgRevDiscovery", "ready")
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// updateCatalogStores updates the catalog store when a pkg is in published state
// otherwise the pkgrev is deleted from the catalog stores
// G/K -> API (versions, etc)
// G/R -> K
// PKG/REV -> DEP
func (r *reconciler) updateCatalogStores(ctx context.Context, cr *pkgv1alpha1.PackageRevision) error {
	// a recorder is needed for each pkgRev processing, hence we attach it to the context
	pkgRevRecorder := recorder.New[diag.Diagnostic]()
	r.catalogStore.InitializeRecorder(pkgRevRecorder)
	log := log.FromContext(ctx)
	if cr.GetDeletionTimestamp() != nil {
		if cr.Spec.Lifecycle != pkgv1alpha1.PackageRevisionLifecyclePublished {
			// delete all the pkgRev from the apiStore
			// and potentially delete the api entry when no pkgRevs are available
			r.catalogStore.DeletePkgRev(ctx, cr)
			if pkgRevRecorder.Get().HasError() {
				log.Error("cannot update catalog stores", "error", pkgRevRecorder.Get().Error())
				return pkgRevRecorder.Get().Error()
			}
			return nil
		}
		// we cannot delete a pkgRev that is published
		return fmt.Errorf("cannot delete a pkgRev that is published, update lifecycle to deletion proposed")
	}
	// get the various resources
	// outputs is relevant for the API catalog store -> retains the apis exposed by the packages
	// packages and resources determine the dependencies this pkgRev has against other packages
	packages, resources, inputs, outputs, err := r.getPackageResources(ctx, cr)
	if err != nil {
		log.Error("cannot get package resources", "error", err.Error())
		return err
	}
	log.Debug("pkg resources", "packages", len(packages), "resources", len(resources), "inputs", len(inputs), "outputs", len(outputs))
	r.catalogStore.UpdatePkgRevAPI(ctx, cr, outputs)
	r.catalogStore.UpdatePkgRevDependencies(ctx, cr, packages, inputs, resources)
	if pkgRevRecorder.Get().HasError() {
		log.Error("cannot update catalog stores", "error", pkgRevRecorder.Get().Error())
		return pkgRevRecorder.Get().Error()
	}
	return nil

}
