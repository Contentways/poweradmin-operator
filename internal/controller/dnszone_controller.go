/*
Copyright 2026 Patrick Omland.

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

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	poweradmin "contentways.dev/contentways/poweradmin-go/poweradmin"
	dnsv1alpha1 "contentways.dev/contentways/poweradmin-operator/api/v1alpha1"
)

const (
	finalizerName = "dns.contentways.org/finalizer"

	conditionReady = "Ready"

	reasonSynced       = "Synced"
	reasonSyncFailed   = "SyncFailed"
	reasonDeleted      = "Deleted"
	reasonDeleteFailed = "DeleteFailed"
)

// DNSZoneReconciler reconciles a DNSZone object.
type DNSZoneReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	PoweradminClient *poweradmin.Client
}

// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnszones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnszones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnszones/finalizers,verbs=update

// Reconcile moves the current state of the DNSZone closer to the desired state.
func (r *DNSZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch the DNSZone CR.
	zone := &dnsv1alpha1.DNSZone{}
	if err := r.Get(ctx, req.NamespacedName, zone); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Handle deletion.
	if !zone.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, zone)
	}

	// 3. Ensure finalizer is set.
	if !controllerutil.ContainsFinalizer(zone, finalizerName) {
		controllerutil.AddFinalizer(zone, finalizerName)
		if err := r.Update(ctx, zone); err != nil {
			return ctrl.Result{}, fmt.Errorf("add finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. Reconcile desired state.
	if zone.Status.ZoneId == 0 {
		log.Info("Creating zone in Poweradmin", "zone", zone.Spec.Name)
		return r.reconcileCreate(ctx, zone)
	}

	log.Info("Updating zone in Poweradmin", "zone", zone.Spec.Name, "id", zone.Status.ZoneId)
	return r.reconcileUpdate(ctx, zone)
}

func (r *DNSZoneReconciler) reconcileCreate(ctx context.Context, zone *dnsv1alpha1.DNSZone) (ctrl.Result, error) {
	id, _, err := r.PoweradminClient.Zone.Create(ctx, poweradmin.ZoneCreateOpts{
		Name:    zone.Spec.Name,
		Type:    poweradmin.ZoneType(zone.Spec.Type),
		Masters: zone.Spec.Masters,
	})
	if err != nil {
		return ctrl.Result{}, r.setConditionFailed(ctx, zone, reasonSyncFailed, fmt.Sprintf("create zone: %s", err))
	}

	zone.Status.ZoneId = id
	r.setConditionReady(zone, reasonSynced, "Zone created successfully")

	if err := r.Status().Update(ctx, zone); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status after create: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *DNSZoneReconciler) reconcileUpdate(ctx context.Context, zone *dnsv1alpha1.DNSZone) (ctrl.Result, error) {
	zoneType := poweradmin.ZoneType(zone.Spec.Type)
	_, _, err := r.PoweradminClient.Zone.Update(ctx, zone.Status.ZoneId, poweradmin.ZoneUpdateOpts{
		Type:    &zoneType,
		Masters: &zone.Spec.Masters,
	})
	if err != nil {
		return ctrl.Result{}, r.setConditionFailed(ctx, zone, reasonSyncFailed, fmt.Sprintf("update zone: %s", err))
	}

	r.setConditionReady(zone, reasonSynced, "Zone synced successfully")

	if err := r.Status().Update(ctx, zone); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status after update: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *DNSZoneReconciler) reconcileDelete(ctx context.Context, zone *dnsv1alpha1.DNSZone) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if zone.Status.ZoneId != 0 {
		log.Info("Deleting zone from Poweradmin", "zone", zone.Spec.Name, "id", zone.Status.ZoneId)
		_, err := r.PoweradminClient.Zone.Delete(ctx, zone.Status.ZoneId)
		if err != nil && !poweradmin.IsNotFound(err) {
			return ctrl.Result{}, r.setConditionFailed(ctx, zone, reasonDeleteFailed, fmt.Sprintf("delete zone: %s", err))
		}
	}

	controllerutil.RemoveFinalizer(zone, finalizerName)
	if err := r.Update(ctx, zone); err != nil {
		return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
	}

	log.Info("Zone deleted", "zone", zone.Spec.Name)
	return ctrl.Result{}, nil
}

// setConditionReady sets the Ready condition to True on the zone (in-memory only).
func (r *DNSZoneReconciler) setConditionReady(zone *dnsv1alpha1.DNSZone, reason, message string) {
	meta.SetStatusCondition(&zone.Status.Conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: zone.Generation,
	})
}

// setConditionFailed sets the Ready condition to False, updates status, and returns the original error.
func (r *DNSZoneReconciler) setConditionFailed(ctx context.Context, zone *dnsv1alpha1.DNSZone, reason, message string) error {
	meta.SetStatusCondition(&zone.Status.Conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: zone.Generation,
	})
	_ = r.Status().Update(ctx, zone)
	return fmt.Errorf("%s", message)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DNSZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dnsv1alpha1.DNSZone{}).
		Named("dnszone").
		Complete(r)
}
