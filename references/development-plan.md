# Development Plan: Build & Test OpenShift Operator Skills

## Context

We have a validated architecture of 5 skills + 3 subagents for building OpenShift operators with Claude Agentic Skills (see `architecture.md`). This plan defines the progressive build order, testing methodology at each stage, and sample prompts that exercise each skill across the four composition scenarios.

## Testing Methodology

Testing a skill means invoking it with a prompt and verifying the output. Three layers:

| Test Layer | What It Validates | When Run |
|-----------|-------------------|----------|
| **Unit Test** | Skill in isolation produces correct artifacts | After each skill is built |
| **Integration Test** | Skills compose correctly — output of one is valid input for the next | After each skill pair/chain is ready |
| **E2E Scenario Test** | Full scenario (A/B/C/D) produces a working operator | After all skills + subagents are ready |

Verification methods for each layer:
- **Structural**: Are the right files produced in the right locations?
- **Compilable**: Does the generated Go code compile? (`go build ./...`)
- **Pattern-correct**: Does the code follow idempotency patterns? (validation scripts)
- **Testable**: Do generated tests run? (`make test`)
- **Bundleable**: Does the OLM bundle pass validation? (validation scripts)

---

## Build Order & Dependencies

```
Sprint 1: scaffolding-operator         (no dependencies)
Sprint 2: designing-operator-api       (references scaffolding output)
Sprint 3: implementing-reconciliation  (references API types)
Sprint 4: testing-operator             (references controller patterns)
Sprint 5: bundling-operator            (references API + controller)
Sprint 6: operator-reviewer            (subagent, uses skills 2+3)
Sprint 7: operator-test-generator      (subagent, uses skill 4)
Sprint 8: operator-bundle-validator    (subagent, uses skill 5)
```

Each sprint follows: **Build skill → Unit test → Integration test with prior skills**

---

## Sprint 1: `scaffolding-operator`

### Patterns Covered

| Pattern | Workflow | Description |
|---------|----------|-------------|
| A | New project | Scaffold complete operator project from scratch |
| B | Same-group kind | Add kind to existing API group (flat layout) |
| C | Different-group kind | Add kind in new API group (multi-group layout, aliased imports) |
| D | Cluster-scoped resource | Scaffold with `namespaced: false` and `scope=Cluster` marker |

### Build

27 files in `.claude/skills/scaffolding-operator/` (1 SKILL.md, 3 references, 1 script with 49 checks, 22 templates). Validated against `operator-sdk` v1.37.0. Updated: removed kube-rbac-proxy sidecar templates (deprecated in operator-sdk v1.33+), added metrics-service and metrics-reader-clusterrole. Dockerfile uses `FROM --platform=$BUILDPLATFORM` for cross-compilation on Apple Silicon.

See `tests/scaffolding-operator/test_guide.md` for full test prompts, verification commands, and acceptance criteria across all 4 patterns.

See `tests/scaffolding-operator/gap_analysis.md` for detailed comparison against `operator-sdk` output.

---

## Sprint 2: `designing-operator-api`

### Patterns Covered

| Pattern | Description | Workflow |
|---------|-------------|----------|
| E | Resource-only (no controller) — documented in references, scaffolded by Sprint 1 | Ref only |
| F | Controller-only for external types — documented in references, scaffolded by Sprint 1 | Ref only |
| G | Multiple API versions (v1alpha1 → v1beta1) with `+kubebuilder:storageversion` | Workflow D |
| H | Validating/defaulting webhooks — Default(), ValidateCreate/Update/Delete() | Workflow C |
| I | Conversion webhooks — hub-and-spoke pattern | Workflow C (with conversion) |

### Build

24 files in `.claude/skills/designing-operator-api/` (1 SKILL.md, 7 references, 1 script with 14 checks, 11 templates, 4 examples). Validated against `operator-sdk`.

See `tests/designing-operator-api/test_guide.md` for full test prompts, verification commands, and acceptance criteria across all workflows.

See `tests/designing-operator-api/gap_analysis.md` for detailed comparison against `operator-sdk` output.

---

## Sprint 3: `implementing-reconciliation`

### Build

