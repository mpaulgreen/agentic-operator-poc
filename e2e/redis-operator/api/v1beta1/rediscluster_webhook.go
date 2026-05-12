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

var redisclusterlog = logf.Log.WithName("rediscluster-v1beta1-resource")

func (r *RedisCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-cache-redis-example-com-v1beta1-rediscluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=cache.redis.example.com,resources=redisclusters,verbs=create;update,versions=v1beta1,name=mredisclusterv1beta1.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &RedisCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (r *RedisCluster) Default() {
	redisclusterlog.Info("default", "name", r.Name)

	if r.Spec.Replicas == 0 {
		r.Spec.Replicas = 3
	}
	if r.Spec.Version == "" {
		r.Spec.Version = "7.4"
	}
	if r.Spec.Sentinel != nil && r.Spec.Sentinel.Enabled && r.Spec.Sentinel.Replicas == 0 {
		r.Spec.Sentinel.Replicas = 3
	}
}

//+kubebuilder:webhook:path=/validate-cache-redis-example-com-v1beta1-rediscluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=cache.redis.example.com,resources=redisclusters,verbs=create;update,versions=v1beta1,name=vredisclusterv1beta1.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &RedisCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *RedisCluster) ValidateCreate() (admission.Warnings, error) {
	redisclusterlog.Info("validate create", "name", r.Name)
	return nil, r.validateRedisCluster()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *RedisCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	redisclusterlog.Info("validate update", "name", r.Name)

	oldCluster, ok := old.(*RedisCluster)
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

	return nil, r.validateRedisCluster()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *RedisCluster) ValidateDelete() (admission.Warnings, error) {
	redisclusterlog.Info("validate delete", "name", r.Name)
	return nil, nil
}

// validateRedisCluster validates common fields for create and update.
func (r *RedisCluster) validateRedisCluster() error {
	if r.Spec.Replicas < 1 {
		return fmt.Errorf("replicas must be at least 1, got %d", r.Spec.Replicas)
	}
	if r.Spec.Sentinel != nil && r.Spec.Sentinel.Enabled {
		if r.Spec.Sentinel.Replicas%2 == 0 {
			return fmt.Errorf("sentinel.replicas must be odd for quorum, got %d", r.Spec.Sentinel.Replicas)
		}
	}
	if r.Spec.Auth != nil {
		if r.Spec.Auth.Password != "" && r.Spec.Auth.ExistingSecret != "" {
			return fmt.Errorf("auth.password and auth.existingSecret are mutually exclusive")
		}
	}
	// TLS validation: if tls.enabled is true, either secretName or certManager must be provided
	if r.Spec.TLS != nil && r.Spec.TLS.Enabled {
		if r.Spec.TLS.SecretName == "" && !r.Spec.TLS.CertManager {
			return fmt.Errorf("tls.secretName is required when tls.enabled is true and tls.certManager is false")
		}
	}
	return nil
}
