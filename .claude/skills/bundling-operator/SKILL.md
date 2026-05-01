---
name: bundling-operator
description: >
  Generates OLM (Operator Lifecycle Manager) bundles for operator distribution.
  Use when user asks to create an OLM bundle, generate a CSV, package an operator
  for OLM, prepare for certification, update a bundle version, add scorecard tests,
  or create a catalog source.
---

# Bundling Operator for OLM

Generate complete OLM bundles from an existing operator project. This skill reads the CRD, controller RBAC markers, and types to produce a ClusterServiceVersion (CSV), bundle metadata, scorecard config, and Dockerfile — ready for `operator-sdk bundle validate` and Red Hat certification.

## Template Variables

| Variable | Description | Example | Default |
|----------|-------------|---------|---------|
| OPERATOR_NAME | Operator package name | redis-operator | — |
| VERSION | Semantic version | 0.1.0 | — |
| DISPLAY_NAME | Human-readable name | Redis Operator | — |
| DESCRIPTION | Operator description | Manages Redis clusters... | — |
| GROUP | API group | cache.redis.example.com | — |
| KIND | CRD kind | RedisCluster | — |
| KIND_PLURAL | Plural form | redisclusters | — |
| API_VERSION | API version | v1alpha1 | — |
| CHANNEL | OLM channel | alpha | alpha |
| CATEGORY | OperatorHub category | Database | — |
| MATURITY | Maturity level | alpha | alpha |
| IMAGE | Operator container image | quay.io/example/redis-operator:v0.1.0 | — |
| SA_NAME | ServiceAccount name | <operator>-controller-manager | — |
| REPLACES | Previous CSV name (for upgrades) | redis-operator.v0.1.0 | (none) |

## Bundle Directory Layout

```
<project-root>/
├── bundle/
│   ├── manifests/
│   │   ├── <operator>.clusterserviceversion.yaml   # CSV (main artifact)
│   │   └── <group>_<plural>.yaml                   # CRD (from config/crd/bases/)
│   ├── metadata/
│   │   └── annotations.yaml                         # Package, channel, mediatype
│   └── tests/
│       └── scorecard/
│           └── config.yaml                           # Scorecard test configuration
└── bundle.Dockerfile                                 # Bundle image (at project root)
```

## Workflow A: Generate Initial OLM Bundle

Use when the user has a complete operator project and needs to package it for OLM distribution.

1. **Collect requirements** — the user provides:
   - Operator name, version, display name, description
   - Channel (alpha/beta/stable), category (Database, Monitoring, etc.), maturity
   - Install modes (OwnNamespace, SingleNamespace, AllNamespaces)
   - Container image registry URL
   - Icon (optional — base64-encoded SVG or PNG)

2. **Read the operator project** to extract:
   - **CRD** from `config/crd/bases/<group>_<plural>.yaml` — kind, group, version, spec/status fields
   - **RBAC markers** from controller `.go` files — `//+kubebuilder:rbac:groups=...` → map to CSV clusterPermissions
   - **Types** from `api/<version>/<kind>_types.go` — Spec fields → specDescriptors, Status fields → statusDescriptors
   - **Webhook config** from `config/webhook/` if present → webhookdefinitions

3. **Generate bundle files** using templates:
   - `bundle/manifests/<operator>.clusterserviceversion.yaml` — see `assets/templates/csv.yaml.tmpl`
   - `bundle/manifests/<group>_<plural>.yaml` — copy from `config/crd/bases/`
   - `bundle/metadata/annotations.yaml` — see `assets/templates/annotations.yaml.tmpl`
   - `bundle/tests/scorecard/config.yaml` — see `assets/templates/scorecard-config.yaml.tmpl`
   - `bundle.Dockerfile` — see `assets/templates/bundle.dockerfile.tmpl`

4. **Build alm-examples** — create a sample CR JSON from the CRD spec with sensible defaults:
   ```json
   [{"apiVersion": "<group>/<version>", "kind": "<Kind>", "metadata": {"name": "<kind>-sample"}, "spec": {...}}]
   ```

5. **Map RBAC markers to CSV permissions**:
   - Controller `//+kubebuilder:rbac` markers → `spec.install.spec.clusterPermissions[0].rules`
   - Always add namespace-scoped permissions for leader election: `configmaps`, `leases`, `events`
   - Add `tokenreviews` and `subjectaccessreviews` for secure metrics (if kube-rbac-proxy used)

6. **Build descriptors** from types.go:
   - Each exported Spec field → specDescriptor with `{path, displayName, description}`
   - Each exported Status field → statusDescriptor with `{path, displayName, description}`

7. **Verify** with `scripts/validate-bundle-structure.sh`, `scripts/validate-csv.py`, and `scripts/check-scorecard-readiness.py`.

## Workflow B: Update Bundle for New Version

Use when the user has an existing bundle and needs to update it for a new operator version.

1. **Read existing CSV** to get current version, RBAC, descriptors, and alm-examples.

2. **Update version fields**:
   - `metadata.name` → `<operator>.v<new-version>`
   - `spec.version` → `<new-version>`
   - `spec.replaces` → `<operator>.v<old-version>`

3. **Add new RBAC entries** for any new resources the controller now manages.

4. **Add new descriptors** for any new Spec/Status fields added to the CRD.

5. **Update alm-examples** to include new fields with sensible defaults.

6. **Preserve all unchanged sections** — do not remove existing RBAC rules, descriptors, or annotations.

7. **Verify** with all three validation scripts.

## RBAC Mapping: Controller Markers → CSV

| Controller Marker | CSV Location |
|-------------------|--------------|
| `//+kubebuilder:rbac:groups="",resources=secrets` | `spec.install.spec.clusterPermissions[0].rules` |
| `//+kubebuilder:rbac:groups=apps,resources=statefulsets` | `spec.install.spec.clusterPermissions[0].rules` |
| `//+kubebuilder:rbac:groups=<group>,resources=<plural>/status` | `spec.install.spec.clusterPermissions[0].rules` |
| `//+kubebuilder:rbac:groups=<group>,resources=<plural>/finalizers` | `spec.install.spec.clusterPermissions[0].rules` |
| Leader election (always added) | `spec.install.spec.permissions[0].rules` |

## Capability Levels

| Level | Description |
|-------|-------------|
| Basic Install | Operator installs and configures the operand |
| Seamless Upgrades | Operator handles version upgrades of itself and operand |
| Full Lifecycle | Operator manages backup, restore, scaling |
| Deep Insights | Operator exposes metrics, alerts, log analysis |
| Auto Pilot | Operator auto-tunes, auto-scales, auto-heals |

## Install Modes

| Type | Description | Default |
|------|-------------|---------|
| OwnNamespace | Watches only the namespace it's installed in | supported |
| SingleNamespace | Watches one user-specified namespace | supported |
| MultiNamespace | Watches multiple namespaces (rarely used) | not supported |
| AllNamespaces | Watches all namespaces (cluster-admin) | supported |
