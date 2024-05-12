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

package packagevariant

import (
	"context"
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/henderiw/logger/log"
	"github.com/henderiw/resource"
	"github.com/pkgserver-dev/pkgserver/apis/condition"
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	"github.com/pkgserver-dev/pkgserver/apis/pkgid"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers"
	"github.com/pkgserver-dev/pkgserver/pkg/reconcilers/ctrlconfig"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func init() {
	reconcilers.Register("packagevariant", &reconciler{})
}

const (
	controllerName = "PackageVariantController"
	finalizer      = "packagevariant.pkg.kform.dev/finalizer"
	// errors
	errGetCr        = "cannot get cr"
	errUpdateStatus = "cannot update status"
)

//+kubebuilder:rbac:groups=pkg.kform.dev,resources=packagevariant,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pkg.kform.dev,resources=packagevariant/status,verbs=get;update;patch

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
		For(&configv1alpha1.PackageVariant{}).
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

	cr := &configv1alpha1.PackageVariant{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			log.Error(errGetCr, "error", err)
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetCr)
		}
		return ctrl.Result{}, nil
	}

	cr = cr.DeepCopy()

	if !cr.GetDeletionTimestamp().IsZero() {
		// delete the package revisions
		done, err := r.deletePackageRevisions(ctx, cr)
		if err != nil {
			log.Error("cannot delete pkgRevs", "error", err)
			cr.SetConditions(condition.Failed(err.Error()))
			r.recorder.Eventf(cr, corev1.EventTypeWarning,
				"Error", "error %s", err.Error())
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}
		if !done {
			cr.SetConditions(condition.Failed("deleting"))
			r.recorder.Eventf(cr, corev1.EventTypeNormal,
				"packageVariant", "deleting")
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)	
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

	// validate the upstream package and downstream repo
	if err := r.validateUpstream(ctx, cr); err != nil {
		log.Error("failed validation upstream", "error", err)
		cr.SetConditions(condition.Failed(err.Error()))
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"Error", "error %s", err.Error())
		return ctrl.Result{RequeueAfter: 2 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}
	if err := r.validateDownstreamRepo(ctx, cr); err != nil {
		log.Error("failed validation downstream repo", "error", err)
		cr.SetConditions(condition.Failed(err.Error()))
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"Error", "error %s", err.Error())
		return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	if err := r.ensurePackageRevision(ctx, cr); err != nil {
		log.Error("failed ensure packageRevisions", "error", err)
		cr.SetConditions(condition.Failed(err.Error()))
		r.recorder.Eventf(cr, corev1.EventTypeWarning,
			"Error", "error %s", err.Error())
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	cr.SetConditions(condition.Ready())
	r.recorder.Eventf(cr, corev1.EventTypeNormal,
		"packageVariant", "ready")
	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
}

func (r *reconciler) validateUpstream(ctx context.Context, pvar *configv1alpha1.PackageVariant) error {
	opts := []client.ListOption{
		client.InNamespace(pvar.Namespace),
	}

	pkgRevList := pkgv1alpha1.PackageRevisionList{}
	if err := r.List(ctx, &pkgRevList, opts...); err != nil {
		return err
	}

	for _, pkgRev := range pkgRevList.Items {
		if pkgRev.Spec.PackageID.Repository == pvar.Spec.Upstream.Repository &&
			pkgRev.Spec.PackageID.Package == pvar.Spec.Upstream.Package &&
			pkgRev.Spec.PackageID.Revision == pvar.Spec.Upstream.Revision {
			if pkgRev.GetCondition(condition.ConditionTypeReady).Status == metav1.ConditionTrue {
				return nil
			}
			return fmt.Errorf("pkg %s not ready", pkgRev.Name)
		}
	}
	return fmt.Errorf("upstream pkg %s not found", pvar.Spec.Upstream.PkgRevName())
}

func (r *reconciler) validateDownstreamRepo(ctx context.Context, pkgVar *configv1alpha1.PackageVariant) error {
	repo := &configv1alpha1.Repository{}
	return r.Get(ctx, types.NamespacedName{
		Name:      pkgVar.Spec.Downstream.Repository,
		Namespace: pkgVar.Namespace,
	}, repo)
}

func (r *reconciler) deletePackageRevisions(ctx context.Context, pkgVar *configv1alpha1.PackageVariant) (bool, error) {
	log := log.FromContext(ctx)
	// get downstream packageRevisions, the hash does not matter
	pkgRevs, _, err := r.getDownstreamPRs(ctx, pkgVar, [sha1.Size]byte{})
	if err != nil {
		log.Error("cannot get downstream package revisions", "error", err)
		return false, err
	}
	// delete the remianing packageRevisions
	allDone := true
	for _, pkgRev := range pkgRevs {
		pkgRev := pkgRev
		log.Info("deleteOrOrphan delete from pkgVar", "pkgRev", pkgRev.Name)
		done, err := r.deleteOrOrphan(ctx, pkgRev, pkgVar)
		if  err != nil {
			log.Error("cannot delete or orphan the pkgRev", "pkgRev", pkgRev.Name)
			continue
		}
		if !done  {
			allDone = false
		}
	}
	return allDone, nil
}

func (r *reconciler) ensurePackageRevision(ctx context.Context, pkgVar *configv1alpha1.PackageVariant) error {
	log := log.FromContext(ctx)

	hash, err := pkgVar.Spec.CalculateHash()
	if err != nil {
		return err
	}

	// get downstream packageRevisions
	pkgRevs, exists, err := r.getDownstreamPRs(ctx, pkgVar, hash)
	if err != nil {
		log.Error("cannot get downstream package revisions", "error", err)
		return err
	}
	if !exists {
		pkgRev := buildPackageRevision(ctx, pkgVar, hash)
		if err := r.Create(ctx, pkgRev); err != nil {
			log.Error("created packageRevision from pkgVar failed", "pkgRev", pkgRev.Name, "pkgID", pkgRev.Spec.PackageID)
			return err
		}
		log.Info("created packageRevision", "pkgRev", pkgRev.Name)
		/*
			pkgRevResources := pkgv1alpha1.BuildPackageRevisionResources(
				pkgRev.ObjectMeta,
				pkgv1alpha1.PackageRevisionResourcesSpec{
					PackageID: *pkgRev.Spec.PackageID.DeepCopy(),
				},
				pkgv1alpha1.PackageRevisionResourcesStatus{},
			)
			if err := r.Create(ctx, pkgRev); err != nil {
				log.Error("created packageRevisionResources from pkgVar failed", "pkgRev", pkgRev.Name, "pkgID", pkgRev.Spec.PackageID)
				return err
			}
			log.Info("created packageRevisionresources", "pkgRev", pkgRevResources.Name)
		*/
	}
	// delete the packageRevisions that are no longer relevant
	for _, pkgRev := range pkgRevs {
		log.Info("deleteOrOrphan ensurePackageRevision", "pkgRev", pkgRev.Name)
		if pkgRev.Spec.PackageID.Workspace != pkgVar.GetWorkspaceName(hash) {
			if _, err := r.deleteOrOrphan(ctx, pkgRev, pkgVar); err != nil {
				log.Error("cannot delete or orphan the pkgRev", "pkgRev", pkgRev.Name)
			}
		}
	}
	return nil
}

func (r *reconciler) getDownstreamPRs(ctx context.Context, pkgVar *configv1alpha1.PackageVariant, hash [sha1.Size]byte) ([]*pkgv1alpha1.PackageRevision, bool, error) {
	log := log.FromContext(ctx)
	// track data to return
	existingPkgrevs := []*pkgv1alpha1.PackageRevision{}
	exists := false

	// get the current packageRevisions
	opts := []client.ListOption{
		client.InNamespace(pkgVar.Namespace),
	}
	pkgRevList := pkgv1alpha1.PackageRevisionList{}
	if err := r.List(ctx, &pkgRevList, opts...); err != nil {
		return nil, false, err
	}

	for _, pkgRev := range pkgRevList.Items {
		pkgRev := pkgRev
		owned := isOwned(ctx, &pkgRev, pkgVar)
		// delete packageRevisions that were owned but no longer match the
		// downstream package or repository

		if pkgRev.Spec.PackageID.Target == pkgVar.Spec.Downstream.Target &&
			pkgRev.Spec.PackageID.Repository == pkgVar.Spec.Downstream.Repository &&
			pkgRev.Spec.PackageID.Realm == pkgVar.Spec.Downstream.Realm &&
			pkgRev.Spec.PackageID.Package == pkgVar.Spec.Downstream.Package {

			if pkgRev.Spec.PackageID.Workspace == pkgVar.GetWorkspaceName(hash) {
				// if the pkgRev with the proper workspace name exists
				exists = true
			}
			log.Info("getDownstreamPRs added pkgRev", "pkgRev", pkgRev.Name)
			// add all pkgRevs including the one with the proper workspaceName
			existingPkgrevs = append(existingPkgrevs, &pkgRev)
		} else {
			// different target
			if owned {
				// We own this package, but it isn't a match for our downstream target,
				// which means that we created it but no longer need it.
				log.Info("deleteOrOrphan getDownstreamPRs", "pkgRev", pkgRev.Name)
				if _, err := r.deleteOrOrphan(ctx, &pkgRev, pkgVar); err != nil {
					log.Error("cannot delete or orphan pkgRev", "pkgRev", pkgRev.Name)
					continue
				}
			}
		}

		// from here we only see package that match the downstream package
		// there can be drafts or revisions
		/*
			if !owned && pkgVar.Spec.AdoptionPolicy == configv1alpha1.AdoptionPolicyAdoptExisting {
				log.Info("package variant adopts package revision", "packageVariant", pkgVar.Name, "packageRevision", pkgRev.Name)
				if err := r.adoptPackageRevision(ctx, &pkgRev, pkgVar); err != nil {
					log.Error("package variant cannot adopts package revision", "packageVariant", pkgVar.Name, "packageRevision", pkgRev.Name)
				}
			}
		*/

	}
	return existingPkgrevs, exists, nil
}

func (r *reconciler) deleteOrOrphan(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, pkgVar *configv1alpha1.PackageVariant) (bool, error) {
	log := log.FromContext(ctx)
	log.Info("deleteOrOrphan", "pkgRev", pkgRev.Name)
	switch pkgVar.Spec.DeletionPolicy {
	case configv1alpha1.DeletionPolicyDelete:
		return r.deletePackageRevision(ctx, pkgRev) // indicate true for successfull delete and false when deletion proposed or err
	case configv1alpha1.DeletionPolicyOrphan:
		return true, r.orphanPackageRevision(ctx, pkgRev, pkgVar)
	default:
		// this should never happen, because the pv should already be validated beforehand
		return false, fmt.Errorf("unexpected deletionPolicy %s for pkgVar %s", string(pkgVar.Spec.DeletionPolicy), pkgVar.Name)
	}
}

func (r *reconciler) deletePackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision) (bool, error) {
	switch pkgRev.Spec.Lifecycle {
	case pkgv1alpha1.PackageRevisionLifecycleDraft, pkgv1alpha1.PackageRevisionLifecycleProposed, pkgv1alpha1.PackageRevisionLifecycleDeletionProposed:
		if err := r.Client.Delete(ctx, pkgRev); err != nil {
			return false, err
		}
		return true, nil
	case pkgv1alpha1.PackageRevisionLifecyclePublished:
		pkgRev.Spec.Lifecycle = pkgv1alpha1.PackageRevisionLifecycleDeletionProposed
		if err := r.Client.Update(ctx, pkgRev); err != nil {
			return false, err
		}
		return false, nil
	default:
		// should never happen
		return false, fmt.Errorf("unexpected lifecycle %s for pkgVar %s", string(pkgRev.Spec.Lifecycle), pkgRev.Name)
	}
}

