// Example: Server-Side Apply reconciler (modern alternative to check-create).
// SSA reconstructs desired state from scratch and applies it. No Get-then-Update race.

package controller

// Server-Side Apply pattern:
//
// func (r *Reconciler) reconcileDeployment(ctx context.Context, cr *v1alpha1.MyKind) error {
//     desired := r.deploymentForKind(cr)  // pure function
//
//     // Apply with SSA — creates if absent, updates if changed, no-op if same
//     err := r.Patch(ctx, desired,
//         client.Apply,
//         client.FieldOwner("my-controller"),
//         client.ForceOwnership,
//     )
//     if err != nil {
//         r.Recorder.Event(cr, "Warning", "DeploymentFailed", err.Error())
//         return err
//     }
//
//     r.Recorder.Event(cr, "Normal", "DeploymentApplied", desired.Name)
//     return nil
// }
//
// Key differences from check-create:
//   - No Get() call needed (SSA handles existence check)
//   - No errors.IsNotFound() check
//   - No SetResourceVersion()
//   - Uses client.Apply patch strategy instead of Create/Update
//   - FieldOwner prevents conflicts between controllers
//   - ForceOwnership overrides other field managers
//
// Caveats:
//   - Fake client (for testing) does not support SSA
//   - Must include ALL managed fields in every apply (omitted = removed)
//   - More complex for partial updates (e.g., updating just replicas)
//
// When to use SSA vs check-create:
//   - SSA: New controllers, clean codebase, no fake-client tests
//   - Check-create: Existing operators, need fake-client testing, simple cases
