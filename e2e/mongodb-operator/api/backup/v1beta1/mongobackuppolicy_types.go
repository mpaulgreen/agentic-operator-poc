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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MongoBackupPolicySpec defines the desired state of MongoBackupPolicy.
type MongoBackupPolicySpec struct {
	// ClusterRef is the name of the MongoCluster to back up.
	// +kubebuilder:validation:MinLength=1
	ClusterRef string `json:"clusterRef"`

	// Schedule is a cron expression for backup frequency.
	// +kubebuilder:validation:MinLength=1
	Schedule string `json:"schedule"`

	// RetentionDays is the number of days to retain backups.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=90
	// +kubebuilder:default=30
	RetentionDays int32 `json:"retentionDays,omitempty"`

	// StorageSize is the PVC size for storing backups.
	// +kubebuilder:validation:Pattern=`^[0-9]+[KMGT]i$`
	StorageSize string `json:"storageSize"`
}

// MongoBackupPolicyStatus defines the observed state of MongoBackupPolicy.
type MongoBackupPolicyStatus struct {
	// Phase represents the current lifecycle phase.
	// +kubebuilder:validation:Enum=Pending;Active;Failed
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// LastBackupTime is when the last backup completed.
	// +optional
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`

	// NextBackupTime is the next scheduled backup time.
	// +optional
	NextBackupTime *metav1.Time `json:"nextBackupTime,omitempty"`

	// BackupCount is the total number of completed backups.
	BackupCount int32 `json:"backupCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef`
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// MongoBackupPolicy is the Schema for the mongobackuppolicies API.
type MongoBackupPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoBackupPolicySpec   `json:"spec,omitempty"`
	Status MongoBackupPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MongoBackupPolicyList contains a list of MongoBackupPolicy.
type MongoBackupPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoBackupPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MongoBackupPolicy{}, &MongoBackupPolicyList{})
}
