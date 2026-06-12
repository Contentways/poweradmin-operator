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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	poweradmin "contentways.dev/contentways/poweradmin-go/v2/poweradmin"
	dnsv1alpha1 "contentways.dev/contentways/poweradmin-operator/api/v1alpha1"
)

const recordFinalizerName = "dns.contentways.org/record-finalizer"

// DNSRecordReconciler reconciles a DNSRecord object.
type DNSRecordReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	CredentialsSecretName string
	// NewClient can be overridden in tests to inject a mock client.
	// If nil, credentials are read from the namespace Secret.
	NewClient func(url, apiKey string) (*poweradmin.Client, error)
}

// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnsrecords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnsrecords/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnsrecords/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile moves the current state of the DNSRecord closer to the desired state.
func (r *DNSRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	record := &dnsv1alpha1.DNSRecord{}
	if err := r.Get(ctx, req.NamespacedName, record); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	paClient, err := r.buildClient(ctx, req.Namespace)
	if err != nil {
		return ctrl.Result{}, r.setConditionFailed(ctx, record, "CredentialsNotFound", err.Error())
	}

	if !record.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, record, paClient)
	}

	if !controllerutil.ContainsFinalizer(record, recordFinalizerName) {
		controllerutil.AddFinalizer(record, recordFinalizerName)
		if err := r.Update(ctx, record); err != nil {
			return ctrl.Result{}, fmt.Errorf("add finalizer: %w", err)
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	zoneID, err := r.resolveZoneID(ctx, record.Spec.ZoneName, paClient)
	if err != nil {
		return ctrl.Result{}, r.setConditionFailed(ctx, record, "ZoneNotFound", fmt.Sprintf("resolve zone %s: %s", record.Spec.ZoneName, err))
	}

	if record.Status.RecordID == "" {
		log.Info("Creating record in Poweradmin", "zone", record.Spec.ZoneName, "name", record.Spec.Name, "type", record.Spec.Type)
		return r.reconcileCreate(ctx, record, zoneID, paClient)
	}

	log.Info("Updating record in Poweradmin", "zone", record.Spec.ZoneName, "name", record.Spec.Name, "id", record.Status.RecordID)
	return r.reconcileUpdate(ctx, record, paClient)
}

func (r *DNSRecordReconciler) reconcileCreate(ctx context.Context, record *dnsv1alpha1.DNSRecord, zoneID int, paClient *poweradmin.Client) (ctrl.Result, error) {
	id, _, err := paClient.Record.Create(ctx, zoneID, poweradmin.RecordCreateOpts{
		Name:     record.Spec.Name,
		Type:     record.Spec.Type,
		Content:  record.Spec.Content,
		TTL:      record.Spec.TTL,
		Priority: record.Spec.Priority,
		Disabled: record.Spec.Disabled,
	})
	if err != nil {
		return ctrl.Result{}, r.setConditionFailed(ctx, record, reasonSyncFailed, fmt.Sprintf("create record: %s", err))
	}

	record.Status.RecordID = id
	record.Status.ZoneID = zoneID
	r.setConditionReady(record, reasonSynced, "Record created successfully")

	if err := r.Status().Update(ctx, record); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status after create: %w", err)
	}
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *DNSRecordReconciler) reconcileUpdate(ctx context.Context, record *dnsv1alpha1.DNSRecord, paClient *poweradmin.Client) (ctrl.Result, error) {
	_, _, err := paClient.Record.Update(ctx, record.Status.ZoneID, record.Status.RecordID, poweradmin.RecordUpdateOpts{
		Name:    record.Spec.Name,
		Type:    record.Spec.Type,
		Content: record.Spec.Content,
		TTL:     &record.Spec.TTL,
	})
	if err != nil {
		if poweradmin.IsNotFound(err) {
			record.Status.RecordID = ""
			record.Status.ZoneID = 0
			if err := r.Status().Update(ctx, record); err != nil {
				return ctrl.Result{}, fmt.Errorf("reset after external deletion: %w", err)
			}
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, r.setConditionFailed(ctx, record, reasonSyncFailed, fmt.Sprintf("update record: %s", err))
	}

	r.setConditionReady(record, reasonSynced, "Record synced successfully")

	if err := r.Status().Update(ctx, record); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status after update: %w", err)
	}
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *DNSRecordReconciler) reconcileDelete(ctx context.Context, record *dnsv1alpha1.DNSRecord, paClient *poweradmin.Client) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if record.Status.RecordID != "" {
		log.Info("Deleting record from Poweradmin", "name", record.Spec.Name, "id", record.Status.RecordID)
		_, err := paClient.Record.Delete(ctx, record.Status.ZoneID, record.Status.RecordID)
		if err != nil && !poweradmin.IsNotFound(err) {
			return ctrl.Result{}, r.setConditionFailed(ctx, record, reasonDeleteFailed, fmt.Sprintf("delete record: %s", err))
		}
	}

	controllerutil.RemoveFinalizer(record, recordFinalizerName)
	if err := r.Update(ctx, record); err != nil {
		return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
	}

	log.Info("Record deleted", "name", record.Spec.Name)
	return ctrl.Result{}, nil
}

func (r *DNSRecordReconciler) resolveZoneID(ctx context.Context, zoneName string, paClient *poweradmin.Client) (int, error) {
	zone, _, err := paClient.Zone.GetByName(ctx, zoneName)
	if err != nil {
		return 0, err
	}
	return zone.ID, nil
}

// buildClient returns a Poweradmin client. In tests, NewClient can be overridden.
func (r *DNSRecordReconciler) buildClient(ctx context.Context, namespace string) (*poweradmin.Client, error) {
	if r.NewClient != nil {
		return r.NewClient("", "")
	}
	return r.newPoweradminClient(ctx, namespace)
}

// newPoweradminClient reads credentials from the namespace Secret and builds a client.
func (r *DNSRecordReconciler) newPoweradminClient(ctx context.Context, namespace string) (*poweradmin.Client, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      r.CredentialsSecretName,
		Namespace: namespace,
	}, secret); err != nil {
		return nil, fmt.Errorf("get credentials secret %q in namespace %q: %w", r.CredentialsSecretName, namespace, err)
	}

	url := string(secret.Data["POWERADMIN_URL"])
	apiKey := string(secret.Data["POWERADMIN_API_KEY"])

	if url == "" || apiKey == "" {
		return nil, fmt.Errorf("secret %q in namespace %q must contain POWERADMIN_URL and POWERADMIN_API_KEY", r.CredentialsSecretName, namespace)
	}

	return poweradmin.NewClient(
		poweradmin.WithBaseURL(url),
		poweradmin.WithAPIKey(apiKey),
	)
}

func (r *DNSRecordReconciler) setConditionReady(record *dnsv1alpha1.DNSRecord, reason, message string) {
	meta.SetStatusCondition(&record.Status.Conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: record.Generation,
	})
}

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
