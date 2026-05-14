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

// ElasticsearchClusterSpec defines the desired state of ElasticsearchCluster.
type ElasticsearchClusterSpec struct {
	// Replicas is the number of Elasticsearch data nodes.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=9
	// +kubebuilder:default=3
	Replicas int32 `json:"replicas,omitempty"`

	// Version is the Elasticsearch version to deploy.
	// +kubebuilder:validation:Enum="8.12";"8.14"
	// +kubebuilder:default="8.14"
	Version string `json:"version,omitempty"`

	// Storage defines the persistent storage configuration.
	Storage StorageSpec `json:"storage"`

	// Resources defines the CPU and memory resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Auth defines the optional authentication configuration.
	// +optional
	Auth *AuthSpec `json:"auth,omitempty"`

	// Backup defines the optional snapshot backup configuration.
	// +optional
	Backup *BackupSpec `json:"backup,omitempty"`
}

// StorageSpec defines storage configuration.
type StorageSpec struct {
	// Size is the storage volume size (e.g., "10Gi").
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMGT]i$`
	Size string `json:"size"`

	// StorageClassName is the name of the StorageClass to use.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// AuthSpec defines authentication configuration.
type AuthSpec struct {
	// AdminPassword is the elastic superuser password. Mutually exclusive with ExistingSecret.
	// +optional
	AdminPassword string `json:"adminPassword,omitempty"`

	// ExistingSecret is the name of an existing Secret containing the password.
	// Mutually exclusive with AdminPassword.
	// +optional
	ExistingSecret string `json:"existingSecret,omitempty"`
}

// BackupSpec defines snapshot backup configuration.
type BackupSpec struct {
	// Enabled indicates whether backup is active.
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Schedule is a cron expression for snapshot frequency.
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// RetentionDays is the number of days to retain snapshots.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=30
	// +kubebuilder:default=7
	RetentionDays int32 `json:"retentionDays,omitempty"`
}

// ElasticsearchClusterStatus defines the observed state of ElasticsearchCluster.
type ElasticsearchClusterStatus struct {
	// Phase represents the current lifecycle phase.
	// +kubebuilder:validation:Enum=Pending;Initializing;Running;Failed;Degraded
	Phase string `json:"phase,omitempty"`

	// ReadyReplicas is the number of Elasticsearch nodes that are ready.
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// CurrentVersion is the currently running Elasticsearch version.
	CurrentVersion string `json:"currentVersion,omitempty"`

	// HttpEndpoint is the HTTP service connection endpoint.
	HttpEndpoint string `json:"httpEndpoint,omitempty"`

	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.currentVersion`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ElasticsearchCluster is the Schema for the elasticsearchclusters API.
type ElasticsearchCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticsearchClusterSpec   `json:"spec,omitempty"`
	Status ElasticsearchClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ElasticsearchClusterList contains a list of ElasticsearchCluster.
type ElasticsearchClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticsearchCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticsearchCluster{}, &ElasticsearchClusterList{})
}
