# Finalizer Lifecycle

## Why Finalizers

Without a finalizer, deleting a CR immediately removes it — your controller never gets a chance to clean up external resources (databases, cloud resources, DNS records).

With a finalizer, deletion is blocked until your controller removes the finalizer.

## Add on First Reconcile

```go
const finalizerName = "cache.redis.example.com/finalizer"

// In Reconcile(), after fetching CR
if !controllerutil.ContainsFinalizer(cr, finalizerName) {
    controllerutil.AddFinalizer(cr, finalizerName)
    if err := r.Update(ctx, cr); err != nil {
        return ctrl.Result{}, err
    }
}
```

## Handle Deletion

```go
// In Reconcile(), after fetching CR
if !cr.DeletionTimestamp.IsZero() {
    return r.handleDeletion(ctx, cr)
}
```

```go
func (r *Reconciler) handleDeletion(ctx context.Context, cr *v1alpha1.MyKind) (ctrl.Result, error) {
    if !controllerutil.ContainsFinalizer(cr, finalizerName) {
        return ctrl.Result{}, nil
    }

    // Perform cleanup
    r.Recorder.Event(cr, corev1.EventTypeNormal, "Deleting", "Starting cleanup")

    // ... cleanup external resources here ...

    // Remove finalizer
    controllerutil.RemoveFinalizer(cr, finalizerName)
    if err := r.Update(ctx, cr); err != nil {
        return ctrl.Result{}, err
    }

    r.Recorder.Event(cr, corev1.EventTypeNormal, "Deleted", "Cleanup completed")
    return ctrl.Result{}, nil
}
```

## Common Pitfalls

**Race condition**: Always re-fetch the CR before removing the finalizer if multiple reconciles might be in-flight:
```go
// Re-fetch to get latest version
if err := r.Get(ctx, client.ObjectKeyFromObject(cr), cr); err != nil {
    return ctrl.Result{}, err
}
controllerutil.RemoveFinalizer(cr, finalizerName)
if err := r.Update(ctx, cr); err != nil {
    return ctrl.Result{}, err
}
```

**Stuck finalizer**: If cleanup fails permanently, the CR is stuck in "Terminating". Always ensure cleanup eventually completes or has a timeout.

**Owner references vs finalizers**: Owner references handle cascade deletion of owned resources automatically. Finalizers are only needed for resources NOT owned (external systems, cross-namespace resources).
