// Example: Production webhook with defaulting and cross-field validation.
// Pattern from model-registry-operator: OneOf validation + computed defaults.

package v1alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var dblog = logf.Log.WithName("databasecluster-resource")

func (r *DatabaseCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}

// --- Defaulting ---

var _ webhook.Defaulter = &DatabaseCluster{}

func (r *DatabaseCluster) Default() {
	dblog.Info("default", "name", r.Name)

	if r.Spec.Replicas == 0 {
		r.Spec.Replicas = 3
	}
	if r.Spec.Version == "" {
		r.Spec.Version = "16"
	}
	if r.Spec.Backup != nil && r.Spec.Backup.RetentionDays == 0 {
		r.Spec.Backup.RetentionDays = 7
	}
}

// --- Validation ---

var _ webhook.Validator = &DatabaseCluster{}

func (r *DatabaseCluster) ValidateCreate() (admission.Warnings, error) {
	dblog.Info("validate create", "name", r.Name)
	return r.validate()
}

func (r *DatabaseCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	dblog.Info("validate update", "name", r.Name)

	oldCluster := old.(*DatabaseCluster)

	// Immutable field check
	if r.Spec.Storage.StorageClassName != nil && oldCluster.Spec.Storage.StorageClassName != nil {
		if *r.Spec.Storage.StorageClassName != *oldCluster.Spec.Storage.StorageClassName {
			return nil, fmt.Errorf("storageClassName is immutable after creation")
		}
	}

	return r.validate()
}

func (r *DatabaseCluster) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

// validate contains shared validation logic.
func (r *DatabaseCluster) validate() (admission.Warnings, error) {
	var warnings admission.Warnings

	if r.Spec.Replicas > 10 {
		return nil, fmt.Errorf("replicas must not exceed 10, got %d", r.Spec.Replicas)
	}

	if r.Spec.Replicas%2 == 0 {
		warnings = append(warnings, "even replica count may cause split-brain; consider odd numbers")
	}

	if r.Spec.Backup != nil && r.Spec.Backup.RetentionDays > 30 {
		return nil, fmt.Errorf("backup retention must not exceed 30 days, got %d", r.Spec.Backup.RetentionDays)
	}

	return warnings, nil
}
