// Example: Status with Kubernetes-standard Conditions pattern.
// Shows condition types, reasons, and how to set them in a controller.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition type constants for the operator.
const (
	// ConditionTypeAvailable indicates the resource is ready and serving.
	ConditionTypeAvailable = "Available"

	// ConditionTypeProgressing indicates the resource is being created or updated.
	ConditionTypeProgressing = "Progressing"

	// ConditionTypeDegraded indicates the resource has issues but is partially functional.
	ConditionTypeDegraded = "Degraded"
)

// Condition reason constants.
const (
	ReasonReconciling       = "Reconciling"
	ReasonReconcileSuccess  = "ReconcileSuccess"
	ReasonReconcileError    = "ReconcileError"
	ReasonResourcesReady    = "ResourcesReady"
	ReasonResourcesNotReady = "ResourcesNotReady"
)

// ExampleStatus shows the recommended Status struct pattern.
type ExampleStatus struct {
	// Conditions represent the latest available observations of the resource's state.
	// Standard condition types: Available, Progressing, Degraded.
	//
	// Use meta.SetStatusCondition() to set conditions:
	//   meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
	//       Type:               ConditionTypeAvailable,
	//       Status:             metav1.ConditionTrue,
	//       Reason:             ReasonResourcesReady,
	//       Message:            "All resources are ready",
	//       ObservedGeneration: resource.Generation,
	//   })
	//
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// PrintColumn example for extracting a specific condition's status:
//
// +kubebuilder:printcolumn:name="Available",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].reason`,priority=1
