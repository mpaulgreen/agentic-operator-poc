# Status Conventions

## Always Use metav1.Condition

Use the standard Kubernetes condition type, not custom structs:

```go
import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type Status struct {
    Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}
```

The `patchStrategy:"merge"` and `patchMergeKey:"type"` annotations enable server-side apply to merge conditions correctly.

## Standard Condition Types

| Type | When True | When False |
|------|-----------|------------|
| `Available` | Resource is ready and serving | Resource is not ready |
| `Progressing` | Reconciliation is in progress | Reconciliation is complete or not started |
| `Degraded` | Resource has issues but is partially functional | Resource is healthy |
| `Ready` | All sub-resources are ready | Some sub-resources are not ready |

Define as constants:
```go
const (
    ConditionTypeAvailable   = "Available"
    ConditionTypeProgressing = "Progressing"
    ConditionTypeDegraded    = "Degraded"
)
```

## Condition Fields

```go
metav1.Condition{
    Type:               "Available",
    Status:             metav1.ConditionTrue,   // True, False, or Unknown
    Reason:             "ResourcesReady",       // PascalCase, machine-readable
    Message:            "All pods are running",  // Human-readable
    LastTransitionTime: metav1.Now(),            // When status last changed
    ObservedGeneration: resource.Generation,     // Which spec version this reflects
}
```

## Setting Conditions in Controller

```go
import "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
import "k8s.io/apimachinery/pkg/api/meta"

meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
    Type:               ConditionTypeAvailable,
    Status:             metav1.ConditionTrue,
    Reason:             "ReconcileSuccess",
    Message:            "All resources reconciled successfully",
    ObservedGeneration: resource.Generation,
})
```

## Phase Enum Pattern

Optional — use when a simple lifecycle state is useful alongside conditions:

```go
type Status struct {
    // +kubebuilder:validation:Enum=Pending;Initializing;Running;Failed;Terminating
    Phase string `json:"phase,omitempty"`

    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

**When to use Phase vs Conditions-only:**
- Use Phase for simple lifecycle (database: Pending → Running → Failed)
- Use Conditions-only for complex state (multiple independent dimensions)
- Many modern operators use both: Phase for high-level, Conditions for detail

## Print Columns for Conditions

```go
// Show the Available condition's status in kubectl get
// +kubebuilder:printcolumn:name="Available",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].status`

// Show reason in wide output
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].reason`,priority=1
```

## ObservedGeneration

Always set `ObservedGeneration` when updating conditions. This lets users know whether the controller has processed the latest spec change:

```go
if resource.Status.ObservedGeneration != resource.Generation {
    // Spec has changed, need to reconcile
}
```
