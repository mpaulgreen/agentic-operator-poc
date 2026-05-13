/*
Copyright 2026.

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

package v1beta1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var mongoclusterlog = logf.Log.WithName("mongocluster-v1beta1-resource")

func (r *MongoCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-database-mongodb-example-com-v1beta1-mongocluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=database.mongodb.example.com,resources=mongoclusters,verbs=create;update,versions=v1beta1,name=mmongoclusterv1beta1.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &MongoCluster{}

func (r *MongoCluster) Default() {
	mongoclusterlog.Info("default", "name", r.Name)

	if r.Spec.Replicas == 0 {
		r.Spec.Replicas = 3
	}
	if r.Spec.Version == "" {
		r.Spec.Version = "7.0"
	}
	if r.Spec.Backup != nil && r.Spec.Backup.Enabled && r.Spec.Backup.RetentionDays == 0 {
		r.Spec.Backup.RetentionDays = 7
	}
}

//+kubebuilder:webhook:path=/validate-database-mongodb-example-com-v1beta1-mongocluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=database.mongodb.example.com,resources=mongoclusters,verbs=create;update,versions=v1beta1,name=vmongoclusterv1beta1.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &MongoCluster{}

func (r *MongoCluster) ValidateCreate() (admission.Warnings, error) {
	mongoclusterlog.Info("validate create", "name", r.Name)
	return nil, r.validateMongoCluster()
}

func (r *MongoCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	mongoclusterlog.Info("validate update", "name", r.Name)

	oldCluster, ok := old.(*MongoCluster)
	if ok {
		oldSize, err := resource.ParseQuantity(oldCluster.Spec.Storage.Size)
		if err != nil {
			return nil, fmt.Errorf("invalid old storage size %q: %v", oldCluster.Spec.Storage.Size, err)
		}
		newSize, err := resource.ParseQuantity(r.Spec.Storage.Size)
		if err != nil {
			return nil, fmt.Errorf("invalid new storage size %q: %v", r.Spec.Storage.Size, err)
		}
		if newSize.Cmp(oldSize) < 0 {
			return nil, fmt.Errorf("storage size cannot be reduced from %s to %s", oldCluster.Spec.Storage.Size, r.Spec.Storage.Size)
		}
	}

	return nil, r.validateMongoCluster()
}

func (r *MongoCluster) ValidateDelete() (admission.Warnings, error) {
	mongoclusterlog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *MongoCluster) validateMongoCluster() error {
	if r.Spec.Replicas < 1 {
		return fmt.Errorf("replicas must be at least 1, got %d", r.Spec.Replicas)
	}
	if r.Spec.Replicas%2 == 0 {
		return fmt.Errorf("replicas must be odd for replica set elections, got %d", r.Spec.Replicas)
	}
	if r.Spec.Auth != nil {
		if r.Spec.Auth.AdminPassword != "" && r.Spec.Auth.ExistingSecret != "" {
			return fmt.Errorf("auth.adminPassword and auth.existingSecret are mutually exclusive")
		}
	}
	if r.Spec.Backup != nil && r.Spec.Backup.RetentionDays > 30 {
		return fmt.Errorf("backup.retentionDays must be at most 30, got %d", r.Spec.Backup.RetentionDays)
	}
	if r.Spec.Sharding != nil && r.Spec.Sharding.Enabled {
		if r.Spec.Sharding.Shards < 1 {
			return fmt.Errorf("sharding.shards must be at least 1 when sharding is enabled, got %d", r.Spec.Sharding.Shards)
		}
	}
	return nil
}
