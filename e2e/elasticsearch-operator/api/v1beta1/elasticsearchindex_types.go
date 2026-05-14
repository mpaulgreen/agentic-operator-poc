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

// ElasticsearchIndexSpec defines the desired state of ElasticsearchIndex.
type ElasticsearchIndexSpec struct {
	// IndexName is the Elasticsearch index name.
	IndexName string `json:"indexName"`

	// Shards is the number of primary shards.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=50
	// +kubebuilder:default=1
	Shards int32 `json:"shards,omitempty"`

	// Replicas is the number of replica shards.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=5
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	// ClusterRef is the name of the ElasticsearchCluster this index belongs to.
	ClusterRef string `json:"clusterRef"`
}

// ElasticsearchIndexStatus defines the observed state of ElasticsearchIndex.
type ElasticsearchIndexStatus struct {
	// Phase represents the current lifecycle phase.
	// +kubebuilder:validation:Enum=Pending;Active;Failed
	Phase string `json:"phase,omitempty"`

	// IndexReady indicates whether the index template is configured.
	IndexReady bool `json:"indexReady,omitempty"`

	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Index",type=string,JSONPath=`.spec.indexName`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ElasticsearchIndex is the Schema for the elasticsearchindices API.
type ElasticsearchIndex struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticsearchIndexSpec   `json:"spec,omitempty"`
	Status ElasticsearchIndexStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ElasticsearchIndexList contains a list of ElasticsearchIndex.
type ElasticsearchIndexList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticsearchIndex `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticsearchIndex{}, &ElasticsearchIndexList{})
}