func (r *reconciler) orphanPackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, pkgVar *configv1alpha1.PackageVariant) error {
	pkgRev.ObjectMeta.OwnerReferences = removeOwnerRefByUID(ctx, pkgRev, pkgVar)
	if err := r.Client.Update(ctx, pkgRev); err != nil {
		return err
	}
	return nil
}

/*
func (r *reconciler) adoptPackageRevision(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, pkgVar *configv1alpha1.PackageVariant) error {
	pkgRev.ObjectMeta.OwnerReferences = append(pkgRev.OwnerReferences, getOwnerReference(ctx, pkgVar))
	if len(pkgVar.Spec.PackageContext.Labels) > 0 && pkgRev.ObjectMeta.Labels == nil {
		pkgRev.ObjectMeta.Labels = make(map[string]string)
	}
	for k, v := range pkgVar.Spec.PackageContext.Labels {
		pkgRev.ObjectMeta.Labels[k] = v
	}
	if len(pkgVar.Spec.PackageContext.Annotations) > 0 && pkgRev.ObjectMeta.Annotations == nil {
		pkgRev.ObjectMeta.Annotations = make(map[string]string)
	}
	for k, v := range pkgVar.Spec.PackageContext.Annotations {
		pkgRev.ObjectMeta.Annotations[k] = v
	}
	return r.Client.Update(ctx, pkgRev)
}
*/