19 files in `.claude/skills/implementing-reconciliation/` (1 SKILL.md, 7 references, 2 scripts, 6 templates, 3 examples). Scripts validated against real database-operator (10 RBAC markers, all idempotency checks pass).

See `tests/implementing-reconciliation/test_guide.md` for full test prompts, verification commands, and acceptance criteria.

See `tests/implementing-reconciliation/gap_analysis.md` for detailed comparison against `operator-sdk` output.

### Unit Test

**Test 3.1 — Simple controller (3 resource types)**
```
Prompt: "Using the implementing-reconciliation skill, implement a controller 
for RedisCluster that reconciles these resources in order:
1. Secret (redis-credentials with generated password)
2. Service (headless service on port 6379)
3. StatefulSet (redis containers with spec.version image, spec.replicas count, 
   volume mounts for persistent storage)

Use the check-create idempotency pattern for each resource. Add finalizer 
for cleanup. Set owner references on all created resources. Update status 
conditions (Available, Progressing, Degraded) and status.readyReplicas 
from StatefulSet.

Generate these files:
- internal/controller/rediscluster_controller.go
- internal/controller/rediscluster_reconcilers.go  
- internal/controller/rediscluster_status.go
- internal/controller/rediscluster_conditions.go
- internal/controller/rediscluster_helpers.go"
```

Expected output:
- Controller with Reconcile() following three-phase pattern (fetch → orchestrate → status)
- Each reconcileX method using check-create: `Get → IsNotFound? → Create → SetOwnerRef → RecordEvent`
- Finalizer add on create, cleanup + remove on delete
- RBAC markers for all managed resource types
- Status updater reading StatefulSet readyReplicas

Verification:
```bash
# Idempotency check
python3 .claude/skills/implementing-reconciliation/scripts/check-idempotency.py \
  /tmp/redis-operator-test/internal/controller/rediscluster_reconcilers.go

# RBAC check
python3 .claude/skills/implementing-reconciliation/scripts/validate-rbac-annotations.py \
  /tmp/redis-operator-test/internal/controller/rediscluster_controller.go

# Compilation
cd /tmp/redis-operator-test && go build ./internal/controller/...
```

Acceptance criteria:
- [ ] All reconciler methods follow check-create pattern
- [ ] Owner references set on every created resource
- [ ] Finalizer implemented (add + cleanup + remove)
- [ ] RBAC annotations cover all managed resource types
- [ ] Events recorded for create and error
- [ ] Status conditions updated with proper types/reasons
- [ ] check-idempotency.py passes (no event-type-dependent logic)
- [ ] validate-rbac-annotations.py passes (no over/under-granting)
- [ ] Code compiles

**Test 3.2 — Add new resource to existing controller**
```
Prompt: "Using the implementing-reconciliation skill, add a new reconcileConfigMap() 
method to the existing RedisCluster controller. The ConfigMap should contain 
redis.conf with settings: maxmemory 256mb, maxmemory-policy allkeys-lru, 
timeout 300, tcp-keepalive 60. Follow the same check-create pattern as existing 
reconciler methods. Add the appropriate RBAC marker and call it in the correct 
dependency order (after Secret, before StatefulSet)."
```

Acceptance criteria:
- [ ] New reconcileConfigMap() follows same pattern as existing methods
- [ ] RBAC marker added for configmaps
- [ ] Called in correct position in Reconcile() chain
- [ ] ConfigMap data contains redis.conf settings
- [ ] Owner reference set
- [ ] Code compiles

### Integration Test (Sprint 1 + 2 + 3)

**Test I-1.2.3 — Scaffold → Design → Implement**
```
Prompt: "Build a complete operator from scratch:
1. Scaffold a project called 'cache-operator' with domain 'cache.example.com'
2. Design a CRD for CacheCluster with spec: engine (redis/memcached), 
   replicas (1-5), maxMemory string, evictionPolicy string. 
   Status: phase, readyReplicas, conditions.
3. Implement the controller that reconciles: Secret, ConfigMap, Service, 
   Deployment (not StatefulSet since cache is ephemeral). Use check-create 
   pattern, finalizers, owner refs, status updates."
```

