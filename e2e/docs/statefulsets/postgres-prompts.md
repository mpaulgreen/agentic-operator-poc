# PostgreSQL Operator — E2E Scenario Prompts

Progressive enhancement of a PostgreSQL operator across 4 scenarios, each exercising a different `designing-operator-api` workflow.

**Operator location**: `e2e/postgres-operator/`
**Validation guide**: `e2e/docs/statefulsets/postgres-e2e-validation.md`
**Status**: All 4 scenarios EXECUTED and validated on OpenShift — 111/111 test conditions pass.

| Scenario | Feature | Bundle | Workflow | New Resource | Tests |
|----------|---------|--------|----------|-------------|-------|
| A | Core operator (from scratch) | v0.1.0 | A (new) | StatefulSet, Service, Secret, ConfigMap, CronJob | 31 |
| B | High Availability | v0.2.0 | B (modify) | PodDisruptionBudget (policy/v1) | 25 |
| C | Webhooks + Network Security | v0.3.0 | C (webhooks) | NetworkPolicy (networking.k8s.io/v1) | 27 |
| D | API Maturity + Connection Pooling | v0.4.0 | D (versioning) | Deployment (apps/v1) | 28 |

---

## Scenario A: New Operator from Scratch

```
Prompt: "Build me a complete OpenShift operator that manages PostgreSQL 
clusters. Requirements:

Spec:
- replicas: 1-5, default 3
- version: enum 14/15/16, default 16
- storage: size (string), storageClassName (string)
- resources: cpu/memory requests and limits
- backup: enabled (bool), schedule (cron string), retentionDays (1-30)

Status:
- phase: Pending/Initializing/Running/Failed/Degraded
- readyReplicas, currentVersion, endpoint
- conditions: Available, Progressing, Degraded, BackupReady

Controller should reconcile:
- Secret (superuser credentials)
- ConfigMap (postgresql.conf)
- Service (headless, port 5432)
- StatefulSet (postgres containers with PVCs)
- CronJob (if backup.enabled, pg_dump on schedule)

Generate the complete project, tests, and OLM bundle v0.1.0 on alpha channel.
Review the code for best practices before finalizing."
```

**Acceptance criteria** (EXECUTED — all pass):
- [x] Complete project structure valid (49/49 scaffold checks)
- [x] Types compile with all markers (14/14 type checks)
- [x] Controller compiles with 5 reconciler methods (9 RBAC, 5/5 idempotent)
- [x] Tests compile and cover all methods (5/5, 16 test cases)
- [x] Bundle valid with matching descriptors (3/3 scripts, 0 warnings)
- [x] Code review shows no Critical issues (0 Critical, 0 Warning)
- [x] Total files created: 53 (including full config/ kustomize structure)
- [x] Deployed to OpenShift: 31/31 test conditions pass (both make deploy and OLM paths)

---

## Scenario B: Add High Availability (designing-operator-api Workflow B)

Builds on the postgres-operator from Scenario A. Adds PodDisruptionBudget (policy/v1) + pod anti-affinity.

```
Prompt: "Add High Availability support to the existing PostgreSQL operator at 
e2e/postgres-operator/.

1. Add to the CRD: HASpec with minAvailable (*int32, min 1), 
   maxUnavailable (*int32, min 1, mutually exclusive with minAvailable),
   antiAffinityMode (string enum preferred/required, default preferred)
2. Add to Status: HAReady condition
3. Add reconcilePodDisruptionBudget() — creates PDB from policy/v1 
   when spec.ha is non-nil, uses minAvailable or maxUnavailable from spec,
   defaults to minAvailable=replicas-1 when neither is set
4. Update reconcileStatefulSet() to add pod anti-affinity based on 
   antiAffinityMode when spec.ha is non-nil
5. Generate tests for the new reconciler method
6. Update the OLM bundle from v0.1.0 to v0.2.0 with replaces set correctly"
```

**Acceptance criteria** (EXECUTED — all pass):
- [x] HASpec struct with minAvailable, maxUnavailable, antiAffinityMode fields and markers
- [x] reconcilePodDisruptionBudget() follows check-create idempotency pattern
- [x] PDB only created when spec.ha is non-nil
- [x] Anti-affinity added to StatefulSet PodTemplateSpec
- [x] HAReady condition helpers added
- [x] RBAC for policy/poddisruptionbudgets, Owns PDB
- [x] ~5 PDB test cases, all existing tests pass
- [x] CSV v0.2.0 with replaces v0.1.0, PDB RBAC + descriptors
- [x] Code review: 0 Critical, bundle validates
- [x] Deployed to OpenShift: 25/25 test conditions pass (both make deploy and OLM paths)

---

## Scenario C: Webhooks + Network Security (designing-operator-api Workflow C)

Builds on Scenario B (v0.2.0). Adds defaulting/validating webhooks + NetworkPolicy (networking.k8s.io/v1).

**Prerequisite**: cert-manager operator on OpenShift.

