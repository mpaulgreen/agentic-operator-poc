// Example: Simple reconciler with 4 resources (database-operator pattern).
// Shows three-phase reconciliation, check-create idempotency, owner refs,
// event recording, finalizers, and status updates.

package controller

// Reconcile structure (three phases):
//
// func (r *DatabaseClusterReconciler) Reconcile(ctx, req) (ctrl.Result, error) {
//     // PHASE 1: FETCH
//     cr := &v1alpha1.DatabaseCluster{}
//     r.Get(ctx, req.NamespacedName, cr)
//     if !cr.DeletionTimestamp.IsZero() { return r.handleDeletion(ctx, cr) }
//
//     // PHASE 2: ORCHESTRATE (dependency order)
//     // Add finalizer
//     if !controllerutil.ContainsFinalizer(cr, finalizerName) { ... }
//
//     r.reconcileSecret(ctx, cr)      // 1. credentials
//     r.reconcileConfigMap(ctx, cr)   // 2. configuration
//     r.reconcileService(ctx, cr)     // 3. networking
//     r.reconcileStatefulSet(ctx, cr) // 4. workload (depends on 1,2,3)
//
//     // PHASE 3: STATUS
//     return r.updateStatus(ctx, cr)
// }
//
// Check-create pattern (repeated for each resource):
//
// func (r *Reconciler) reconcileSecret(ctx, cr) error {
//     existing := &corev1.Secret{}
//     err := r.Get(ctx, key, existing)
//     if err == nil { return nil }           // EXISTS
//     if !errors.IsNotFound(err) { return err } // ERROR
//
//     desired := r.secretForCluster(cr)      // BUILD
//     controllerutil.SetControllerReference(cr, desired, r.Scheme) // OWNER REF
//     r.Create(ctx, desired)                 // CREATE
//     r.Recorder.Event(cr, "Normal", "SecretCreated", name) // EVENT
//     return nil
// }
//
// File split:
//   controller.go     — Reconcile() + SetupWithManager() + RBAC
//   reconcilers.go    — reconcileSecret/ConfigMap/Service/StatefulSet
//   status.go         — updateStatus() + updatePhase()
//   conditions.go     — setCondition() + convenience setters
//   helpers.go        — labelsForCluster() + generatePassword()
