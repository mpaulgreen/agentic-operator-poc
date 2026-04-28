# Type Design Patterns

## Spec Organization

### Simple Spec (Level 1)
Flat fields, no nesting. Good for operators managing a single resource type.
```go
type AppSpec struct {
    Replicas int32  `json:"replicas,omitempty"`
    Image    string `json:"image"`
    Port     int32  `json:"port,omitempty"`
}
```

### Complex Spec (Level 3+)
Group related fields into nested structs when 3+ fields belong together.
```go
type DatabaseSpec struct {
    Replicas  int32                        `json:"replicas,omitempty"`
    Version   string                       `json:"version,omitempty"`
    Storage   StorageSpec                  `json:"storage"`           // required nested
    Resources corev1.ResourceRequirements  `json:"resources,omitempty"` // embedded K8s type
    Backup    *BackupSpec                  `json:"backup,omitempty"`  // optional nested (pointer)
}
```

## Optional vs Required Fields

| Pattern | When to use | Example |
|---------|-------------|---------|
| `Field Type` (value) | Always required, zero value is meaningful | `Replicas int32` |
| `Field *Type` (pointer) | Truly optional, nil means "not configured" | `Backup *BackupSpec` |
| `omitempty` tag | Omit from JSON when zero/nil | `json:"field,omitempty"` |
| `+kubebuilder:validation:Required` | Explicitly required by CRD validation | Required string fields |

**Rule of thumb**: Use pointers for optional nested structs and optional scalars where zero-value has a different meaning from "not set".

## Embedded Kubernetes Types

Use existing K8s types instead of reinventing:
```go
import corev1 "k8s.io/api/core/v1"

type Spec struct {
    Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}
```

This gives users the familiar `requests`/`limits` with `cpu`/`memory` without custom validation.

## Secret References

Pattern for referencing a Secret by name and key:
```go
type SecretKeyValue struct {
    Name string `json:"name"`  // Secret name
    Key  string `json:"key"`   // Key within the Secret
}

type Spec struct {
    PasswordSecret SecretKeyValue `json:"passwordSecret"`
}
```

## JSON Tag Conventions

- Use **camelCase** for JSON tags: `json:"storageClassName"` not `json:"storage_class_name"`
- Always include `json` tag on every exported field
- Use `omitempty` for optional fields
- Use `,inline` only for TypeMeta and ObjectMeta

## OneOf Patterns

Kubernetes CRDs don't natively support "exactly one of A or B". Implement in webhooks:
```go
type Spec struct {
    Postgres *PostgresConfig `json:"postgres,omitempty"`
    MySQL    *MySQLConfig    `json:"mysql,omitempty"`
}
// Webhook validates: exactly one of Postgres or MySQL must be set
```
