# OLM Bundle Structure

An OLM bundle is an immutable packaging format for a single version of an operator.

## Directory Layout

```
<project-root>/
├── bundle/
│   ├── manifests/                    # Kubernetes manifests
│   │   ├── <operator>.clusterserviceversion.yaml  # Required: the CSV
│   │   ├── <group>_<plural>.yaml     # Required: CRD(s)
│   │   ├── *_v1_service.yaml         # Optional: metrics Service
│   │   └── *_clusterrole.yaml        # Optional: editor/viewer roles
│   ├── metadata/
│   │   └── annotations.yaml          # Required: package metadata
│   └── tests/
│       └── scorecard/
│           └── config.yaml            # Required: scorecard test config
└── bundle.Dockerfile                  # Required: bundle image (at project root)
```

## manifests/

Contains the CSV and all Kubernetes resources the operator needs at install time:
- **CSV** (exactly one): The main artifact
- **CRDs**: One per owned API type (from `config/crd/bases/`)
- **Services**: Metrics service if secure metrics enabled
- **ClusterRoles**: Editor and viewer roles for the CRD (from `config/rbac/`)

CRDs in the bundle must match the `spec.customresourcedefinitions.owned` entries in the CSV.

## metadata/annotations.yaml

Required keys:
```yaml
annotations:
  operators.operatorframework.io.bundle.mediatype.v1: registry+v1
  operators.operatorframework.io.bundle.manifests.v1: manifests/
  operators.operatorframework.io.bundle.metadata.v1: metadata/
  operators.operatorframework.io.bundle.package.v1: <operator-name>
  operators.operatorframework.io.bundle.channels.v1: <channel>
```

Optional keys:
```yaml
  operators.operatorframework.io.bundle.channel.default.v1: stable
  operators.operatorframework.io.metrics.builder: operator-sdk-v1.37.0
  operators.operatorframework.io.metrics.project_layout: go.kubebuilder.io/v4
  operators.operatorframework.io.test.mediatype.v1: scorecard+v1
  operators.operatorframework.io.test.config.v1: tests/scorecard/
```

## bundle.Dockerfile

Located at the project root (not inside bundle/). Uses `FROM scratch` and mirrors annotations as LABELs:

```dockerfile
FROM scratch
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
...
COPY bundle/manifests /manifests/
COPY bundle/metadata /metadata/
COPY bundle/tests/scorecard /tests/scorecard/
```

## Generation with operator-sdk

`make bundle` runs these steps:
1. `operator-sdk generate kustomize manifests` — generates CSV skeleton
2. `kustomize build config/manifests` — applies overlays
3. `operator-sdk bundle validate` — validates the result

Our skill generates the same output directly, with richer descriptors and alm-examples.
