# Red Hat Operator Certification Checklist

Requirements for publishing an operator on the Red Hat Ecosystem Catalog (certified operators).

## Bundle Requirements

| # | Requirement | How to Check |
|---|-------------|--------------|
| 1 | Valid bundle structure (manifests/, metadata/) | `validate-bundle-structure.sh` |
| 2 | CSV passes `operator-sdk bundle validate` | `operator-sdk bundle validate bundle/` |
| 3 | All scorecard tests pass | `operator-sdk scorecard bundle/` |
| 4 | alm-examples with valid sample CRs | `check-scorecard-readiness.py` |
| 5 | specDescriptors for all Spec fields | CSV `specDescriptors` section |
| 6 | statusDescriptors for all Status fields | CSV `statusDescriptors` section |

## CSV Requirements

| # | Requirement | Details |
|---|-------------|---------|
| 1 | displayName set | Human-readable operator name |
| 2 | description with overview | Markdown description with features |
| 3 | icon present (non-empty) | Base64-encoded SVG or PNG, min 120x120px |
| 4 | maintainers with valid email | At least one maintainer entry |
| 5 | links (documentation, source) | At least documentation link |
| 6 | categories from allowed list | Database, Monitoring, Networking, Security, etc. |
| 7 | capabilities level accurate | Must reflect actual operator features |
| 8 | minKubeVersion set | Minimum supported Kubernetes version |

## Security Requirements

| # | Requirement | How to Verify |
|---|-------------|---------------|
| 1 | Container runs as non-root | `securityContext.runAsNonRoot: true` in deployment |
| 2 | No privileged containers | No `privileged: true` in securityContext |
| 3 | Capabilities dropped | `capabilities.drop: [ALL]` on all containers |
| 4 | No host network/PID/IPC | hostNetwork, hostPID, hostIPC all false or absent |
| 5 | RBAC uses least privilege | No `*` in verbs, resources, or apiGroups |
| 6 | Read-only root filesystem | `readOnlyRootFilesystem: true` (recommended) |

## Image Requirements

| # | Requirement | Details |
|---|-------------|---------|
| 1 | Image from approved registry | quay.io, registry.redhat.io, or registry.connect.redhat.com |
| 2 | Image signed | Must pass Red Hat container certification |
| 3 | Base image is UBI | Based on Red Hat Universal Base Image |
| 4 | No latest tag | Must use specific version tags |

## Testing Requirements

| # | Requirement | Details |
|---|-------------|---------|
| 1 | Operator installs cleanly | Fresh install on supported OCP version |
| 2 | Operator upgrades cleanly | Upgrade from previous version |
| 3 | CR create/delete works | Managed resources created and cleaned up |
| 4 | No cluster-scoped side effects | Unless explicitly declared |

## Submission Process

1. Create Preflight scan: `preflight check operator <bundle-image>`
2. Submit via Red Hat Partner Connect portal
3. Automated testing pipeline runs
4. Manual review by Red Hat certification team
5. Published to certified catalog
