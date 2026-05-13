# MongoDB Operator — E2E Scenario Prompts

Progressive enhancement of a MongoDB operator across 5 scenarios. This is the **gap-coverage test** — targeting skill patterns that PostgreSQL (111 tests) and Redis (139 tests) did not exercise: **Job reconciliation (batch/v1)** and **different-group multi-CRD (scaffolding Workflow C)**.

**Operator location**: `e2e/mongodb-operator/`
**Validation guide**: `e2e/docs/statefulsets/mongodb-e2e-validation.md` (to be created)
**Status**: Pending
**Success criteria**: Zero new skill/subagent modifications needed. Both untested gaps validated.

| Scenario | Feature | Bundle | Workflow | New Resource | Gap Tested |
|----------|---------|--------|----------|-------------|------------|
| A | Core operator with Job backup | v0.1.0 | A (new) | StatefulSet, Service×2, Secret, ConfigMap, Job | **Job (batch/v1)** |
| B | Add Arbiter node | v0.2.0 | B (modify) | Deployment (arbiter) | 3rd operand validation |
| C | Webhooks + Network Security | v0.3.0 | C (webhooks) | NetworkPolicy | 3rd operand validation |
| D | API Maturity + Sharding Config | v0.4.0 | D (versioning) | — (ShardingSpec in v1beta1) | 3rd operand validation |
| E | Different-group CRD (MongoBackupPolicy) | v0.5.0 | Expand (scaffolding C) | MongoBackupPolicy CRD + controller | **Scaffolding Workflow C** |

## Why MongoDB — Gap Analysis

After 2 operators and 250 tests, we confirmed skills are general-purpose for all tested patterns. MongoDB targets the two highest-severity untested gaps:

| Gap | Severity | PostgreSQL | Redis | MongoDB |
|-----|----------|-----------|-------|---------|
| Job (batch/v1) reconciliation | HIGH | CronJob only | — | **Scenario A: mongodump Job** |
| Different-group CRD layout | HIGH | — | Same-group (Scenario E) | **Scenario E: backup group** |
| Cluster-scoped CRDs | HIGH | — | — | Doesn't fit — needs Infrastructure category |
| CEL validation rules | MEDIUM | — | — | Not worth forcing — webhook validation covers same need |
| Conversion webhooks (hub-spoke) | MEDIUM | — | — | Not worth forcing — requires contrived breaking change |

## What MongoDB Tests Differently

| Aspect | PostgreSQL | Redis | MongoDB | What This Validates |
|--------|-----------|-------|---------|---------------------|
| Backup mechanism | CronJob (trigger) | — | **Job (batch/v1)** | New resource type never reconciled before |
| Multi-CRD layout | — | Same-group (cache) | **Different-group (database + backup)** | Scaffolding Workflow C with aliased imports |
| HA approach | PDB + anti-affinity | Sentinel Deployment | Arbiter Deployment (vote-only) | 3rd validation of conditional Deployment |
| Config format | postgresql.conf (key=value) | redis.conf (directive) | mongod.conf (YAML) | Different ConfigMap format |
| Ports | 5432 | 6379 + 26379 | 27017 | Different port |
| Auth | user/password Secret | requirepass Secret | keyFile Secret + user/password | Inter-node auth (keyFile for replica set) |
| Image | rhel9/postgresql-16 | rhel9/redis-7 | UBI micro mock (enterprise MongoDB needs license) | E2E image strategy |
| Replica Set | — | — | Replica set with election | MongoDB-specific topology |

---

## Scenario A: Core MongoDB Replica Set with Job Backup (New from Scratch)

