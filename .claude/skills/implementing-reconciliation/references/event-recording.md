# Event Recording

## Event Recorder Setup

```go
type MyReconciler struct {
    client.Client
    Scheme   *runtime.Scheme
    Recorder record.EventRecorder  // import "k8s.io/client-go/tools/record"
}
```

In main.go:
```go
Recorder: mgr.GetEventRecorderFor("my-operator"),
```

## Event Types

```go
corev1.EventTypeNormal  = "Normal"   // informational
corev1.EventTypeWarning = "Warning"  // something went wrong
```

## When to Record Events

| Situation | Type | Reason | Example |
|-----------|------|--------|---------|
| Resource created | Normal | `SecretCreated` | `"Created credentials Secret my-secret"` |
| Resource updated | Normal | `StatefulSetUpdated` | `"Updated replicas from 3 to 5"` |
| Resource creation failed | Warning | `SecretFailed` | `"Failed to create Secret: already exists"` |
| Phase transition | Normal | `PhaseChanged` | `"Phase changed from Initializing to Running"` |
| Cluster ready | Normal | `ClusterReady` | `"All 3/3 replicas are ready"` |
| Deletion started | Normal | `Deleting` | `"Starting cluster cleanup"` |
| Deletion completed | Normal | `Deleted` | `"Cluster cleanup completed"` |
| Reconcile error | Warning | `ReconcileError` | `"Failed to reconcile StatefulSet: timeout"` |

## Reason Conventions

- PascalCase: `SecretCreated`, not `secret_created`
- Past tense for completed actions: `Created`, `Updated`, `Deleted`
- Present participle for in-progress: `Deleting`, `Reconciling`
- Suffix `Failed` for errors: `SecretCreationFailed`

## Message Conventions

- Include the resource name: `"Created Secret my-cluster-credentials"`
- Include relevant values: `"Updated replicas from 3 to 5"`
- Include error details: `fmt.Sprintf("Failed to create: %v", err)`
- Keep concise — events are visible in `kubectl describe`

## Recording Pattern

```go
// Success
r.Recorder.Event(cr, corev1.EventTypeNormal, "SecretCreated",
    fmt.Sprintf("Created credentials Secret %s", secretName))

// Failure
r.Recorder.Event(cr, corev1.EventTypeWarning, "SecretFailed",
    fmt.Sprintf("Failed to create Secret %s: %v", secretName, err))
```
