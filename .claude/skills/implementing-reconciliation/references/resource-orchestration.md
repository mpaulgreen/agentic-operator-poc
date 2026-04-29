# Resource Orchestration

## Dependency Ordering

Resources must be created in dependency order:

```
1. Secrets       (referenced by pods)
2. ConfigMaps    (referenced by pods)
3. Services      (networking, before StatefulSet for DNS)
4. StatefulSet    (depends on Secrets, ConfigMaps, Services)
```

If you create StatefulSet before its Secret exists, the pod will fail to start.

## Resource Builders (Pure Functions)

Separate resource construction from reconciliation:

```go
func statefulSetForCluster(cr *v1alpha1.RedisCluster) *appsv1.StatefulSet {
    replicas := cr.Spec.Replicas
    return &appsv1.StatefulSet{
        ObjectMeta: metav1.ObjectMeta{
            Name:      cr.Name,
            Namespace: cr.Namespace,
            Labels:    labelsForCluster(cr),
        },
        Spec: appsv1.StatefulSetSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: labelsForCluster(cr),
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{Labels: labelsForCluster(cr)},
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{{
                        Name:  "redis",
                        Image: fmt.Sprintf("redis:%s", cr.Spec.Version),
                        Ports: []corev1.ContainerPort{{ContainerPort: 6379}},
                    }},
                },
            },
        },
    }
}
```

**Rules**: No API calls. No side effects. Takes CR, returns desired object. Independently testable.

## Labels

Consistent labels on all managed resources:

```go
func labelsForCluster(cr *v1alpha1.RedisCluster) map[string]string {
    return map[string]string{
        "app.kubernetes.io/name":       "redis",
        "app.kubernetes.io/instance":   cr.Name,
        "app.kubernetes.io/managed-by": "redis-operator",
        "app.kubernetes.io/part-of":    cr.Name,
    }
}
```

## Owner References

```go
controllerutil.SetControllerReference(cr, object, r.Scheme)
```

Sets `metadata.ownerReferences[0]` with `controller: true`, `blockOwnerDeletion: true`. One controller reference per object (cannot have two controllers).

## Scaling to 10+ Resources

For complex operators, use a generic helper:

```go
func (r *Reconciler) createOrUpdate(ctx context.Context, desired client.Object) error {
    existing := desired.DeepCopyObject().(client.Object)
    err := r.Get(ctx, client.ObjectKeyFromObject(desired), existing)
    if err != nil {
        if errors.IsNotFound(err) {
            return r.Create(ctx, desired)
        }
        return err
    }
    desired.SetResourceVersion(existing.GetResourceVersion())
    return r.Update(ctx, desired)
}
```