func getOwnerReference(ctx context.Context, pkgVar *configv1alpha1.PackageVariant) metav1.OwnerReference {
	log := log.FromContext(ctx)
	log.Debug("getOwnerReference")
	return metav1.OwnerReference{
		APIVersion:         pkgVar.APIVersion,
		Kind:               pkgVar.Kind,
		Name:               pkgVar.Name,
		UID:                pkgVar.UID,
		Controller:         ptr.To[bool](true),
		BlockOwnerDeletion: nil,
	}
}

func removeOwnerRefByUID(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, pkgVar *configv1alpha1.PackageVariant) []metav1.OwnerReference {
	log := log.FromContext(ctx)
	log.Debug("removeOwnerRefByUID")
	var newOwnerReferences []metav1.OwnerReference
	for _, ownerRef := range pkgRev.GetOwnerReferences() {
		if ownerRef.APIVersion == pkgVar.APIVersion &&
			ownerRef.Kind == pkgVar.Kind &&
			ownerRef.Name == pkgVar.Name &&
			ownerRef.UID == pkgVar.UID {
			continue
		}
		newOwnerReferences = append(newOwnerReferences, ownerRef)
	}
	return newOwnerReferences
}

func isOwned(ctx context.Context, pkgRev *pkgv1alpha1.PackageRevision, pkgVar *configv1alpha1.PackageVariant) bool {
	log := log.FromContext(ctx)
	log.Debug("isOwned")
	for _, ownerRef := range pkgRev.GetOwnerReferences() {
		if ownerRef.APIVersion == pkgVar.APIVersion &&
			ownerRef.Kind == pkgVar.Kind &&
			ownerRef.Name == pkgVar.Name &&
			ownerRef.UID == pkgVar.UID {
			return true
		}
	}
	return false
}

func buildPackageRevision(ctx context.Context, pkgVar *configv1alpha1.PackageVariant, hash [sha1.Size]byte) *pkgv1alpha1.PackageRevision {
	// this is the first package revision
	pkgID := pkgid.PackageID{
		Target:     pkgVar.Spec.Downstream.Target,
		Repository: pkgVar.Spec.Downstream.Repository,
		Realm:      pkgVar.Spec.Downstream.Realm,
		Package:    pkgVar.Spec.Downstream.Package,
		Workspace:  pkgVar.GetWorkspaceName(hash),
	}
	return pkgv1alpha1.BuildPackageRevision(
		metav1.ObjectMeta{
			Namespace:       pkgVar.Namespace,
			Name:            pkgID.PkgRevString(),
			OwnerReferences: []metav1.OwnerReference{getOwnerReference(ctx, pkgVar)},
			Labels:          pkgVar.Spec.PackageContext.Labels,
			Annotations:     pkgVar.Spec.PackageContext.Annotations,
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
	)
}
