# Reconciliation Architecture

## Reconcile() Signature

```go
func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx)
    // ...
}
```

`req.NamespacedName` identifies WHICH resource changed. The reconciler re-reads current state and computes desired state (level-based, not edge-based).

## Three-Phase Structure

### Phase 1: Fetch

```go
cr := &v1alpha1.MyKind{}
if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
    if errors.IsNotFound(err) {
        return ctrl.Result{}, nil  // Deleted, nothing to do
    }
    return ctrl.Result{}, err
}

if !cr.DeletionTimestamp.IsZero() {
    return r.handleDeletion(ctx, cr)
}
```

### Phase 2: Orchestrate

```go
// Finalizer
if !controllerutil.ContainsFinalizer(cr, finalizerName) {
    controllerutil.AddFinalizer(cr, finalizerName)
    if err := r.Update(ctx, cr); err != nil { return ctrl.Result{}, err }
}

// Reconcile sub-resources in dependency order
if err := r.reconcileSecret(ctx, cr); err != nil {
    return r.handleError(ctx, cr, err, "Secret failed")
}
if err := r.reconcileService(ctx, cr); err != nil {
    return r.handleError(ctx, cr, err, "Service failed")
}
if err := r.reconcileStatefulSet(ctx, cr); err != nil {
    return r.handleError(ctx, cr, err, "StatefulSet failed")
}
```

### Phase 3: Status

```go
return r.updateStatus(ctx, cr)
```

## SetupWithManager

```go
func (r *MyReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1alpha1.MyKind{}).
        Owns(&appsv1.StatefulSet{}).
        Owns(&corev1.Service{}).
        Owns(&corev1.Secret{}).
        WithEventFilter(predicate.GenerationChangedPredicate{}).
        Complete(r)
}
```

- `For()`: primary resource (your CR)
- `Owns()`: secondary resources (triggers reconcile when child changes)
- `WithEventFilter()`: skip status-only updates (only on primary, not Owns)

## Reconciler Struct

```go
type MyReconciler struct {
    client.Client
    Scheme   *runtime.Scheme
    Recorder record.EventRecorder
}
```

Always include `Recorder` for event recording. Set up in main.go:
```go
Recorder: mgr.GetEventRecorderFor("my-operator"),
```
