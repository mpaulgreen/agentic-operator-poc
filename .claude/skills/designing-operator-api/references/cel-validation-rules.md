# CEL Validation Rules for CRDs

CEL (Common Expression Language) enables declarative cross-field validation directly in CRD schemas (Kubernetes 1.25+). Use when kubebuilder markers can't express the rule.

## When to Use CEL vs Markers vs Webhooks

| Rule type | Use |
|-----------|-----|
| Single field range/enum/pattern | Kubebuilder markers |
| Cross-field validation | CEL |
| External lookups, complex logic | Webhooks |

## Syntax

```go
// +kubebuilder:validation:XValidation:rule="<expression>",message="<error message>"
```

Apply to a struct to validate relationships between its fields.

## Examples

### Ensure min <= max
```go
// +kubebuilder:validation:XValidation:rule="self.minReplicas <= self.maxReplicas",message="minReplicas must not exceed maxReplicas"
type AutoscalingSpec struct {
    MinReplicas int32 `json:"minReplicas"`
    MaxReplicas int32 `json:"maxReplicas"`
}
```

### Conditional required field
```go
// +kubebuilder:validation:XValidation:rule="!self.backupEnabled || has(self.backupSchedule)",message="backupSchedule is required when backupEnabled is true"
type Spec struct {
    BackupEnabled  bool    `json:"backupEnabled"`
    BackupSchedule *string `json:"backupSchedule,omitempty"`
}
```

### Immutable field (can't change after creation)
```go
// +kubebuilder:validation:XValidation:rule="self.storageClass == oldSelf.storageClass",message="storageClass is immutable"
type StorageSpec struct {
    StorageClass string `json:"storageClass"`
    Size         string `json:"size"`
}
```
Note: `oldSelf` is only available on update, not create.

### String format validation
```go
// +kubebuilder:validation:XValidation:rule="self.endpoint.startsWith('https://')",message="endpoint must use HTTPS"
type Spec struct {
    Endpoint string `json:"endpoint"`
}
```

## CEL Functions Available

| Function | Example |
|----------|---------|
| `has(field)` | `has(self.backup)` — field is set |
| `self.field` | Access current object fields |
| `oldSelf.field` | Access previous value (update only) |
| `size(list)` | `size(self.items) > 0` |
| `startsWith/endsWith` | `self.name.endsWith('-prod')` |
| `matches(regex)` | `self.version.matches('^v[0-9]+')` |
| Arithmetic | `self.min <= self.max` |
| Logical | `!self.enabled \|\| has(self.config)` |

## Limitations

- No external API calls or DNS lookups
- Expression cost limits (prevents expensive operations)
- String operations limited to ~100KB
- Not supported on scalar fields (only structs)
- `oldSelf` only available with `XValidation` on the type, not on individual fields
