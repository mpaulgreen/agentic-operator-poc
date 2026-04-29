# Error Handling Patterns

## handleError Helper

Central error handler that records events, updates status, and requeues:

```go
func (r *Reconciler) handleError(ctx context.Context, cr *v1alpha1.MyKind, err error, msg string) (ctrl.Result, error) {
    log.FromContext(ctx).Error(err, msg)

    r.Recorder.Event(cr, corev1.EventTypeWarning, "ReconcileError", fmt.Sprintf("%s: %v", msg, err))

    cr.Status.Phase = "Failed"
    if updateErr := r.Status().Update(ctx, cr); updateErr != nil {
        log.FromContext(ctx).Error(updateErr, "Failed to update status after error")
    }

    return ctrl.Result{RequeueAfter: 30 * time.Second}, err
}
```

## Requeue Decision Tree

```
Error is transient (network, conflict)?
  → return ctrl.Result{}, err                    // exponential backoff

Error is permanent (invalid spec)?
  → return ctrl.Result{}, reconcile.TerminalError(err)  // no retry

Waiting for async operation?
  → return ctrl.Result{RequeueAfter: 10*time.Second}, nil  // poll

Everything succeeded?
  → return ctrl.Result{}, nil                    // done
```

## Sequential vs Parallel Error Handling

**Sequential** (default — fail-fast):
```go
if err := r.reconcileSecret(ctx, cr); err != nil {
    return r.handleError(ctx, cr, err, "Secret failed")
}
if err := r.reconcileService(ctx, cr); err != nil {
    return r.handleError(ctx, cr, err, "Service failed")
}
```
Use when resources have dependencies (Service before StatefulSet).

**Aggregate** (for independent resources):
```go
var errs []error
if err := r.reconcileEditorRole(ctx, cr); err != nil {
    errs = append(errs, err)
}
if err := r.reconcileViewerRole(ctx, cr); err != nil {
    errs = append(errs, err)
}
if len(errs) > 0 {
    return r.handleError(ctx, cr, errors.Join(errs...), "RBAC roles failed")
}
```
Use when resources are independent (editor and viewer roles).

## Status Update on Error

Always update status before returning an error:
```go
cr.Status.Phase = "Failed"
r.Status().Update(ctx, cr)  // best-effort, don't fail on status update failure
```

This ensures the user sees "Failed" phase even if the controller restarts.
