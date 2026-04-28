# Kubebuilder Validation Markers Reference

## Numeric Constraints

```go
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=10
// +kubebuilder:default=3
Replicas int32 `json:"replicas,omitempty"`

// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=65535
// +kubebuilder:default=8080
Port *int32 `json:"port,omitempty"`

// +kubebuilder:validation:ExclusiveMinimum=true
// +kubebuilder:validation:Minimum=0
PositiveValue float64 `json:"positiveValue"`
```

## String Constraints

```go
// Enum — values separated by semicolons (not commas)
// +kubebuilder:validation:Enum="14";"15";"16"
// +kubebuilder:default="16"
Version string `json:"version,omitempty"`

// Pattern — use backticks for regex
// +kubebuilder:validation:Pattern=`^[0-9]+[KMGT]i$`
Size string `json:"size"`

// Length constraints
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=253
Name string `json:"name"`

// Format (validates common formats)
// +kubebuilder:validation:Format=uri
Endpoint string `json:"endpoint,omitempty"`
```

## Boolean and Default Values

```go
// +kubebuilder:default=true
EnableMetrics bool `json:"enableMetrics,omitempty"`

// +kubebuilder:default="latest"
Tag string `json:"tag,omitempty"`

// +kubebuilder:default=3
Replicas int32 `json:"replicas,omitempty"`
```

## Required Fields

```go
// +kubebuilder:validation:Required
Host string `json:"host"`
```

Note: Fields without `omitempty` and without a pointer type are implicitly required at the Go level but not at the CRD level. Use the marker for CRD-level enforcement.

## Array/Slice Constraints

```go
// +kubebuilder:validation:MinItems=1
// +kubebuilder:validation:MaxItems=10
Targets []string `json:"targets"`
```

## Object-Level Markers (on the root type)

```go
// +kubebuilder:object:root=true          — marks as CRD root type
// +kubebuilder:subresource:status        — enables /status subresource
// +kubebuilder:resource:scope=Cluster    — cluster-scoped (default: Namespaced)
// +kubebuilder:resource:shortName=rc     — kubectl short name
// +kubebuilder:resource:singular=redis   — override singular name
```

## Print Columns (on the root type)

```go
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].status`,priority=1
```

Types: `string`, `integer`, `number`, `boolean`, `date`. Use `priority=1` to hide from default `kubectl get` (shown with `-o wide`).

## Common Gotchas

| Mistake | Fix |
|---------|-----|
| `Enum="a","b","c"` | `Enum="a";"b";"c"` (semicolons, not commas) |
| `Pattern="^[0-9]+$"` | `Pattern=` `` `^[0-9]+$` `` (backticks for regex) |
| Missing json tag | Every exported field needs `json:"fieldName"` |
| `+kubebuilder:default=nil` | Use `*Type` with `omitempty` instead |
| Marker on wrong line | Markers must be directly above the field, no blank line |