```
Prompt: "Build me a complete OpenShift operator that manages MongoDB 
replica sets. Requirements:

Kind: MongoCluster
Group: database
Domain: mongodb.example.com
Version: v1alpha1

Spec:
- replicas: 1-7, default 3 (should be odd for elections)
- version: enum "7.0"/"8.0", default "7.0"
- storage: size (string pattern ^[0-9]+[KMGT]i$), storageClassName (optional)
- resources: cpu/memory requests and limits (optional)
- auth: *AuthSpec (optional)
  - adminPassword: string (the admin user password)
  - existingSecret: string (name of existing Secret, mutually exclusive with adminPassword)
  - keyFile: string (optional, name of existing Secret with replica set keyFile.
    If empty, operator generates a random keyFile)
- backup: *BackupSpec (optional)
  - enabled: bool (default false)
  - retentionDays: int32 (1-30, default 7)

Status:
- phase: Pending/Initializing/Running/Failed/Degraded
- readyReplicas, currentVersion
- primaryEndpoint (client service endpoint)
- conditions: Available, Progressing, Degraded, BackupReady
- lastBackupTime (metav1.Time, when last backup Job completed)

Controller should reconcile:
- Secret (admin credentials — generate random password if auth.adminPassword
  and auth.existingSecret are both empty)
- Secret (keyFile for replica set internal auth — generate random 
  keyFile if auth.keyFile is empty)
- ConfigMap (mongod.conf in YAML format with storage.dbPath, 
  net.port, replication.replSetName, security.keyFile reference)
- Service (headless for StatefulSet DNS, port 27017, name <name>-headless)
- Service (client-facing ClusterIP, port 27017, name <name>-client)
- StatefulSet (MongoDB containers with PVCs, image 
  registry.access.redhat.com/ubi9/ubi-micro with sleep infinity as mock
  container for E2E testing — Red Hat certified MongoDB images 
  (registry.connect.redhat.com/mongodb/enterprise-*) require enterprise 
  license. Data dir /var/lib/mongodb/data)
- Job (batch/v1) — if backup.enabled, create a one-shot backup Job:
  Name: <name>-backup-<timestamp>
  Image: same as StatefulSet (UBI micro for E2E)
  Command: /bin/sleep 5 (mock — completes in 5 seconds for E2E.
    Production would use mongodump --archive --gzip with the real 
    MongoDB image, admin Secret env vars, and backup PVC mount)
  Owner reference to MongoCluster
  Only create if no active backup Job exists (check-create with 
  label selector, not just by name)
  Set BackupReady condition and lastBackupTime on completion

Generate the complete project, tests, and OLM bundle v0.1.0 on alpha channel.
Review the code for best practices before finalizing."
```

**What this specifically validates**:
- **Job (batch/v1) reconciliation** — genuinely untested resource type. Neither PostgreSQL nor Redis reconcile a Job directly
- Job check-create pattern differs from other resources — Jobs are immutable once created, so the pattern is "check if active Job exists by label, create new if not"
- Two Secrets (admin + keyFile) — multiple Secrets in one reconciler
- E2E image: UBI micro mock container (`registry.access.redhat.com/ubi9/ubi-micro` with `sleep infinity`). Red Hat certified MongoDB images (`registry.connect.redhat.com/mongodb/enterprise-*`) require enterprise license — mock container tests operator reconciliation without needing a real MongoDB process
- YAML-format ConfigMap (vs key=value for PostgreSQL/Redis)
- RBAC must include `batch/jobs` — new API group for RBAC markers

**Acceptance criteria**:
- [ ] Complete project structure valid (validate-project-structure.sh passes)
- [ ] Types compile with all markers (validate-api-types.py passes)
- [ ] Controller compiles with 7 reconciler methods — reconcileAdminSecret, reconcileKeyFileSecret, reconcileConfigMap, reconcileHeadlessService, reconcileClientService, reconcileStatefulSet, reconcileBackupJob
- [ ] RBAC markers cover all managed resources including `batch/jobs` (validate-rbac-annotations.py passes)
- [ ] All reconcilers follow check-create idempotency (check-idempotency.py passes)
- [ ] **Job reconciler uses label selector for existence check** (not name-based, since Jobs have timestamps)
- [ ] Tests compile and cover all methods (generate-test-matrix.py: 7/7 methods)
- [ ] Bundle valid (validate-csv.py + validate-bundle-structure.sh pass)
- [ ] Code review: 0 Critical (operator-reviewer subagent)
- [ ] No skill/subagent modifications required

---

## Scenario B: Add Arbiter Node (Workflow B — Add Feature)

Builds on the MongoDB operator from Scenario A. Adds an arbiter node for replica set elections without storing data.

```
Prompt: "Add MongoDB Arbiter support to the existing MongoDB operator at
e2e/mongodb-operator/.

1. Add to the CRD: ArbiterSpec with:
   - enabled: bool (default false)
   - resources: ResourceRequirements (optional, arbiter needs minimal resources)
2. Add to Status: ArbiterReady condition
3. Add reconcileArbiter() method that:
   - Only created when spec.arbiter is non-nil and spec.arbiter.enabled is true
   - Creates a Deployment (apps/v1) for the arbiter:
     Name: <name>-arbiter
     Replicas: 1 (always 1 — arbiter is a single vote-only member)
     Port: 27017
     Image: same MongoDB image
     Command: mongod --replSet <name> --port 27017 --noauth (arbiter 
       doesn't store data, just votes)
     No PVC — arbiter doesn't persist data
     Labels: component=arbiter
   - When disabled, deletes the Deployment
   - Sets owner references
   - Records events
4. Generate tests for the new reconciler method
5. Review the code
6. Update the OLM bundle from v0.1.0 to v0.2.0 with replaces"
```