Verification:
```bash
# Full chain
bash .claude/skills/scaffolding-operator/scripts/validate-project-structure.sh /tmp/cache-operator-test/
python3 .claude/skills/designing-operator-api/scripts/validate-api-types.py /tmp/cache-operator-test/api/v1alpha1/cachecluster_types.go
python3 .claude/skills/implementing-reconciliation/scripts/check-idempotency.py /tmp/cache-operator-test/internal/controller/cachecluster_reconcilers.go
cd /tmp/cache-operator-test && go mod tidy && go build ./...
```

Acceptance criteria:
- [ ] All three validation scripts pass
- [ ] Complete project compiles
- [ ] Controller references types from API package correctly
- [ ] RBAC markers match resources actually created in reconcilers

---

## Sprint 4: `testing-operator`

### Build

12 files in `.claude/skills/testing-operator/` (1 SKILL.md, 4 references, 2 scripts, 4 templates, 1 example). Scripts validated: check-test-coverage finds test files/cases, generate-test-matrix verifies 100% method coverage.

See `tests/testing-operator/test_guide.md` for full test prompts, verification commands, and acceptance criteria.

See `tests/testing-operator/gap_analysis.md` for detailed comparison against `operator-sdk` test output.

---

## Sprint 5: `bundling-operator`

### Build

15 files in `.claude/skills/bundling-operator/` (1 SKILL.md, 6 references, 3 scripts, 4 templates, 1 example). Scripts validated against database-operator bundle (0 errors each).

See `tests/bundling-operator/test_guide.md` for full test prompts, verification commands, and acceptance criteria.

See `tests/bundling-operator/gap_analysis.md` for detailed comparison against `make bundle` output.

### Unit Test

**Test 5.1 — Generate initial bundle**
```
Prompt: see tests/bundling-operator/test_guide.md Test 5.1
```

**Test 5.2 — Update bundle for new version**
```
Prompt: see tests/bundling-operator/test_guide.md Test 5.2
```

### Integration Test (All 5 Skills)

**Test I-1.2.3.4.5 — Full skill chain**
```
Prompt: "Build a complete operator end-to-end:
1. Scaffold 'metrics-collector-operator' with domain 'observability.example.com'
2. Design CRD for MetricsCollector: spec has scrapeInterval (string), 
   targets ([]string), retentionDays (1-90), storageSize (string). 
   Status: phase, targetsDiscovered (int), lastScrape (timestamp), conditions.
3. Implement controller reconciling: ConfigMap (scrape config), 
   Service (metrics endpoint), Deployment (collector), ServiceMonitor (Prometheus)
4. Generate test suite
5. Create OLM bundle v0.1.0 with channel 'stable', category 'Monitoring'"
```

Verification:
```bash
# All validations
bash .claude/skills/scaffolding-operator/scripts/validate-project-structure.sh /tmp/metrics-operator-test/
python3 .claude/skills/designing-operator-api/scripts/validate-api-types.py /tmp/metrics-operator-test/api/v1alpha1/metricscollector_types.go
python3 .claude/skills/implementing-reconciliation/scripts/check-idempotency.py /tmp/metrics-operator-test/internal/controller/metricscollector_reconcilers.go
python3 .claude/skills/bundling-operator/scripts/validate-csv.py /tmp/metrics-operator-test/bundle/manifests/metrics-collector-operator.clusterserviceversion.yaml
cd /tmp/metrics-operator-test && go mod tidy && go build ./... && make test
```

Acceptance criteria:
- [ ] All 5 validation scripts pass
- [ ] Project compiles end-to-end
- [ ] Tests compile and run
- [ ] Bundle is structurally valid
- [ ] CSV descriptors match CRD fields
- [ ] RBAC in CSV matches controller annotations

---

## Sprint 6: `operator-reviewer` (Subagent)

### Build

1 agent definition at `.claude/agents/operator-reviewer.md`. Composes skills 2+3 (designing-operator-api + implementing-reconciliation). Runs 3 validation scripts + manual checklist. Produces structured review with severity, line numbers, and fix suggestions.

See `tests/operator-reviewer/test_guide.md` for full test prompts (including flawed code to plant), verification commands, and acceptance criteria.

See `tests/operator-reviewer/gap_analysis.md` for comparison of automated vs manual review coverage.

