# Redis Operator — E2E Scenario Prompts

Progressive enhancement of a Redis operator across 5 scenarios. This is the **generality test** — validating that the skills (with 17 bug fixes from PostgreSQL) produce correct code for a different stateful workload without requiring further skill modifications.

**Operator location**: `e2e/redis-operator/`
**Validation guide**: `e2e/docs/statefulsets/redis-e2e-validation.md` (to be created)
**Status**: Pending
**Success criteria**: Zero new skill/subagent modifications needed.

| Scenario | Feature | Bundle | Workflow | New Resource | Bug Regression Coverage |
|----------|---------|--------|----------|-------------|------------------------|
| A | Core operator (from scratch) | v0.1.0 | A (new) | StatefulSet, Service×2, Secret, ConfigMap | #2, #3, #5, #10 |
| B | Redis Sentinel HA | v0.2.0 | B (modify) | Deployment + Service (sentinel) | #11, #12, #13, #17 |
| C | Webhooks + Network Security | v0.3.0 | C (webhooks) | NetworkPolicy | #14, #15 |
| D | API Maturity + TLS | v0.4.0 | D (versioning) | — (TLSSpec in v1beta1) | #15, #16 |
| E | Add Second CRD (RedisUser) | v0.5.0 | Expand (scaffolding B) | RedisUser CRD + controller | Multi-CRD gap |

## What Redis Tests Differently from PostgreSQL

| Aspect | PostgreSQL | Redis | What This Validates |
|--------|-----------|-------|---------------------|
| Services | 1 headless | 2 (headless + client ClusterIP) | Multiple Services in one reconciler |
| Auth | Optional BackupSpec | Required AuthSpec (password ref) | Required vs optional nested types |
| Config format | postgresql.conf (key=value) | redis.conf (directive-based) | ConfigMap not hardcoded |
| Ports | 5432 | 6379 (data) + 26379 (sentinel) | Different port numbers throughout |
| HA approach | PDB + anti-affinity | Sentinel Deployment (quorum-based) | Different HA architecture |
| Image | rhel9/postgresql-16 | rhel9/redis-7 | Different image, same pattern |
| Sidecar pattern | PgBouncer (pooler) | Redis Sentinel (monitor) | Different purpose, same Deployment pattern |

---

## Scenario A: Core Redis Cluster (New from Scratch)

```
Prompt: "Build me a complete OpenShift operator that manages Redis clusters.
Requirements:

Kind: RedisCluster
Group: cache
Domain: redis.example.com
Version: v1alpha1

Spec:
- replicas: 1-6, default 3
- version: enum "7.2"/"7.4", default "7.4"
- storage: size (string pattern ^[0-9]+[KMGT]i$), storageClassName (optional)
- resources: cpu/memory requests and limits (optional)
- auth: *AuthSpec (optional)
  - password: string (the Redis requirepass value)
  - existingSecret: string (name of existing Secret, mutually exclusive with password)

Status:
- phase: Pending/Initializing/Running/Failed/Degraded
- readyReplicas, currentVersion
- masterEndpoint (client service endpoint)
- conditions: Available, Progressing, Degraded

Controller should reconcile:
- Secret (auth credentials — generate random password if auth.password and 
  auth.existingSecret are both empty)
- ConfigMap (redis.conf with bind, port, maxmemory-policy, 
  appendonly, requirepass reference)
- Service (headless for StatefulSet DNS, port 6379, name <name>-headless)
- Service (client-facing ClusterIP, port 6379, name <name>-client)
- StatefulSet (Redis containers with PVCs, image registry.redhat.io/rhel9/redis-7,
  data dir /var/lib/redis/data)

Generate the complete project, tests, and OLM bundle v0.1.0 on alpha channel.
Review the code for best practices before finalizing."
```

