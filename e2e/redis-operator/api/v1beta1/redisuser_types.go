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

// RedisUserSpec defines the desired state of RedisUser.
type RedisUserSpec struct {
	// Username is the Redis ACL username.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern=`^[a-zA-Z][a-zA-Z0-9_-]*$`
	Username string `json:"username"`

	// Permissions is a list of Redis ACL rules (e.g., "+@read", "~key:*").
	// +optional
	Permissions []string `json:"permissions,omitempty"`

	// ClusterRef is the name of the RedisCluster this user belongs to.
	// +kubebuilder:validation:MinLength=1
	ClusterRef string `json:"clusterRef"`

	// PasswordSecret is the name of an existing Secret containing the user password.
	// If empty, the operator generates a random password.
	// +optional
	PasswordSecret string `json:"passwordSecret,omitempty"`
}

// RedisUserStatus defines the observed state of RedisUser.
type RedisUserStatus struct {
	// Phase represents the current lifecycle phase of the RedisUser.
	// +kubebuilder:validation:Enum=Pending;Active;Failed
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the resource's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// PasswordSecretName is the name of the Secret containing the user password.
	PasswordSecretName string `json:"passwordSecretName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Username",type=string,JSONPath=`.spec.username`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RedisUser is the Schema for the redisusers API.
type RedisUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisUserSpec   `json:"spec,omitempty"`
	Status RedisUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RedisUserList contains a list of RedisUser.
type RedisUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisUser{}, &RedisUserList{})
}