---

## Sprint 7: `operator-test-generator` (Subagent)

### Build

1 agent definition at `.claude/agents/operator-test-generator.md`. Uses skill 4 (testing-operator). Discovers reconciler methods, generates suite_test.go + controller_test.go, validates with go vet + test matrix.

See `tests/operator-test-generator/test_guide.md` for full test prompts, verification commands, and acceptance criteria.

See `tests/operator-test-generator/gap_analysis.md` for comparison against manual test writing and operator-sdk.

---

## Sprint 8: `operator-bundle-validator` (Subagent)

### Build

1 agent definition at `.claude/agents/operator-bundle-validator.md`. Uses skill 5 (bundling-operator). Runs 3 validation scripts + certification checklist inspection.

See `tests/operator-bundle-validator/test_guide.md` for full test prompts (including 4 issues to plant), verification commands, and acceptance criteria.

See `tests/operator-bundle-validator/gap_analysis.md` for comparison against `operator-sdk bundle validate`.

---

## E2E Scenario Tests (After All Sprints)

These test complete workflows across all skills and subagents. Scenario A has been executed and deployed to a live OpenShift cluster.

> **Mandatory workflow (from CLAUDE.md):** E2E tests MUST follow the skill/subagent workflow: Skills for generation (Steps 1-3, 5), Subagents for verification (Steps 4a, 4b, 6). Do NOT write operator code from training knowledge.

> **Lessons learned from Scenario A on OpenShift:**
> 1. **kube-rbac-proxy image defunct** — removed sidecar, use controller-runtime built-in `filters.WithAuthenticationAndAuthorization`
> 2. **Dockerfile QEMU crash on Apple Silicon** — use `FROM --platform=$BUILDPLATFORM` for native cross-compilation
> 3. **Upstream container images crash on OpenShift** — use `registry.redhat.io/rhel9/postgresql-16` (not `postgres:16`), env vars use `POSTGRESQL_*` prefix, data dir `/var/lib/pgsql/data`
> 4. All three issues resulted in skill template updates so future operators don't hit them.
>
> See `e2e/openshift-e2e-validation.md` for the full OpenShift cluster validation guide (31 test conditions across 12 phases).

### Scenario A: New Operator from Scratch

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

Verification — run ALL validation scripts, compile, test, bundle validate.

Acceptance criteria (Scenario A EXECUTED — all pass):
- [x] Complete project structure valid (49/49 scaffold checks)
- [x] Types compile with all markers (14/14 type checks)
- [x] Controller compiles with 5 reconciler methods (9 RBAC, 5/5 idempotent)
- [x] Tests compile and cover all methods (5/5, 16 test cases)
- [x] Bundle valid with matching descriptors (3/3 scripts, 0 warnings)
- [x] Code review shows no Critical issues (0 Critical, 0 Warning)
- [x] Total files created: 53 (including full config/ kustomize structure)
- [x] Deployed to OpenShift: 3/3 PostgreSQL pods Running, Phase=Running

Output at `e2e/postgres-operator/`. OpenShift validation guide at `e2e/openshift-e2e-validation.md`.

---

### Scenario B: Add High Availability (designing-operator-api Workflow B)

Builds on the postgres-operator from Scenario A at `e2e/postgres-operator/`. Adds PodDisruptionBudget (policy/v1) + pod anti-affinity. Tests all 4 non-scaffolding skills in Workflow B + all 3 subagents.

