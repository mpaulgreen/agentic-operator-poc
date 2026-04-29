# RBAC Annotations

## Syntax

```go
//+kubebuilder:rbac:groups=<api-group>,resources=<plural>,verbs=<verb1>;<verb2>
```

Place directly above the `Reconcile()` function. `make manifests` generates ClusterRole from these.

## Standard Annotations for a Controller

```go
// Own CRD
//+kubebuilder:rbac:groups=cache.redis.example.com,resources=redisclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.redis.example.com,resources=redisclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.redis.example.com,resources=redisclusters/finalizers,verbs=update

// Managed resources (one per type)
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Read-only resources
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Events
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
```

## Rules

| Rule | Example |
|------|---------|
| One marker per resource type | Don't combine secrets and configmaps |
| Use `groups=core` or `groups=""` for core API | Not `groups=v1` |
| Add `/status` subresource | For your own CRD |
| Add `/finalizers` subresource | If using finalizers |
| Add events permission | `resources=events,verbs=create;patch` |
| No wildcard verbs | Never use `verbs=*` |
| Read-only for unmanaged resources | `verbs=get;list;watch` for Pods, PVCs you only read |

## Least Privilege

Only grant verbs you actually use:
- `get;list;watch` — for resources you read but don't create
- `get;list;watch;create;update;patch;delete` — for resources you fully manage
- `create;patch` — for events
- `update` — for finalizers
