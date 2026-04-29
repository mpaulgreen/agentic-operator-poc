// Example: Complex reconciler with generic createOrUpdate (model-registry pattern).
// For operators managing 10+ resource types, use a generic helper instead of
// individual reconcileX methods.

package controller

// Generic createOrUpdate pattern:
//
// type OperationResult string
// const (
//     ResourceCreated   OperationResult = "created"
//     ResourceUpdated   OperationResult = "updated"
//     ResourceUnchanged OperationResult = "unchanged"
// )
//
// func (r *Reconciler) createOrUpdate(ctx context.Context, desired client.Object) (OperationResult, error) {
//     existing := desired.DeepCopyObject().(client.Object)
//     err := r.Get(ctx, client.ObjectKeyFromObject(desired), existing)
//     if err != nil {
//         if errors.IsNotFound(err) {
//             return ResourceCreated, r.Create(ctx, desired)
//         }
//         return ResourceUnchanged, err
//     }
//     // Preserve resourceVersion for update
//     desired.SetResourceVersion(existing.GetResourceVersion())
//     return ResourceUpdated, r.Update(ctx, desired)
// }
//
// Usage in Reconcile():
//
// resources := []client.Object{
//     r.serviceAccountForRegistry(cr),
//     r.roleForRegistry(cr),
//     r.roleBindingForRegistry(cr),
//     r.serviceForRegistry(cr),
//     r.deploymentForRegistry(cr),
// }
//
// for _, obj := range resources {
//     controllerutil.SetControllerReference(cr, obj, r.Scheme)
//     result, err := r.createOrUpdate(ctx, obj)
//     if err != nil { return r.handleError(ctx, cr, err, "Failed") }
//     if result != ResourceUnchanged {
//         r.Recorder.Event(cr, "Normal", string(result), obj.GetName())
//     }
// }
//
// Conditional resources (feature flags):
//
// if r.IsOpenShift {
//     resources = append(resources, r.routeForRegistry(cr))
// }
// if r.HasIstio {
//     resources = append(resources,
//         r.virtualServiceForRegistry(cr),
//         r.destinationRuleForRegistry(cr),
//     )
// }
