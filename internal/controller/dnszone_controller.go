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

	poweradmin "contentways.dev/contentways/poweradmin-go/poweradmin"
	dnsv1alpha1 "contentways.dev/contentways/poweradmin-operator/api/v1alpha1"
)

const (
	finalizerName  = "dns.contentways.org/finalizer"
	conditionReady = "Ready"

	reasonSynced       = "Synced"
	reasonSyncFailed   = "SyncFailed"
	reasonDeleteFailed = "DeleteFailed"
)

// DNSZoneReconciler reconciles a DNSZone object.
type DNSZoneReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	CredentialsSecretName string
	// NewClient can be overridden in tests to inject a mock client.
	// If nil, credentials are read from the namespace Secret.
	NewClient func(url, apiKey string) (*poweradmin.Client, error)
}

// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnszones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnszones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dns.contentways.org,resources=dnszones/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile moves the current state of the DNSZone closer to the desired state.
func (r *DNSZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	zone := &dnsv1alpha1.DNSZone{}
	if err := r.Get(ctx, req.NamespacedName, zone); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	paClient, err := r.buildClient(ctx, req.Namespace)
	if err != nil {
		return ctrl.Result{}, r.setConditionFailed(ctx, zone, "CredentialsNotFound", err.Error())
	}

	if !zone.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, zone, paClient)
	}

	if !controllerutil.ContainsFinalizer(zone, finalizerName) {
		controllerutil.AddFinalizer(zone, finalizerName)
		if err := r.Update(ctx, zone); err != nil {
			return ctrl.Result{}, fmt.Errorf("add finalizer: %w", err)
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	if zone.Status.ZoneID == 0 {
		log.Info("Creating zone in Poweradmin", "zone", zone.Spec.Name)
		return r.reconcileCreate(ctx, zone, paClient)
	}

	log.Info("Updating zone in Poweradmin", "zone", zone.Spec.Name, "id", zone.Status.ZoneID)
	return r.reconcileUpdate(ctx, zone, paClient)
}

func (r *DNSZoneReconciler) reconcileCreate(ctx context.Context, zone *dnsv1alpha1.DNSZone, paClient *poweradmin.Client) (ctrl.Result, error) {
	id, _, err := paClient.Zone.Create(ctx, poweradmin.ZoneCreateOpts{
		Name:    zone.Spec.Name,
		Type:    poweradmin.ZoneType(zone.Spec.Type),
		Masters: zone.Spec.Masters,
	})
	if err != nil {
		return ctrl.Result{}, r.setConditionFailed(ctx, zone, reasonSyncFailed, fmt.Sprintf("create zone: %s", err))
	}

	// Create NS records for each nameserver
	for _, ns := range zone.Spec.Nameservers {
		_, _, err := paClient.Record.Create(ctx, id, poweradmin.RecordCreateOpts{
			Name:    zone.Spec.Name,
			Type:    "NS",
			Content: ns,
			TTL:     3600,
		})
		if err != nil {
			return ctrl.Result{}, r.setConditionFailed(ctx, zone, reasonSyncFailed, fmt.Sprintf("create NS record for %s: %s", ns, err))
		}
	}

	zone.Status.ZoneID = id
	r.setConditionReady(zone, reasonSynced, "Zone created successfully")

	if err := r.Status().Update(ctx, zone); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status after create: %w", err)
	}
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *DNSZoneReconciler) reconcileUpdate(ctx context.Context, zone *dnsv1alpha1.DNSZone, paClient *poweradmin.Client) (ctrl.Result, error) {
	zoneType := poweradmin.ZoneType(zone.Spec.Type)
	_, _, err := paClient.Zone.Update(ctx, zone.Status.ZoneID, poweradmin.ZoneUpdateOpts{
		Type:    &zoneType,
		Masters: &zone.Spec.Masters,
	})
	if err != nil {
		if poweradmin.IsNotFound(err) {
			zone.Status.ZoneID = 0
			if err := r.Status().Update(ctx, zone); err != nil {
				return ctrl.Result{}, fmt.Errorf("reset after external deletion: %w", err)
			}
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, r.setConditionFailed(ctx, zone, reasonSyncFailed, fmt.Sprintf("update zone: %s", err))
	}

	r.setConditionReady(zone, reasonSynced, "Zone synced successfully")

	if err := r.Status().Update(ctx, zone); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status after update: %w", err)
	}
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *DNSZoneReconciler) reconcileDelete(ctx context.Context, zone *dnsv1alpha1.DNSZone, paClient *poweradmin.Client) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if zone.Status.ZoneID != 0 {
		log.Info("Deleting zone from Poweradmin", "zone", zone.Spec.Name, "id", zone.Status.ZoneID)
		_, err := paClient.Zone.Delete(ctx, zone.Status.ZoneID)
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

// buildClient returns a Poweradmin client. In tests, NewClient can be overridden.
func (r *DNSZoneReconciler) buildClient(ctx context.Context, namespace string) (*poweradmin.Client, error) {
	if r.NewClient != nil {
		return r.NewClient("", "")
	}
	return r.newPoweradminClient(ctx, namespace)
}

// newPoweradminClient reads credentials from the namespace Secret and builds a client.
func (r *DNSZoneReconciler) newPoweradminClient(ctx context.Context, namespace string) (*poweradmin.Client, error) {
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

func (r *DNSZoneReconciler) setConditionReady(zone *dnsv1alpha1.DNSZone, reason, message string) {
	meta.SetStatusCondition(&zone.Status.Conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: zone.Generation,
	})
}

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
