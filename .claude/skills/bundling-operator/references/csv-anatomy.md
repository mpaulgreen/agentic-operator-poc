# ClusterServiceVersion Anatomy

The CSV is the primary artifact in an OLM bundle. It describes everything OLM needs to install, manage, and display the operator.

## Required Sections

### metadata.annotations
- `alm-examples`: JSON array of sample CRs (one per owned CRD). OLM uses this to pre-populate the "Create" form in the console.
- `capabilities`: One of: Basic Install, Seamless Upgrades, Full Lifecycle, Deep Insights, Auto Pilot
- `categories`: Comma-separated from the predefined list (Database, Monitoring, Networking, etc.)

### spec.customresourcedefinitions.owned
Each owned CRD needs: `kind`, `name` (plural.group), `version`, `displayName`, `description`.

Optional but recommended:
- `specDescriptors`: Array of `{path, displayName, description}` for Spec fields
- `statusDescriptors`: Array of `{path, displayName, description}` for Status fields

### spec.install
Two sub-sections:
- `clusterPermissions`: Cluster-scoped RBAC rules (for CRD access, cross-namespace resources)
- `permissions`: Namespace-scoped RBAC rules (leader election: configmaps, leases, events)
- `deployments`: Pod template for the operator controller manager

### spec.installModes
Four types: OwnNamespace, SingleNamespace, MultiNamespace, AllNamespaces. At least one must be supported.

### Metadata fields
- `displayName`, `description`, `icon`, `maturity`, `version`, `provider`, `maintainers`, `links`, `keywords`

## RBAC Mapping

Controller `//+kubebuilder:rbac` markers map directly to `spec.install.spec.clusterPermissions[0].rules`:

```
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
```
becomes:
```yaml
- apiGroups: [""]
  resources: [secrets]
  verbs: [get, list, watch, create, update, patch, delete]
```

Always add to clusterPermissions:
- `authentication.k8s.io/tokenreviews` (create) — for secure metrics
- `authorization.k8s.io/subjectaccessreviews` (create) — for secure metrics

Always add to namespace permissions:
- `configmaps` (CRUD) — leader election
- `coordination.k8s.io/leases` (CRUD) — leader election
- `events` (create, patch) — event recording

## Descriptors

specDescriptors and statusDescriptors provide UI hints for the OpenShift console:

```yaml
specDescriptors:
- description: Number of Redis replicas
  displayName: Replicas
  path: replicas
- description: Storage configuration
  displayName: Storage
  path: storage
statusDescriptors:
- description: Current operational phase
  displayName: Phase
  path: phase
- description: Conditions for the resource
  displayName: Conditions
  path: conditions
```

The `path` field uses dot notation relative to spec/status (e.g., `storage.size`, `backup.schedule`).

## Version Upgrades

For version bumps:
- `metadata.name`: `<operator>.v<new-version>`
- `spec.version`: `<new-version>`
- `spec.replaces`: `<operator>.v<old-version>` (creates upgrade graph)
- `spec.skips`: List of versions to skip (for channel pruning)
