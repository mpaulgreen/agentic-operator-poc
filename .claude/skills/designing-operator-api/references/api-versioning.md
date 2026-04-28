# API Versioning

## Version Progression

| Version | Stability | Breaking Changes | Example |
|---------|-----------|-----------------|---------|
| `v1alpha1` | Experimental | Expected | Initial development |
| `v1beta1` | Stabilizing | Possible but discouraged | Feature-complete, gathering feedback |
| `v1` | Stable | Not allowed | Production-ready |

## Storage Version

One version is the **storage version** — the format used to persist resources in etcd. Mark with:

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

type RedisCluster struct { ... }
```

When promoting versions:
1. Add `+kubebuilder:storageversion` to the new version's root type
2. Remove it from the old version
3. All existing resources are served at both versions but stored in the new format

## Creating a New Version

```bash
# Scaffold new version
operator-sdk create api --group cache --version v1beta1 --kind RedisCluster --resource --controller=false
```

This creates `api/v1beta1/` with types and groupversion_info. Use `--controller=false` since the controller already exists for v1alpha1.

## Directory Structure

```
api/
├── v1alpha1/
│   ├── groupversion_info.go
│   ├── rediscluster_types.go      # Old version (no storageversion marker)
│   └── zz_generated.deepcopy.go
└── v1beta1/
    ├── groupversion_info.go
    ├── rediscluster_types.go      # New version (+kubebuilder:storageversion)
    └── zz_generated.deepcopy.go
```

## Conversion Webhook (Hub-and-Spoke)

When field schemas differ between versions, a conversion webhook converts between them.

**Hub**: The canonical version (usually the storage version). All other versions convert to/from the hub.

```
v1alpha1 ←→ Hub (v1beta1) ←→ v1
```

This reduces N×N conversions to N×1. Each spoke only needs `ConvertTo(hub)` and `ConvertFrom(hub)`.

## When Conversion Is Needed

| Scenario | Conversion? |
|----------|-------------|
| Same fields, just version bump | No — `None` strategy works |
| Fields renamed | Yes — webhook maps old→new names |
| Fields added in new version | Yes — webhook sets defaults for old→new |
| Fields removed in new version | Yes — webhook drops fields for new→old |
| Field types changed | Yes — webhook transforms types |

## CRD Configuration

Without conversion (same schema):
```yaml
spec:
  conversion:
    strategy: None
```

With conversion webhook:
```yaml
spec:
  conversion:
    strategy: Webhook
    webhook:
      clientConfig:
        service:
          name: webhook-service
          path: /convert
```

## Key Considerations

- Only change storage version with a careful migration plan
- Run `kubectl get --all-namespaces <resource>` after changing storage version to trigger re-storage
- Keep old versions served for backward compatibility
- Test conversions bidirectionally (old→new and new→old)
