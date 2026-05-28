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

const recordFinalizerName = "dns.contentways.org/record-finalizer"

// DNSRecordReconciler reconciles a DNSRecord object.
type DNSRecordReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	PoweradminClient *poweradmin.Client
}

// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnsrecords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnsrecords/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnsrecords/finalizers,verbs=update

// Reconcile moves the current state of the DNSRecord closer to the desired state.
func (r *DNSRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch the DNSRecord CR.
	record := &dnsv1alpha1.DNSRecord{}
	if err := r.Get(ctx, req.NamespacedName, record); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Handle deletion.
	if !record.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, record)
	}

	// 3. Ensure finalizer is set.
	if !controllerutil.ContainsFinalizer(record, recordFinalizerName) {
		controllerutil.AddFinalizer(record, recordFinalizerName)
		if err := r.Update(ctx, record); err != nil {
			return ctrl.Result{}, fmt.Errorf("add finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. Resolve zone ID from zone name.
	zoneID, err := r.resolveZoneID(ctx, record.Spec.ZoneName)
	if err != nil {
		return ctrl.Result{}, r.setConditionFailed(ctx, record, "ZoneNotFound", fmt.Sprintf("resolve zone %s: %s", record.Spec.ZoneName, err))
	}

	// 5. Reconcile desired state.
	if record.Status.RecordId == 0 {
		log.Info("Creating record in Poweradmin", "zone", record.Spec.ZoneName, "name", record.Spec.Name, "type", record.Spec.Type)
		return r.reconcileCreate(ctx, record, zoneID)
	}

	log.Info("Updating record in Poweradmin", "zone", record.Spec.ZoneName, "name", record.Spec.Name, "id", record.Status.RecordId)
	return r.reconcileUpdate(ctx, record)
}

func (r *DNSRecordReconciler) reconcileCreate(ctx context.Context, record *dnsv1alpha1.DNSRecord, zoneID int) (ctrl.Result, error) {
	id, _, err := r.PoweradminClient.Record.Create(ctx, int(zoneID), poweradmin.RecordCreateOpts{
		Name:     record.Spec.Name,
		Type:     record.Spec.Type,
		Content:  record.Spec.Content,
		TTL:      record.Spec.TTL,
		Priority: record.Spec.Priority,
		Disabled: record.Spec.Disabled,
	})
	if err != nil {
		return ctrl.Result{}, r.setConditionFailed(ctx, record, "SyncFailed", fmt.Sprintf("create record: %s", err))
	}

	record.Status.RecordId = int(id)
	record.Status.ZoneId = zoneID
	r.setConditionReady(record, "Synced", "Record created successfully")

	if err := r.Status().Update(ctx, record); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status after create: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *DNSRecordReconciler) reconcileUpdate(ctx context.Context, record *dnsv1alpha1.DNSRecord) (ctrl.Result, error) {
	_, _, err := r.PoweradminClient.Record.Update(ctx, record.Status.ZoneId, int64(record.Status.RecordId), poweradmin.RecordUpdateOpts{
		Name:    record.Spec.Name,
		Type:    record.Spec.Type,
		Content: record.Spec.Content,
		TTL:     &record.Spec.TTL,
	})
	if err != nil {
		return ctrl.Result{}, r.setConditionFailed(ctx, record, "SyncFailed", fmt.Sprintf("update record: %s", err))
	}

	r.setConditionReady(record, "Synced", "Record synced successfully")

	if err := r.Status().Update(ctx, record); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status after update: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *DNSRecordReconciler) reconcileDelete(ctx context.Context, record *dnsv1alpha1.DNSRecord) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if record.Status.RecordId != 0 {
		log.Info("Deleting record from Poweradmin", "name", record.Spec.Name, "id", record.Status.RecordId)
		_, err := r.PoweradminClient.Record.Delete(ctx, int(record.Status.ZoneId), int64(record.Status.RecordId))
		if err != nil && !poweradmin.IsNotFound(err) {
			return ctrl.Result{}, r.setConditionFailed(ctx, record, "DeleteFailed", fmt.Sprintf("delete record: %s", err))
		}
	}

	controllerutil.RemoveFinalizer(record, recordFinalizerName)
	if err := r.Update(ctx, record); err != nil {
		return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
	}

	log.Info("Record deleted", "name", record.Spec.Name)
	return ctrl.Result{}, nil
}

// resolveZoneID looks up the zone ID by name from Poweradmin.
func (r *DNSRecordReconciler) resolveZoneID(ctx context.Context, zoneName string) (int, error) {
	zone, _, err := r.PoweradminClient.Zone.GetByName(ctx, zoneName)
	if err != nil {
		return 0, err
	}
	return zone.ID, nil
}

// setConditionReady sets the Ready condition to True (in-memory only).
func (r *DNSRecordReconciler) setConditionReady(record *dnsv1alpha1.DNSRecord, reason, message string) {
	meta.SetStatusCondition(&record.Status.Conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: record.Generation,
	})
}

// setConditionFailed sets the Ready condition to False, updates status, and returns the error.
func (r *DNSRecordReconciler) setConditionFailed(ctx context.Context, record *dnsv1alpha1.DNSRecord, reason, message string) error {
	meta.SetStatusCondition(&record.Status.Conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: record.Generation,
	})
	_ = r.Status().Update(ctx, record)
	return fmt.Errorf("%s", message)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DNSRecordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dnsv1alpha1.DNSRecord{}).
		Named("dnsrecord").
		Complete(r)
}