**What this specifically validates**:
- Workflow B adds a conditional single-replica Deployment (vs Redis Sentinel's multi-replica)
- Arbiter has NO PVC — validates Deployment without volume mounts (simpler pattern)
- Check-update for Deployment mutable fields (resources, image)
- Bundle CRD refresh per Bug #12 fix

**Acceptance criteria**:
- [ ] ArbiterSpec struct with enabled and resources fields
- [ ] reconcileArbiter() creates Deployment when enabled, deletes when disabled
- [ ] Deployment has 1 replica (always), no PVC, correct MongoDB command
- [ ] ArbiterReady condition set/cleared correctly
- [ ] RBAC for apps/deployments added
- [ ] Owns Deployment added to SetupWithManager
- [ ] Tests: create, idempotent, not-created-when-disabled, delete-when-disabled
- [ ] CSV v0.2.0 with replaces v0.1.0
- [ ] Code review: 0 Critical, bundle validates
- [ ] No skill/subagent modifications required

---

## Scenario C: Add Webhooks + NetworkPolicy (Workflow C — Webhooks)

Builds on Scenario B (v0.2.0). Adds defaulting/validating webhooks + NetworkPolicy.

**Prerequisite**: cert-manager operator on OpenShift.

```
Prompt: "Add admission webhooks and network security to the MongoDB operator at
e2e/mongodb-operator/ (which already has Arbiter support at v0.2.0).

1. Add webhooks (designing-operator-api Workflow C):
   Defaulting: set replicas=3 when 0, version=7.0 when empty,
   backup.retentionDays=7 when backup.enabled but retentionDays not set
   Validating:
   - reject replicas as even number (replica set elections require odd count)
   - reject both auth.adminPassword and auth.existingSecret set (mutually exclusive)
   - reject storage size reduction on update
   - reject replicas < 1
   - reject backup.retentionDays > 30
2. Generate all webhook config files (service, cert-manager, patches)
3. Update main.go and kustomization files (including kustomize replacements 
   for cert-manager TLS — per Workflow C step 4)
4. Add reconcileNetworkPolicy() — creates NetworkPolicy from 
   networking.k8s.io/v1, allows port 27017 ingress from same namespace,
   allows DNS egress, always created
5. Add NetworkSecured condition
6. Generate tests for NetworkPolicy + webhook validation
7. Update OLM bundle from v0.2.0 to v0.3.0 with webhook definitions"
```

**What this specifically validates**:
- Webhook generation for a third CRD group (database.mongodb.example.com)
- cert-manager kustomize replacements (Bug #14 regression — third validation)
- MongoDB-specific validation: odd replica count for elections (different from Redis sentinel quorum)
- NetworkPolicy with single port (27017) — different from Redis's two ports

**Acceptance criteria**:
- [ ] Webhook handler with Default() + ValidateCreate/Update/Delete()
- [ ] 9 webhook config files generated
- [ ] Kustomize build produces correct TLS DNS names (Bug #14 regression)
- [ ] reconcileNetworkPolicy() allows port 27017
- [ ] Webhook rejects even replicas, mutually exclusive auth, retention > 30
- [ ] CSV v0.3.0 with replaces v0.2.0, webhookdefinitions, NP RBAC
- [ ] Code review: 0 Critical, bundle validates
- [ ] No skill/subagent modifications required

---

## Scenario D: API Versioning + Sharding Config (Workflow D — Version Promotion)

Builds on Scenario C (v0.3.0). Promotes API to v1beta1 with sharding configuration.

```
Prompt: "Promote the MongoDB operator API to v1beta1 and add sharding 
configuration at e2e/mongodb-operator/ (which already has Arbiter + 
webhooks at v0.3.0).

1. Add API version v1beta1 (designing-operator-api Workflow D):
   Copy types to api/v1beta1/, add +kubebuilder:storageversion,
   add new fields:
   - sharding: *ShardingSpec (optional)
     - enabled: bool (default false)
     - shards: int32 (min 1, max 10, default 2)
     - configServerReplicas: int32 (min 1, max 5, default 3)
   - maxConnections: *int32 (optional, for net.maxIncomingConnections)
   Add to status: shardingEnabled (bool)
2. Update main.go for v1beta1 scheme registration
3. Only register v1beta1 webhook (remove v1alpha1 webhook file — 
   per Workflow D step 4)
4. Do NOT apply CRD conversion webhook patch (strategy: None)
5. Add v1beta1 webhook validation: if sharding.enabled, shards must be >= 1
6. Generate tests for v1beta1 webhook
7. Update OLM bundle from v0.3.0 to v0.4.0 with multi-version CRD,
   maturity alpha→beta, only v1beta1 webhookdefinitions in CSV"
```

**What this specifically validates**:
- API versioning for a third operator
- CRD conversion strategy: None (Bug #15 — third regression test)
- Old webhook file removed (Bug #16 — third regression test)
- ShardingSpec as a different type of v1beta1 enhancement
- No new reconciler needed — sharding config is preparation for future controller work

**Acceptance criteria**:
- [ ] api/v1beta1/ with storageversion, ShardingSpec, maxConnections
- [ ] v1alpha1 webhook file REMOVED (Bug #16)
- [ ] Only v1beta1 webhook registered in main.go
- [ ] CRD conversion strategy: None (Bug #15)
- [ ] CSV webhookdefinitions only contain v1beta1 paths (Bug #16 CSV)
- [ ] CSV v0.4.0 with replaces v0.3.0, multi-version CRD, maturity=beta
- [ ] Code review: 0 Critical, bundle validates
- [ ] No skill/subagent modifications required

---

## Scenario E: Different-Group CRD — MongoBackupPolicy (Scaffolding Workflow C)

Builds on Scenario D (v0.4.0). Adds `MongoBackupPolicy` CRD in a **different API group** (`backup`) from `MongoCluster` (`database`). This is the highest-value scenario — scaffolding Workflow C (different-group layout) has **never been E2E tested**.

**Why this matters**: Production operators frequently span multiple API groups (cert-manager has `cert-manager.io` and `acme.cert-manager.io`; Strimzi has `kafka.strimzi.io` and `core.strimzi.io`). The scaffolding skill's Workflow C generates a multi-group layout with separate `api/<group>/` directories and aliased imports — this path has only been unit-tested, never validated end-to-end with real controller code, tests, and bundle.

```
Prompt: "Add a MongoBackupPolicy CRD to the existing MongoDB operator at
e2e/mongodb-operator/ (which already has MongoCluster at v0.4.0 with 
Arbiter, webhooks, and sharding config).

The MongoBackupPolicy defines scheduled backup policies that reference
MongoCluster instances.

1. Scaffold the new CRD (scaffolding-operator Workflow B, Pattern C — 
   different group):
   Kind: MongoBackupPolicy
   Group: backup (DIFFERENT from MongoCluster's 'database' group)
   Domain: mongodb.example.com (same domain)
   Version: v1beta1
   
   This should create a NEW api/v1beta1/ package under the backup group
   (NOT in the same package as MongoCluster). The PROJECT file should 
   have multigroup=true. Imports in main.go need aliased names:
   databasev1beta1 vs backupv1beta1.

2. Design the MongoBackupPolicy CRD (designing-operator-api Workflow A):
   Spec:
   - clusterRef: string (required, name of MongoCluster to back up)
   - schedule: string (required, cron expression for backup frequency)
   - retentionDays: int32 (1-90, default 30)
   - storageSize: string (required, pattern ^[0-9]+[KMGT]i$, PVC size for backups)
   
   Status:
   - phase: Pending/Active/Failed
   - conditions: Available
   - lastBackupTime: *metav1.Time
   - nextBackupTime: *metav1.Time
   - backupCount: int32

3. Implement the MongoBackupPolicy controller (implementing-reconciliation 
   Workflow A for new controller):
   - reconcilePolicyCronJob() — create CronJob that triggers mongodump
     on the referenced MongoCluster on the specified schedule
   - reconcileBackupPVC() — create PVC for storing backups
   - Register the new controller in main.go with aliased imports
   - Add RBAC for the new CRD (mongobackuppolicies, mongobackuppolicies/status,
     mongobackuppolicies/finalizers)
   - Add RBAC for batch/cronjobs and PVCs

4. Generate tests for the MongoBackupPolicy controller
5. Review the complete project
6. Update the OLM bundle from v0.4.0 to v0.5.0:
   - Add MongoBackupPolicy as a second owned CRD in the CSV
   - The CRD is in group backup.mongodb.example.com (different from 
     database.mongodb.example.com)
   - Add specDescriptors/statusDescriptors for MongoBackupPolicy
   - Update alm-examples with a MongoBackupPolicy sample"
```

**What this specifically validates**:
- **`scaffolding-operator` Workflow C (Pattern C: different-group)** — creates separate api directory, multi-group PROJECT layout, aliased imports in main.go. NEVER E2E tested.
- Multi-group PROJECT file with `multigroup: true`
- Aliased imports: `databasev1beta1` vs `backupv1beta1` in main.go and controller
- Separate api packages: `api/database/v1beta1/` and `api/backup/v1beta1/` (or equivalent layout)
- Two different API groups in one CSV: `database.mongodb.example.com` and `backup.mongodb.example.com`
- Cross-group CRD reference: MongoBackupPolicy.spec.clusterRef → MongoCluster
- CronJob reconciliation (tested in PostgreSQL A but not in a different-group context)
- PVC reconciliation (implicitly tested via StatefulSet VCT, but not as a standalone reconciled resource)

**Acceptance criteria**:
- [ ] PROJECT file has `multigroup: true` (or equivalent)
- [ ] Separate api packages for database and backup groups
- [ ] main.go has aliased imports (databasev1beta1, backupv1beta1)
- [ ] MongoBackupPolicy controller has its own Reconcile() with RBAC
- [ ] reconcilePolicyCronJob() and reconcileBackupPVC() follow check-create pattern
- [ ] RBAC markers include batch/cronjobs, core/persistentvolumeclaims
- [ ] Tests: generate-test-matrix shows coverage for MongoBackupPolicy methods
- [ ] CSV v0.5.0 with TWO owned CRDs from DIFFERENT groups
- [ ] Both CRD groups represented in CSV owned section
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
| E | **Workflow C** | Workflow A (new types) | Workflow A (new controller) | Workflow B | Full | Full | Full |

## Gap Coverage Matrix

Shows what MongoDB tests that PostgreSQL + Redis did NOT:

| Gap | First Tested In | Scenario |
|-----|----------------|----------|
| Job (batch/v1) reconciliation | **MongoDB** | A |
| Different-group CRD layout (scaffolding Workflow C) | **MongoDB** | E |
| Multi-group PROJECT file | **MongoDB** | E |
| Aliased imports in main.go | **MongoDB** | E |
| Two API groups in one CSV | **MongoDB** | E |
| Standalone PVC reconciliation | **MongoDB** | E |
| Odd-replica election validation | **MongoDB** | C |
| YAML-format ConfigMap | **MongoDB** | A |
| Multiple Secrets (admin + keyFile) | **MongoDB** | A |
| batch/jobs RBAC group | **MongoDB** | A |

## Bug Regression Matrix

Each scenario continues to regression-test PostgreSQL bug fixes:

| Bug # | Description | Tested In |
|-------|-------------|-----------|
| 2 | Dockerfile cross-compile | A (image build) |
| 3 | OpenShift image compatibility | A (UBI micro mock) |
| 5 | UBI base image | A (Dockerfile) |
| 10 | manager.yaml namespace | A (scaffold) |
| 11 | Check-update for modified reconcilers | B (arbiter adds to reconciler) |
| 12 | Bundle CRD refresh | B (v0.2.0 bundle) |
| 13 | Audit ALL mutable fields | B (Deployment check-update) |
| 14 | cert-manager kustomize replacements | C (webhook TLS) |
| 15 | CRD conversion strategy: None | D (multi-version CRD) |
| 16 | Multi-version webhook stripping + CSV | D (v1beta1 only webhook) |
| 17 | Third-party image env vars | A (mongodb image) |
| — | Job pattern (new gap) | A (backup Job) |
| — | Different-group CRD (new gap) | E (backup group) |

## Version Chain

| Scenario | Bundle | Replaces | Maturity | API Versions | CRDs |
|----------|--------|----------|----------|-------------|------|
| A | 0.1.0 | — | alpha | v1alpha1 | MongoCluster |
| B | 0.2.0 | 0.1.0 | alpha | v1alpha1 | MongoCluster |
| C | 0.3.0 | 0.2.0 | alpha | v1alpha1 | MongoCluster |
| D | 0.4.0 | 0.3.0 | beta | v1alpha1 + v1beta1 | MongoCluster |
| E | 0.5.0 | 0.4.0 | beta | v1beta1 | MongoCluster + MongoBackupPolicy |

## Gaps NOT Addressed by MongoDB (Needs Different Category)

| Gap | Why Not MongoDB | Better Tested With |
|-----|----------------|-------------------|
| Cluster-scoped CRDs (scaffolding Workflow D) | MongoDB is namespace-scoped | Infrastructure category (ClusterAutoScaler) |
| CEL validation rules | Webhook validation covers same need | Not worth a dedicated operator |
| Conversion webhooks (hub-spoke) | No natural breaking change | When a real breaking change arises |
| ServiceMonitor reconciliation | Forced for MongoDB | Observability category (Prometheus operator) |
| DaemonSet reconciliation | MongoDB doesn't need per-node pods | Infrastructure category |