```
Prompt: "Add High Availability support to the existing PostgreSQL operator at e2e/postgres-operator/.

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

Acceptance criteria:
- [ ] HASpec struct with minAvailable, maxUnavailable, antiAffinityMode fields and markers
- [ ] reconcilePodDisruptionBudget() follows check-create idempotency pattern
- [ ] PDB only created when spec.ha is non-nil
- [ ] Anti-affinity added to StatefulSet PodTemplateSpec
- [ ] HAReady condition helpers added
- [ ] RBAC for policy/poddisruptionbudgets, Owns PDB
- [ ] ~5 PDB test cases, all existing tests pass
- [ ] CSV v0.2.0 with replaces v0.1.0, PDB RBAC + descriptors
- [ ] Code review: 0 Critical, bundle validates

---

### Scenario C: Webhooks + Network Security (designing-operator-api Workflow C)

Builds on Scenario B (v0.2.0). Adds defaulting/validating webhooks + NetworkPolicy (networking.k8s.io/v1). Tests designing-api Workflow C + implementing-reconciliation Workflow B + all 3 subagents.

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

Acceptance criteria:
- [ ] Webhook handler with Default() + ValidateCreate/Update/Delete()
- [ ] 9 webhook config files + kustomization updates
- [ ] reconcileNetworkPolicy() follows check-create pattern
- [ ] NetworkPolicy allows port 5432 ingress, DNS egress
- [ ] ~9 webhook + ~2 NP test cases, all existing tests pass
- [ ] CSV v0.3.0 with replaces v0.2.0, webhookdefinitions, NP RBAC
- [ ] Code review: 0 Critical, bundle validates

---

### Scenario D: API Maturity + Connection Pooling (designing-operator-api Workflow D)

Builds on Scenario C (v0.3.0). Promotes API to v1beta1 + adds PgBouncer Deployment (apps/v1). Tests designing-api Workflow D + implementing-reconciliation Workflow B + all 3 subagents.

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

Acceptance criteria:
- [ ] api/v1beta1/ directory with groupversion_info.go, types.go, deepcopy, webhook
- [ ] v1beta1 has +kubebuilder:storageversion, v1alpha1 does not
- [ ] ConnectionPoolSpec with enabled, poolSize, maxClientConnections, idleTimeout
- [ ] reconcileConnectionPool() creates Deployment + Service when enabled, deletes when disabled
- [ ] ~8 connection pool + webhook test cases, all existing tests pass
- [ ] CSV v0.4.0 with replaces v0.3.0, multi-version CRD, maturity=beta
- [ ] Code review: 0 Critical, bundle validates

---

## Coverage Matrix

| Scenario | designing-api | implementing-reconciliation | bundling | test-generator | reviewer | bundle-validator | New Resource |
|----------|--------------|---------------------------|----------|----------------|----------|-----------------|-------------|
| A (done) | Workflow A | Workflow A | Workflow A | Full | Full | Full | StatefulSet, Service, Secret, ConfigMap, CronJob |
| B | Workflow B | Workflow B | Workflow B | Workflow B | Full | Full | PodDisruptionBudget (policy/v1) |
| C | Workflow C | Workflow B | Workflow B | Workflow B | Full | Full | NetworkPolicy (networking.k8s.io/v1) |
| D | Workflow D | Workflow B | Workflow B | Workflow B | Full | Full | Deployment (apps/v1) |

## Version Chain

| Scenario | Bundle | Replaces | Maturity | API Versions |
|----------|--------|----------|----------|-------------|
| A | 0.1.0 | — | alpha | v1alpha1 |
| B | 0.2.0 | 0.1.0 | alpha | v1alpha1 |
| C | 0.3.0 | 0.2.0 | alpha | v1alpha1 |
| D | 0.4.0 | 0.3.0 | beta | v1alpha1 + v1beta1 (storage) |

---

## Summary: Test Matrix

| Sprint | Component | Unit Tests | Integration Tests | Scenario Coverage |
|--------|-----------|-----------|-------------------|-------------------|
| 1 | scaffolding-operator | 1.1, 1.2 | — | A |
| 2 | designing-operator-api | 2.1, 2.2 | I-1.2 | A, B, C, D |
| 3 | implementing-reconciliation | 3.1, 3.2 | I-1.2.3 | A, B, C, D |
| 4 | testing-operator | 4.1, 4.2 | I-1.2.3.4 | A, B, C, D |
| 5 | bundling-operator | 5.1, 5.2 | I-1.2.3.4.5 | A, B, C, D |
| 6 | operator-reviewer | 6.1, 6.2 | I-6 | A, B, C, D |
| 7 | operator-test-generator | 7.1, 7.2 | I-7 | A, B, C, D |
| 8 | operator-bundle-validator | 8.1, 8.2 | I-8 | A, B, C, D |
| Final | All components | — | — | A, B, C, D |

**Total**: 16 unit tests + 7 integration tests + 4 E2E scenario tests = **27 test points**