**What this specifically validates**:
- Skills generate correct code for a non-PostgreSQL operand
- Two Services reconciled (headless + client) — tests multiple resources of the same type
- AuthSpec with mutually exclusive fields (password vs existingSecret)
- Redis-specific image and env vars (Bug #3/#17 fix: skill guides to check image docs)
- RHEL9 redis image: `registry.redhat.io/rhel9/redis-7`

**Acceptance criteria**:
- [ ] Complete project structure valid (validate-project-structure.sh passes)
- [ ] Types compile with all markers (validate-api-types.py passes)
- [ ] Controller compiles with 5 reconciler methods — reconcileSecret, reconcileConfigMap, reconcileHeadlessService, reconcileClientService, reconcileStatefulSet
- [ ] RBAC markers cover all managed resources (validate-rbac-annotations.py passes)
- [ ] All reconcilers follow check-create idempotency (check-idempotency.py passes)
- [ ] Tests compile and cover all methods (generate-test-matrix.py: 5/5 methods)
- [ ] Bundle valid (validate-csv.py + validate-bundle-structure.sh pass)
- [ ] Code review: 0 Critical (operator-reviewer subagent)
- [ ] No skill/subagent modifications required

---

## Scenario B: Add Redis Sentinel HA (Workflow B — Add Feature)

Builds on the Redis operator from Scenario A. Adds Sentinel for automatic failover.

```
Prompt: "Add Redis Sentinel HA support to the existing Redis operator at 
e2e/redis-operator/.

1. Add to the CRD: SentinelSpec with:
   - enabled: bool (default false)
   - replicas: int32 (min 3, max 7, default 3, should be odd for quorum)
   - image: string (optional, defaults to registry.redhat.io/rhel9/redis-7)
2. Add to Status: SentinelReady condition, sentinelEndpoint string
3. Add reconcileSentinel() method that:
   - Only created when spec.sentinel is non-nil and spec.sentinel.enabled is true
   - Creates a Deployment (apps/v1) for Sentinel with sentinel.replicas pods:
     Name: <name>-sentinel
     Port: 26379
     Command: redis-sentinel with appropriate config
     Labels: component=sentinel
   - Creates a Service (ClusterIP) for Sentinel:
     Name: <name>-sentinel  
     Port: 26379
   - When disabled, deletes both Deployment and Service
   - Sets owner references on both resources
   - Records events
4. Generate tests for the new reconciler method
5. Update the OLM bundle from v0.1.0 to v0.2.0 with replaces set correctly"
```

**What this specifically validates**:
- Workflow B adds a multi-resource feature (Deployment + Service in one reconciler)
- Conditional create/delete of TWO resources (like PostgreSQL D's connection pool, but at Workflow B stage)
- Check-update for ALL mutable fields per Bug #13 fix
- Bundle CRD refresh per Bug #12 fix
- Third-party image verification per Bug #17 fix

**Acceptance criteria**:
- [ ] SentinelSpec struct with enabled, replicas, image fields and markers
- [ ] reconcileSentinel() creates Deployment + Service when enabled, deletes both when disabled
- [ ] Deployment has correct sentinel port (26379), command, labels
- [ ] Service is ClusterIP (not headless) on port 26379
- [ ] SentinelReady condition set/cleared correctly
- [ ] RBAC for apps/deployments added
- [ ] Owns Deployment added to SetupWithManager
- [ ] Tests: create, idempotent, not-created-when-disabled, delete-when-disabled
- [ ] CSV v0.2.0 with replaces v0.1.0, Deployment RBAC + descriptors
- [ ] Code review: 0 Critical, bundle validates
- [ ] No skill/subagent modifications required

---

## Scenario C: Add Webhooks + NetworkPolicy (Workflow C — Webhooks)

Builds on Scenario B (v0.2.0). Adds defaulting/validating webhooks + NetworkPolicy.

**Prerequisite**: cert-manager operator on OpenShift.

```
Prompt: "Add admission webhooks and network security to the Redis operator at
e2e/redis-operator/ (which already has Sentinel support at v0.2.0).

1. Add webhooks (designing-operator-api Workflow C):
   Defaulting: set replicas=3 when 0, version=7.4 when empty,
   sentinel.replicas=3 when sentinel enabled but replicas not set
   Validating: 
   - reject sentinel.replicas as even number (quorum requires odd)
   - reject both auth.password and auth.existingSecret set (mutually exclusive)
   - reject storage size reduction on update
   - reject replicas < 1
2. Generate all webhook config files (service, cert-manager, patches)
3. Update main.go and kustomization files (including kustomize replacements 
   for cert-manager TLS — per Workflow C step 4)
4. Add reconcileNetworkPolicy() — creates NetworkPolicy from 
   networking.k8s.io/v1, allows port 6379 + 26379 ingress from same namespace,
   allows DNS egress, always created
5. Add NetworkSecured condition
6. Generate tests for NetworkPolicy + webhook validation
7. Update OLM bundle from v0.2.0 to v0.3.0 with webhook definitions"
```

**What this specifically validates**:
- Webhook generation for a different CRD (cache.redis.example.com, not database.postgres.example.com)
- cert-manager kustomize replacements work correctly (Bug #14 regression test)
- Redis-specific validation rules (odd sentinel replicas, mutually exclusive auth)
- NetworkPolicy with TWO ingress ports (6379 + 26379) vs PostgreSQL's one (5432)

**Acceptance criteria**:
- [ ] Webhook handler with Default() + ValidateCreate/Update/Delete()
- [ ] 9 webhook config files generated
- [ ] Kustomize build produces correct TLS DNS names (Bug #14 regression)
- [ ] reconcileNetworkPolicy() allows ports 6379 + 26379
- [ ] Webhook rejects even sentinel.replicas, mutually exclusive auth
- [ ] CSV v0.3.0 with replaces v0.2.0, webhookdefinitions, NP RBAC
- [ ] Code review: 0 Critical, bundle validates
- [ ] No skill/subagent modifications required

---

## Scenario D: API Versioning + TLS Support (Workflow D — Version Promotion)

Builds on Scenario C (v0.3.0). Promotes API to v1beta1 with TLS support.

```
Prompt: "Promote the Redis operator API to v1beta1 and add TLS support at 
e2e/redis-operator/ (which already has Sentinel + webhooks at v0.3.0).

1. Add API version v1beta1 (designing-operator-api Workflow D):
   Copy types to api/v1beta1/, add +kubebuilder:storageversion,
   add new fields:
   - tls: *TLSSpec (optional)
     - enabled: bool (default false)
     - secretName: string (name of Secret with tls.crt + tls.key)
     - certManager: bool (default false, if true auto-create Certificate)
   - maxMemory: *resource.Quantity (optional, for Redis maxmemory config)
   Add to status: tlsEnabled (bool)
2. Update main.go for v1beta1 scheme registration
3. Only register v1beta1 webhook (remove v1alpha1 webhook file — 
   per Workflow D step 4, storage version webhook handles all versions)
4. Do NOT apply CRD conversion webhook patch (strategy: None — new fields 
   are optional, per Workflow D step 5)
5. Add v1beta1 webhook validation: if tls.enabled and tls.secretName 
   is empty and tls.certManager is false, reject
6. Generate tests for v1beta1 webhook
7. Update OLM bundle from v0.3.0 to v0.4.0 with multi-version CRD,
   maturity alpha→beta, only v1beta1 webhookdefinitions in CSV"
```

**What this specifically validates**:
- API versioning for a second operator (not just PostgreSQL)
- CRD conversion strategy: None correctly applied (Bug #15 regression)
- Old webhook file removed, only storage version webhook registered (Bug #16 regression)
- CSV webhookdefinitions only contain v1beta1 entries (Bug #16 CSV part)
- TLSSpec as a different type of v1beta1 enhancement (vs PostgreSQL's ConnectionPoolSpec)
- No new reconciler method needed — TLS config is handled by updating ConfigMap and StatefulSet

**Acceptance criteria**:
- [ ] api/v1beta1/ directory with groupversion_info.go, types.go, deepcopy, webhook
- [ ] v1beta1 has +kubebuilder:storageversion, v1alpha1 does not
- [ ] TLSSpec with enabled, secretName, certManager fields
- [ ] v1alpha1 webhook file REMOVED (Bug #16 regression check)
- [ ] Only v1beta1 webhook registered in main.go
- [ ] CRD conversion strategy: None (no webhook patch in CRD kustomization) (Bug #15 regression check)
- [ ] CSV webhookdefinitions only contain v1beta1 paths (Bug #16 CSV regression check)
- [ ] v1beta1 webhook rejects tls.enabled without secretName/certManager
- [ ] CSV v0.4.0 with replaces v0.3.0, multi-version CRD, maturity=beta
- [ ] Code review: 0 Critical, bundle validates
- [ ] No skill/subagent modifications required

---

## Scenario E: Add Second CRD — RedisUser (Scaffolding Workflow B — Expand)

Builds on Scenario D (v0.4.0). Adds a second CRD `RedisUser` to the same operator in the same API group. This tests multi-CRD operator support — the most common gap not covered by Workflows A-D.

**Why this matters**: Production operators almost always manage multiple CRDs (Kafka has KafkaTopic/KafkaUser/KafkaConnect, cert-manager has Certificate/Issuer/ClusterIssuer). The scaffolding skill's Workflow B (Pattern B: same-group) was unit-tested but never E2E tested with real controller code, tests, and bundle updates.

```
Prompt: "Add a RedisUser CRD to the existing Redis operator at 
e2e/redis-operator/ (which already has RedisCluster at v0.4.0 with 
Sentinel, webhooks, and TLS support).

The RedisUser represents a Redis ACL user managed by the operator.

1. Scaffold the new CRD (scaffolding-operator Workflow B, Pattern B — 
   same group):
   Kind: RedisUser
   Group: cache (same as RedisCluster)
   Domain: redis.example.com (same)
   Version: v1beta1 (match the storage version)
   
   This should add types to api/v1beta1/ (same package as RedisCluster)
   and a new controller file. Do NOT create a separate api/ package —
   same-group resources share the api/<version>/ package.

2. Design the RedisUser CRD (designing-operator-api Workflow A for new types):
   Spec:
   - username: string (required, the Redis ACL username)
   - permissions: []string (Redis ACL rules, e.g. '+@read', '~key:*')
   - clusterRef: string (required, name of the RedisCluster this user belongs to)
   - passwordSecret: string (optional, name of Secret containing the password.
     If empty, operator generates a random password)
   
   Status:
   - phase: Pending/Active/Failed
   - conditions: Available
   - passwordSecretName: string (name of Secret with generated password)

3. Implement the RedisUser controller (implementing-reconciliation Workflow A 
   for new controller):
   - reconcileUserSecret() — create Secret with generated password if 
     passwordSecret not provided
   - reconcileUserACL() — create ConfigMap with ACL rules that the 
     RedisCluster StatefulSet can mount
   - Register the new controller in main.go with its own Reconcile loop
   - Add RBAC for the new CRD (redisusers, redisusers/status, redisusers/finalizers)

4. Generate tests for the RedisUser controller (operator-test-generator)
5. Review the complete project (operator-reviewer)
6. Update the OLM bundle from v0.4.0 to v0.5.0:
   - Add RedisUser as a second owned CRD in the CSV
   - Add specDescriptors/statusDescriptors for RedisUser
   - Add RBAC for redisusers in clusterPermissions
   - Update alm-examples with a RedisUser sample
   - Keep all existing RedisCluster sections unchanged"
```

**What this specifically validates**:
- `scaffolding-operator` Workflow B (Pattern B: same-group) — adds a second Kind to existing api/v1beta1/ package
- Controller registration — main.go now has TWO controllers (RedisCluster + RedisUser)
- Separate reconcile loops — RedisUser has its own Reconcile() method and RBAC
- Bundle with multiple owned CRDs — CSV lists both RedisCluster and RedisUser
- Types in shared package — both kinds in api/v1beta1/ without import conflicts
- Cross-resource reference — RedisUser.spec.clusterRef points to a RedisCluster

**Acceptance criteria**:
- [ ] RedisUser types added to api/v1beta1/ (same package as RedisCluster)
- [ ] RedisUser has its own List type and SchemeBuilder.Register
- [ ] New controller file for RedisUser with its own Reconcile()
- [ ] main.go registers BOTH controllers (RedisCluster + RedisUser)
- [ ] RedisUser RBAC markers separate from RedisCluster RBAC
- [ ] reconcileUserSecret() and reconcileUserACL() follow check-create pattern
- [ ] Tests: generate-test-matrix shows coverage for RedisUser methods
- [ ] CSV v0.5.0 with TWO owned CRDs (RedisCluster + RedisUser)
- [ ] CSV has specDescriptors/statusDescriptors for both CRDs
- [ ] alm-examples includes samples for both CRDs
- [ ] Code review: 0 Critical, bundle validates
- [ ] No skill/subagent modifications required

---

## Skill Coverage Matrix

| Scenario | scaffolding | designing-api | implementing-reconciliation | bundling | test-generator | reviewer | bundle-validator |
|----------|-------------|--------------|---------------------------|----------|----------------|----------|-----------------|
| A | Workflow A | Workflow A | Workflow A | Workflow A | Full | Full | Full |
| B | — | Workflow B | Workflow B | Workflow B | Workflow B | Full | Full |
| C | — | Workflow C | Workflow B | Workflow B | Workflow B | Full | Full |
| D | — | Workflow D | — | Workflow B | Workflow B | Full | Full |
| E | **Workflow B** | Workflow A (new types) | Workflow A (new controller) | Workflow B | Full | Full | Full |

## Bug Regression Matrix

Each scenario targets specific bug fixes from PostgreSQL to verify they hold:

| Bug # | Description | Tested In |
|-------|-------------|-----------|
| 2 | Dockerfile cross-compile | A (image build) |
| 3 | OpenShift image compatibility | A (rhel9/redis-7) |
| 5 | UBI base image | A (Dockerfile) |
| 10 | manager.yaml namespace | A (scaffold) |
| 11 | Check-update for modified reconcilers | B (sentinel adds to StatefulSet?) |
| 12 | Bundle CRD refresh | B (v0.2.0 bundle) |
| 13 | Audit ALL mutable fields | B (Deployment check-update) |
| 14 | cert-manager kustomize replacements | C (webhook TLS) |
| 15 | CRD conversion strategy: None | D (multi-version CRD) |
| 16 | Multi-version webhook stripping + CSV | D (v1beta1 only webhook) |
| 17 | Third-party image env vars | A, B (redis, sentinel images) |
| — | Multi-CRD support (new gap) | E (second CRD in same group) |

## Version Chain

| Scenario | Bundle | Replaces | Maturity | API Versions | CRDs |
|----------|--------|----------|----------|-------------|------|
| A | 0.1.0 | — | alpha | v1alpha1 | RedisCluster |
| B | 0.2.0 | 0.1.0 | alpha | v1alpha1 | RedisCluster |
| C | 0.3.0 | 0.2.0 | alpha | v1alpha1 | RedisCluster |
| D | 0.4.0 | 0.3.0 | beta | v1alpha1 + v1beta1 | RedisCluster |
| E | 0.5.0 | 0.4.0 | beta | v1beta1 | RedisCluster + RedisUser |
