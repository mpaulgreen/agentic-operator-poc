# Cluster-Scoped Resource Design Patterns

## When to Use Cluster-Scoped

Use cluster-scoped CRDs when the resource:
- Applies to the entire cluster (global configuration)
- Spans multiple namespaces (cross-namespace policies)
- Manages cluster-level infrastructure (nodes, storage classes)
- Represents a singleton (one instance per cluster)

Examples: `ClusterRole`, `ClusterRedisConfig`, `GlobalPolicy`, `InfrastructureNode`

## Markers

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

type ClusterRedisConfig struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              ClusterRedisConfigSpec   `json:"spec,omitempty"`
    Status            ClusterRedisConfigStatus `json:"status,omitempty"`
}
```

## PROJECT Entry

Cluster-scoped resources omit `namespaced: true`:
```yaml
- api:
    crdVersion: v1          # no "namespaced: true" line
  controller: true
  kind: ClusterRedisConfig
```

## RBAC Implications

- Always uses `ClusterRole` and `ClusterRoleBinding` (not `Role`/`RoleBinding`)
- No namespace in ServiceAccount subject
- Controller needs cluster-wide watch permissions

## Owner Reference Limitations

- Cluster-scoped resources **cannot** set owner references to namespaced resources
- Cluster-scoped resources **can** own other cluster-scoped resources
- For cross-namespace relationships, use labels/annotations instead of owner refs

## Naming Convention

Prefix cluster-scoped kinds with "Cluster" to signal scope:
- `ClusterRedisConfig` (not `RedisConfig` at cluster scope)
- `ClusterPolicy` (not `Policy` at cluster scope)

This avoids confusion when both namespaced and cluster variants exist.

## Sample CR (no namespace)

```yaml
apiVersion: cache.redis.example.com/v1alpha1
kind: ClusterRedisConfig
metadata:
  name: global-redis-config    # no namespace field
spec:
  maxMemory: "256Mi"
```