```
Prompt: "Add admission webhooks and network security to the PostgreSQL operator at
e2e/postgres-operator/ (which already has HA support at v0.2.0).

1. Add webhooks (designing-operator-api Workflow C):
   Defaulting: set replicas=3 when 0, version=16 when empty,
   antiAffinityMode=preferred when ha is nil,
   minAvailable=replicas-1 when ha set but neither field specified
   Validating: reject minAvailable >= replicas, reject both 
   minAvailable and maxUnavailable set, reject backup.enabled 
   without schedule, reject storage size reduction on update
2. Generate all webhook config files (service, cert-manager, patches)
3. Update main.go and kustomization files
4. Add reconcileNetworkPolicy() — creates NetworkPolicy from 
   networking.k8s.io/v1, allows port 5432 ingress from same namespace,
   allows DNS egress, always created (security baseline)
5. Add NetworkSecured condition
6. Generate tests for NetworkPolicy + webhook validation
7. Update OLM bundle from v0.2.0 to v0.3.0 with webhook definitions"
```

**Acceptance criteria** (EXECUTED — all pass):
- [x] Webhook handler with Default() + ValidateCreate/Update/Delete()
- [x] 9 webhook config files + kustomization updates
- [x] reconcileNetworkPolicy() follows check-create pattern
- [x] NetworkPolicy allows port 5432 ingress, DNS egress
- [x] ~9 webhook + ~2 NP test cases, all existing tests pass
- [x] CSV v0.3.0 with replaces v0.2.0, webhookdefinitions, NP RBAC
- [x] Code review: 0 Critical, bundle validates
- [x] Deployed to OpenShift: 27/27 test conditions pass (both make deploy and OLM paths)

---

## Scenario D: API Maturity + Connection Pooling (designing-operator-api Workflow D)

Builds on Scenario C (v0.3.0). Promotes API to v1beta1 + adds PgBouncer Deployment (apps/v1).

```
Prompt: "Promote the PostgreSQL operator API to v1beta1 and add connection pooling
at e2e/postgres-operator/ (which already has HA + webhooks at v0.3.0).

1. Add API version v1beta1 (designing-operator-api Workflow D):
   Copy types to api/v1beta1/, add +kubebuilder:storageversion,
   add new fields: maxMemory (*resource.Quantity), 
   connectionPool (*ConnectionPoolSpec)
   ConnectionPoolSpec: enabled (bool), poolSize (int32, 1-100, default 10),
   maxClientConnections (int32, 1-1000, default 100),
   idleTimeout (string, default 30s)
   Add to status: poolerReady (bool), poolerEndpoint (string),
   ConnectionPoolReady condition
2. Update main.go for v1beta1 scheme + webhook registration
3. Add reconcileConnectionPool() — creates Deployment (apps/v1) for 
   PgBouncer + ClusterIP Service on port 6432 when enabled,
   deletes both when disabled
4. Generate tests for connection pool reconciler + v1beta1 webhook
5. Update OLM bundle from v0.3.0 to v0.4.0 with multi-version CRD,
   maturity alpha→beta"
```

**Acceptance criteria** (EXECUTED — all pass):
- [x] api/v1beta1/ directory with groupversion_info.go, types.go, deepcopy, webhook
- [x] v1beta1 has +kubebuilder:storageversion, v1alpha1 does not
- [x] ConnectionPoolSpec with enabled, poolSize, maxClientConnections, idleTimeout
- [x] reconcileConnectionPool() creates Deployment + Service when enabled, deletes when disabled
- [x] ~8 connection pool + webhook test cases, all existing tests pass
- [x] CSV v0.4.0 with replaces v0.3.0, multi-version CRD, maturity=beta
- [x] Code review: 0 Critical, bundle validates
- [x] Deployed to OpenShift: 28/28 test conditions pass (both make deploy and OLM paths)

---

## Skill Coverage Matrix

| Scenario | designing-api | implementing-reconciliation | bundling | test-generator | reviewer | bundle-validator |
|----------|--------------|---------------------------|----------|----------------|----------|-----------------|
| A | Workflow A | Workflow A | Workflow A | Full | Full | Full |
| B | Workflow B | Workflow B | Workflow B | Workflow B | Full | Full |
| C | Workflow C | Workflow B | Workflow B | Workflow B | Full | Full |
| D | Workflow D | Workflow B | Workflow B | Workflow B | Full | Full |

## Version Chain

| Scenario | Bundle | Replaces | Maturity | API Versions |
|----------|--------|----------|----------|-------------|
| A | 0.1.0 | — | alpha | v1alpha1 |
| B | 0.2.0 | 0.1.0 | alpha | v1alpha1 |
| C | 0.3.0 | 0.2.0 | alpha | v1alpha1 |
| D | 0.4.0 | 0.3.0 | beta | v1alpha1 + v1beta1 (storage) |

## Skill Bugs Found (17 total)

| # | Bug | Scenario | Skill Fixed |
|---|-----|----------|-------------|
| 1-10 | See memory/feedback_e2e_skill_bugs.md | A | scaffolding, bundling, implementing-reconciliation |
| 11 | Check-update missing for modified reconcilers | B | implementing-reconciliation, operator-reviewer |
| 12 | Bundle CRD not refreshed on version update | B | bundling-operator |
| 13 | Check-update scope: all mutable fields, not just changed | C | implementing-reconciliation |
| 14 | Cert-manager kustomize variable substitution | C | designing-operator-api |
| 15 | CRD conversion webhook patch with strategy: None | D | designing-operator-api |
| 16 | Multi-version webhook strips new fields + CSV entries | D | designing-operator-api |
| 17 | Third-party image env var assumptions | D | implementing-reconciliation |
