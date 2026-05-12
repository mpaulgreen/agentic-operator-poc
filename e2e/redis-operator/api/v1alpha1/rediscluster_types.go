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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisClusterSpec defines the desired state of RedisCluster.
type RedisClusterSpec struct {
	// Replicas is the number of Redis instances to run.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=6
	// +kubebuilder:default=3
	Replicas int32 `json:"replicas,omitempty"`

	// Version is the Redis major version to deploy.
	// +kubebuilder:validation:Enum="7.2";"7.4"
	// +kubebuilder:default="7.4"
	Version string `json:"version,omitempty"`

	// Storage defines the persistent storage configuration.
	Storage StorageSpec `json:"storage"`

	// Resources defines the CPU and memory resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Auth defines the optional authentication configuration.
	// +optional
	Auth *AuthSpec `json:"auth,omitempty"`
}

// StorageSpec defines storage configuration for Redis data.
type StorageSpec struct {
	// Size is the storage volume size (e.g., "10Gi").
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMGT]i$`
	Size string `json:"size"`

	// StorageClassName is the name of the StorageClass to use.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// AuthSpec defines authentication configuration for Redis.
type AuthSpec struct {
	// Password is the Redis requirepass value. Mutually exclusive with ExistingSecret.
	// +optional
	Password string `json:"password,omitempty"`

	// ExistingSecret is the name of an existing Secret containing the password.
	// Mutually exclusive with Password.
	// +optional
	ExistingSecret string `json:"existingSecret,omitempty"`
}

// RedisClusterStatus defines the observed state of RedisCluster.
type RedisClusterStatus struct {
	// Phase represents the current lifecycle phase of the RedisCluster.
	// +kubebuilder:validation:Enum=Pending;Initializing;Running;Failed;Degraded
	Phase string `json:"phase,omitempty"`

	// ReadyReplicas is the number of Redis instances that are ready.
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// CurrentVersion is the currently running Redis version.
	CurrentVersion string `json:"currentVersion,omitempty"`

	// MasterEndpoint is the client service connection endpoint.
	MasterEndpoint string `json:"masterEndpoint,omitempty"`

	// Conditions represent the latest available observations of the resource's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.currentVersion`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RedisCluster is the Schema for the redisclusters API.
type RedisCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisClusterSpec   `json:"spec,omitempty"`
	Status RedisClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RedisClusterList contains a list of RedisCluster.
type RedisClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisCluster{}, &RedisClusterList{})
}
