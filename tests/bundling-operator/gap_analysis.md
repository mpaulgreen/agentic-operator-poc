# Sprint 5 Gap Analysis: `bundling-operator` Skill vs operator-sdk

## Bundle Structure

Both produce identical directory layout. The SDK generates via `make bundle` (kustomize + operator-sdk generate). The skill generates directly from operator project files.

| Aspect | SDK (`make bundle`) | Skill |
|--------|---------------------|-------|
| bundle/manifests/ | Generated | Generated |
| bundle/metadata/annotations.yaml | Generated | Generated |
| bundle/tests/scorecard/config.yaml | Generated | Generated |
| bundle.Dockerfile | Generated | Generated |
| CRD in manifests/ | From config/crd/bases/ | From config/crd/bases/ |
| Extra manifests (Service, ClusterRoles) | From kustomize overlays | Not generated (minimal bundle) |

## CSV Content

| Aspect | SDK | Skill | Match? |
|--------|-----|-------|--------|
| apiVersion/kind | v1alpha1/CSV | Same | MATCH |
| metadata.name pattern | `<pkg>.v<ver>` | Same | MATCH |
| alm-examples | Stub with `Foo` field | Rich with all Spec fields | SKILL BETTER |
| specDescriptors | Empty | Per Spec field | SKILL BETTER |
| statusDescriptors | Empty | Per Status field | SKILL BETTER |
| clusterPermissions | From controller RBAC | Same source, same output | MATCH |
| permissions (leader election) | Generated | Same pattern | MATCH |
| deployment template | Full with probes, security | Same pattern | MATCH |
| installModes | 4 modes | 4 modes | MATCH |
| icon | Empty | Empty (user provides) | MATCH |

## What the Skill Adds

1. **specDescriptors**: Automatically generated from types.go Spec fields — enables OpenShift console form rendering
2. **statusDescriptors**: Automatically generated from types.go Status fields — enables status display in console
3. **Rich alm-examples**: Sample CR populated with all Spec fields and sensible defaults, not a stub
4. **Validation scripts**: Pre-flight checks before running scorecard on a cluster
5. **Scorecard readiness**: Verifies descriptor paths match CRD schema

## Remaining Differences (acceptable)

| # | Difference | Notes |
|---|-----------|-------|
| 1 | SDK generates extra manifests (metrics Service, editor/viewer ClusterRoles) | These come from kustomize overlays, not the CSV itself. Skill focuses on the core bundle. |
| 2 | SDK runs `operator-sdk bundle validate` | Requires operator-sdk binary. Skill provides equivalent Python/Bash validators. |
| 3 | SDK icon is empty by default too | Both need user-provided icon for certification. |

## Summary

The skill produces the same bundle structure as `make bundle` with significantly richer CSV content (descriptors, alm-examples). The SDK generates a minimal bundle that requires manual enhancement for certification. The skill generates a certification-ready bundle from the start.
