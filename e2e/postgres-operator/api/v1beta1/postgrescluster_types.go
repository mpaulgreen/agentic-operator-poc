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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PostgresClusterSpec struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=5
	// +kubebuilder:default=3
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Enum="14";"15";"16"
	// +kubebuilder:default="16"
	Version string `json:"version,omitempty"`

	Storage StorageSpec `json:"storage"`

	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +optional
	Backup *BackupSpec `json:"backup,omitempty"`

	// +optional
	HA *HASpec `json:"ha,omitempty"`

	// +optional
	MaxMemory *resource.Quantity `json:"maxMemory,omitempty"`

	// +optional
	ConnectionPool *ConnectionPoolSpec `json:"connectionPool,omitempty"`
}

type StorageSpec struct {
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMGT]i$`
	Size string `json:"size"`

	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

type BackupSpec struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	Schedule string `json:"schedule"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=30
	// +kubebuilder:default=7
	RetentionDays int32 `json:"retentionDays,omitempty"`
}

type HASpec struct {
	// +kubebuilder:validation:Minimum=1
	// +optional
	MinAvailable *int32 `json:"minAvailable,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +optional
	MaxUnavailable *int32 `json:"maxUnavailable,omitempty"`

	// +kubebuilder:validation:Enum="preferred";"required"
	// +kubebuilder:default="preferred"
	AntiAffinityMode string `json:"antiAffinityMode,omitempty"`
}

type ConnectionPoolSpec struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=10
	PoolSize int32 `json:"poolSize,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	// +kubebuilder:default=100
	MaxClientConnections int32 `json:"maxClientConnections,omitempty"`

	// +kubebuilder:default="30s"
	IdleTimeout string `json:"idleTimeout,omitempty"`
}

type PostgresClusterStatus struct {
	// +kubebuilder:validation:Enum=Pending;Initializing;Running;Failed;Degraded
	Phase string `json:"phase,omitempty"`

	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	CurrentVersion string `json:"currentVersion,omitempty"`

	Endpoint string `json:"endpoint,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	PoolerReady bool `json:"poolerReady,omitempty"`

	PoolerEndpoint string `json:"poolerEndpoint,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.currentVersion`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

type PostgresCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresClusterSpec   `json:"spec,omitempty"`
	Status PostgresClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type PostgresClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresCluster{}, &PostgresClusterList{})
}
