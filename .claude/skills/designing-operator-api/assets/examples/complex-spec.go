// Example: Complex Spec with nested types, embedded K8s types, optional fields.
// Level 3+ operator — multiple nested configs, storage, backup, resources.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseClusterSpec defines the desired state of DatabaseCluster.
type DatabaseClusterSpec struct {
	// Replicas is the number of database instances.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=3
	Replicas int32 `json:"replicas,omitempty"`

	// Version is the database engine version.
	// +kubebuilder:validation:Enum="14";"15";"16"
	// +kubebuilder:default="16"
	Version string `json:"version,omitempty"`

	// Storage defines persistent volume configuration.
	Storage StorageSpec `json:"storage"`

	// Resources defines CPU/memory requests and limits.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Backup defines optional backup configuration.
	// +optional
	Backup *BackupSpec `json:"backup,omitempty"`

	// EnableMetrics enables Prometheus metrics endpoint.
	// +kubebuilder:default=true
	EnableMetrics bool `json:"enableMetrics,omitempty"`
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

// BackupSpec defines backup configuration.
type BackupSpec struct {
	// Schedule is a cron expression for backup timing.
	// +kubebuilder:validation:Pattern=`^(\*|([0-5]?\d)) (\*|([01]?\d|2[0-3])) (\*|([012]?\d|3[01])) (\*|(0?[1-9]|1[0-2])) (\*|([0-6]))$`
	Schedule string `json:"schedule"`

	// RetentionDays is how long to keep backups.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=90
	// +kubebuilder:default=7
	RetentionDays int32 `json:"retentionDays,omitempty"`

	// Destination is where backups are stored.
	// +kubebuilder:validation:Enum=s3;local;gcs
	// +kubebuilder:default="local"
	Destination string `json:"destination,omitempty"`
}

// SecretKeyValue references a key within a Kubernetes Secret.
type SecretKeyValue struct {
	// Name is the Secret name.
	Name string `json:"name"`

	// Key is the key within the Secret.
	Key string `json:"key"`
}

type DatabaseClusterStatus struct {
	// Phase is the current lifecycle phase.
	// +kubebuilder:validation:Enum=Pending;Initializing;Running;Failed;Terminating
	Phase string `json:"phase,omitempty"`

	// ReadyReplicas is the number of ready database instances.
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// CurrentVersion is the running database version.
	CurrentVersion string `json:"currentVersion,omitempty"`

	// Endpoint is the connection string.
	Endpoint string `json:"endpoint,omitempty"`

	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

type DatabaseCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DatabaseClusterSpec   `json:"spec,omitempty"`
	Status            DatabaseClusterStatus `json:"status,omitempty"`
}
