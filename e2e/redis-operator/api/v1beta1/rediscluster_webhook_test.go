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
	"testing"
)

func TestDefault_ReplicasZero(t *testing.T) {
	cr := &RedisCluster{Spec: RedisClusterSpec{Storage: StorageSpec{Size: "1Gi"}}}
	cr.Default()
	if cr.Spec.Replicas != 3 {
		t.Errorf("expected replicas=3, got %d", cr.Spec.Replicas)
	}
}

func TestDefault_VersionEmpty(t *testing.T) {
	cr := &RedisCluster{Spec: RedisClusterSpec{Replicas: 3, Storage: StorageSpec{Size: "1Gi"}}}
	cr.Default()
	if cr.Spec.Version != "7.4" {
		t.Errorf("expected version=7.4, got %s", cr.Spec.Version)
	}
}

func TestDefault_SentinelReplicasWhenEnabled(t *testing.T) {
	cr := &RedisCluster{Spec: RedisClusterSpec{Replicas: 3, Version: "7.4", Storage: StorageSpec{Size: "1Gi"}, Sentinel: &SentinelSpec{Enabled: true}}}
	cr.Default()
	if cr.Spec.Sentinel.Replicas != 3 {
		t.Errorf("expected sentinel.replicas=3, got %d", cr.Spec.Sentinel.Replicas)
	}
}

func TestValidate_SentinelReplicasEven(t *testing.T) {
	cr := &RedisCluster{Spec: RedisClusterSpec{Replicas: 3, Version: "7.4", Storage: StorageSpec{Size: "1Gi"}, Sentinel: &SentinelSpec{Enabled: true, Replicas: 4}}}
	_, err := cr.ValidateCreate()
	if err == nil {
		t.Error("expected error for even sentinel replicas")
	}
}

func TestValidate_BothPasswordAndExistingSecret(t *testing.T) {
	cr := &RedisCluster{Spec: RedisClusterSpec{Replicas: 3, Version: "7.4", Storage: StorageSpec{Size: "1Gi"}, Auth: &AuthSpec{Password: "secret", ExistingSecret: "my-secret"}}}
	_, err := cr.ValidateCreate()
	if err == nil {
		t.Error("expected error for both auth.password and auth.existingSecret set")
	}
}

func TestValidate_ReplicasLessThanOne(t *testing.T) {
	cr := &RedisCluster{Spec: RedisClusterSpec{Replicas: 0, Version: "7.4", Storage: StorageSpec{Size: "1Gi"}}}
	_, err := cr.ValidateCreate()
	if err == nil {
		t.Error("expected error for replicas < 1")
	}
}

func TestValidate_StorageSizeReduction(t *testing.T) {
	cr := &RedisCluster{Spec: RedisClusterSpec{Replicas: 3, Version: "7.4", Storage: StorageSpec{Size: "5Gi"}}}
	old := &RedisCluster{Spec: RedisClusterSpec{Replicas: 3, Version: "7.4", Storage: StorageSpec{Size: "10Gi"}}}
	_, err := cr.ValidateUpdate(old)
	if err == nil {
		t.Error("expected error for storage size reduction")
	}
}

// TLS-specific validation tests

func TestValidate_TLSEnabledWithoutSecretOrCertManager(t *testing.T) {
	cr := &RedisCluster{
		Spec: RedisClusterSpec{
			Replicas: 3, Version: "7.4",
			Storage: StorageSpec{Size: "1Gi"},
			TLS:     &TLSSpec{Enabled: true},
		},
	}
	_, err := cr.ValidateCreate()
	if err == nil {
		t.Error("expected error for tls.enabled without secretName or certManager")
	}
}

func TestValidate_TLSEnabledWithSecretName(t *testing.T) {
	cr := &RedisCluster{
		Spec: RedisClusterSpec{
			Replicas: 3, Version: "7.4",
			Storage: StorageSpec{Size: "1Gi"},
			TLS:     &TLSSpec{Enabled: true, SecretName: "redis-tls"},
		},
	}
	_, err := cr.ValidateCreate()
	if err != nil {
		t.Errorf("expected no error for tls with secretName, got: %v", err)
	}
}

func TestValidate_TLSEnabledWithCertManager(t *testing.T) {
	cr := &RedisCluster{
		Spec: RedisClusterSpec{
			Replicas: 3, Version: "7.4",
			Storage: StorageSpec{Size: "1Gi"},
			TLS:     &TLSSpec{Enabled: true, CertManager: true},
		},
	}
	_, err := cr.ValidateCreate()
	if err != nil {
		t.Errorf("expected no error for tls with certManager, got: %v", err)
	}
}

func TestValidate_TLSDisabledNoError(t *testing.T) {
	cr := &RedisCluster{
		Spec: RedisClusterSpec{
			Replicas: 3, Version: "7.4",
			Storage: StorageSpec{Size: "1Gi"},
			TLS:     &TLSSpec{Enabled: false},
		},
	}
	_, err := cr.ValidateCreate()
	if err != nil {
		t.Errorf("expected no error for disabled tls, got: %v", err)
	}
}

func TestValidate_TLSNilNoError(t *testing.T) {
	cr := &RedisCluster{
		Spec: RedisClusterSpec{
			Replicas: 3, Version: "7.4",
			Storage: StorageSpec{Size: "1Gi"},
		},
	}
	_, err := cr.ValidateCreate()
	if err != nil {
		t.Errorf("expected no error for nil tls, got: %v", err)
	}
}
