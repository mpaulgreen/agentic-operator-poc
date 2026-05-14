# Elasticsearch Operator — E2E Scenario Prompts

Progressive enhancement of an Elasticsearch operator across 5 scenarios. This is the **N=4 generality proof** — after PostgreSQL (111 tests, 17 fixes), Redis (139 tests, 0 fixes), and MongoDB (150 tests, 1 fix), Elasticsearch validates that skills produce correct operator code for a 4th stateful workload without further modifications.

**Operator location**: `e2e/elasticsearch-operator/`
**Validation guide**: `e2e/docs/statefulsets/elasticsearch-e2e-validation.md` (to be created)
**Status**: Pending
**Success criteria**: Zero new skill/subagent modifications needed.

| Scenario | Feature | Bundle | Workflow | New Resource | Regression Coverage |
|----------|---------|--------|----------|-------------|----------------------|
| A | Core operator (data nodes) | v0.1.0 | A (new) | StatefulSet, Service×2 (HTTP+transport), Secret, ConfigMap | All Bug #1-18 |
| B | Dedicated master nodes | v0.2.0 | B (modify) | Deployment (master) | #11, #12, #13 |
| C | Webhooks + Network Security | v0.3.0 | C (webhooks) | NetworkPolicy | #14 |
| D | API Maturity + ILM Config | v0.4.0 | D (versioning) | — (ILMSpec in v1beta1) | #15, #16 |
| E | Same-group CRD (ElasticsearchIndex) | v0.5.0 | Expand (scaffolding B) | ElasticsearchIndex CRD + controller | Same-group 2nd test |

## Assessment: Why Elasticsearch After 400 Tests?

After 3 operators and 400 passing tests, all skill workflows are validated:
- Scaffolding Workflows A, B (same-group), C (different-group): all proven
- Designing-API Workflows A-D: all proven 3x each
- Implementing-Reconciliation Workflows A-B: all proven 3x each
- Bundling Workflows A-B: all proven 3x each
- All 3 subagents: all proven 3x each

**What Elasticsearch adds**: N=4 confirmation for the stateful workloads category. If it also finds zero skill bugs, this provides strong statistical confidence that the skills are truly general-purpose — not just "happened to work for 3 specific databases."

**Deferred gaps NOT addressed by Elasticsearch** (need different categories):
- Cluster-scoped CRDs → Infrastructure / Cloud
- DaemonSet reconciliation → Infrastructure / Cloud
- ServiceMonitor reconciliation → Observability
- CEL validation rules → Not a significant gap (webhook validation covers same need)
- Conversion webhooks → Needs a natural breaking API change

## What Elasticsearch Tests Differently

| Aspect | PostgreSQL | Redis | MongoDB | Elasticsearch | What This Validates |
|--------|-----------|-------|---------|---------------|---------------------|
| Ports | 5432 | 6379 + 26379 | 27017 | 9200 (HTTP) + 9300 (transport) | Two ports on StatefulSet (both required) |
| HA approach | PDB + anti-affinity | Sentinel Deployment | Arbiter Deployment | Dedicated master Deployment | Different HA pattern (election via discovery) |
| Config format | postgresql.conf (key=value) | redis.conf (directive) | mongod.conf (YAML) | elasticsearch.yml (YAML) | 2nd YAML config |
| Auth | user/password Secret | requirepass Secret | keyFile + password | Security plugin Secret | Different auth model |
| Multi-CRD | — | Same-group (RedisUser) | Different-group (MongoBackupPolicy) | Same-group (ElasticsearchIndex) | 2nd same-group test |
| Backup | CronJob | — | Job | CronJob (snapshot API) | 2nd CronJob validation |
| Services | 1 headless | 2 (headless + client) | 2 (headless + client) | 2 (HTTP + transport) | Different service purposes |
| Image | rhel9/postgresql-16 | rhel9/redis-7 | UBI micro mock | UBI micro mock | Same E2E approach |

---

## Scenario A: Core Elasticsearch Cluster (New from Scratch)

```
Prompt: "Build me a complete OpenShift operator that manages Elasticsearch 
clusters. Requirements:

Kind: ElasticsearchCluster
Group: search
Domain: elasticsearch.example.com
Version: v1alpha1

Spec:
- replicas: 1-9, default 3
- version: enum "8.12"/"8.14", default "8.14"
- storage: size (string pattern ^[0-9]+[KMGT]i$), storageClassName (optional)
- resources: cpu/memory requests and limits (optional)
- auth: *AuthSpec (optional)
  - adminPassword: string (the elastic superuser password)
  - existingSecret: string (name of existing Secret, mutually exclusive
    with adminPassword)
- backup: *BackupSpec (optional)
  - enabled: bool (default false)
  - schedule: string (cron expression for snapshot frequency)
  - retentionDays: int32 (1-30, default 7)

Status:
- phase: Pending/Initializing/Running/Failed/Degraded
- readyReplicas, currentVersion
- httpEndpoint (HTTP service endpoint, port 9200)
- conditions: Available, Progressing, Degraded, BackupReady

Controller should reconcile:
- Secret (admin credentials — generate random password if auth.adminPassword
  and auth.existingSecret are both empty)
- ConfigMap (elasticsearch.yml in YAML format with cluster.name,
  node.name pattern, network.host, http.port 9200, 
  transport.port 9300, discovery.seed_hosts)
- Service (HTTP API, ClusterIP, port 9200, name <name>-http)
- Service (transport/inter-node, headless ClusterIP None, port 9300, 
  name <name>-transport)
- StatefulSet (Elasticsearch containers with PVCs, image
  registry.access.redhat.com/ubi9/ubi-micro with sleep infinity as mock,
  data dir /usr/share/elasticsearch/data)
- CronJob (if backup.enabled, snapshot on schedule — mock with 
  /bin/sleep 5 for E2E testing)

Generate the complete project, tests, and OLM bundle v0.1.0 on alpha channel.
Review the code for best practices before finalizing."
```

**What this specifically validates**:
- Skills generate correct code for a 4th stateful workload
- Two Services with different purposes (HTTP API vs transport) — tests naming pattern
- Two-port StatefulSet (9200 + 9300)
- CronJob-based backup (2nd validation after PostgreSQL)
- Elasticsearch-specific YAML ConfigMap

**Acceptance criteria**:
- [ ] Complete project structure valid (validate-project-structure.sh passes)
- [ ] Types compile with all markers (validate-api-types.py passes)
- [ ] Controller compiles with 6 reconciler methods — reconcileSecret, reconcileConfigMap, reconcileHTTPService, reconcileTransportService, reconcileStatefulSet, reconcileBackupCronJob
- [ ] RBAC markers cover all managed resources (validate-rbac-annotations.py passes)
- [ ] All reconcilers follow check-create idempotency (check-idempotency.py passes)
- [ ] Tests compile and cover all methods (generate-test-matrix.py: 6/6 methods)
- [ ] Bundle valid (validate-csv.py + validate-bundle-structure.sh pass)
- [ ] Code review: 0 Critical (operator-reviewer subagent)
- [ ] No skill/subagent modifications required

---

## Scenario B: Dedicated Master Nodes (Workflow B — Add Feature)

Builds on the Elasticsearch operator from Scenario A. Adds dedicated master-eligible nodes as a separate Deployment.

```
Prompt: "Add dedicated master node support to the existing Elasticsearch 
operator at e2e/elasticsearch-operator/.

1. Add to the CRD: MasterSpec with:
   - enabled: bool (default false)
   - replicas: int32 (min 1, max 5, default 3, should be odd for quorum)
   - resources: ResourceRequirements (optional, master needs less 
     resources than data nodes)
2. Add to Status: MasterReady condition
3. Add reconcileMaster() method that:
   - Only created when spec.master is non-nil and spec.master.enabled is true
   - Creates a Deployment (apps/v1) for master nodes:
     Name: <name>-master
     Replicas: spec.master.replicas
     Ports: 9200 (HTTP) + 9300 (transport)
     Image: same as data nodes (UBI micro mock)
     Command: /bin/sleep infinity (mock)
     Labels: component=master
     No PVC — master nodes don't store data
   - When disabled, deletes the Deployment
   - Sets owner references
   - Records events
4. Generate tests for the new reconciler method
5. Review the code
6. Update the OLM bundle from v0.1.0 to v0.2.0 with replaces"
```

**What this specifically validates**:
- Workflow B conditional Deployment pattern for 4th operand
- Multi-replica Deployment (unlike MongoDB arbiter which is always 1)
- Master Deployment with 2 ports (unlike Redis Sentinel's single port)
- Check-update for Deployment mutable fields (replicas, resources)
- Bundle CRD refresh per Bug #12

**Acceptance criteria**:
- [ ] MasterSpec struct with enabled, replicas, resources fields and markers
- [ ] reconcileMaster() creates Deployment when enabled, deletes when disabled
- [ ] Deployment has correct master ports (9200+9300), labels (component=master)
- [ ] MasterReady condition set/cleared correctly
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
Prompt: "Add admission webhooks and network security to the Elasticsearch 
operator at e2e/elasticsearch-operator/ (which already has dedicated master 
support at v0.2.0).

1. Add webhooks (designing-operator-api Workflow C):
   Defaulting: set replicas=3 when 0, version=8.14 when empty,
   backup.retentionDays=7 when backup.enabled but retentionDays not set,
   master.replicas=3 when master.enabled but replicas not set
   Validating:
   - reject replicas < 1
   - reject master.replicas as even number (quorum requires odd)
   - reject both auth.adminPassword and auth.existingSecret (mutually exclusive)
   - reject storage size reduction on update
   - reject backup.enabled without backup.schedule
2. Generate all webhook config files (service, cert-manager, patches)
3. Update main.go and kustomization files (including kustomize replacements 
   for cert-manager TLS — per Workflow C step 4)
4. Add reconcileNetworkPolicy() — creates NetworkPolicy from 
   networking.k8s.io/v1, allows ports 9200+9300 ingress from same namespace,
   allows DNS egress, always created
5. Add NetworkSecured condition
6. Generate tests for NetworkPolicy + webhook validation
7. Update OLM bundle from v0.2.0 to v0.3.0 with webhook definitions"
```

**What this specifically validates**:
- Webhook generation for 4th API group (search.elasticsearch.example.com)
- cert-manager kustomize replacements (Bug #14 — 4th regression test)
- NetworkPolicy with TWO ingress ports (9200+9300) — like Redis but different ports
- Backup-schedule cross-field validation (reject enabled without schedule)

**Acceptance criteria**:
- [ ] Webhook handler with Default() + ValidateCreate/Update/Delete()
- [ ] 9 webhook config files generated
- [ ] Kustomize build produces correct TLS DNS names (Bug #14 regression)
- [ ] reconcileNetworkPolicy() allows ports 9200+9300
- [ ] Webhook rejects even master replicas, auth mutual exclusion, backup without schedule
- [ ] CSV v0.3.0 with replaces v0.2.0, webhookdefinitions, NP RBAC
- [ ] Code review: 0 Critical, bundle validates
- [ ] No skill/subagent modifications required

---

## Scenario D: API Versioning + ILM Configuration (Workflow D — Version Promotion)

Builds on Scenario C (v0.3.0). Promotes API to v1beta1 with Index Lifecycle Management (ILM) configuration.

```
Prompt: "Promote the Elasticsearch operator API to v1beta1 and add ILM 
configuration at e2e/elasticsearch-operator/ (which already has dedicated 
master + webhooks at v0.3.0).

1. Add API version v1beta1 (designing-operator-api Workflow D):
   Copy types to api/v1beta1/, add +kubebuilder:storageversion,
   add new fields:
   - ilm: *ILMSpec (optional)
     - enabled: bool (default false)
     - hotPhase: string (duration, e.g. '30d')
     - warmPhase: string (duration, e.g. '90d')
     - deletePhase: string (duration, e.g. '365d')
   - maxShards: *int32 (optional, for cluster.max_shards_per_node)
   Add to status: ilmEnabled (bool)
2. Update main.go for v1beta1 scheme registration
3. Only register v1beta1 webhook (remove v1alpha1 webhook file — 
   per Workflow D step 4)
4. Do NOT apply CRD conversion webhook patch (strategy: None)
5. Add v1beta1 webhook validation: if ilm.enabled, hotPhase is required
6. Generate tests for v1beta1 webhook
7. Update OLM bundle from v0.3.0 to v0.4.0 with multi-version CRD,
   maturity alpha→beta, only v1beta1 webhookdefinitions in CSV"
```

**What this specifically validates**:
- API versioning for 4th operator
- CRD conversion strategy: None (Bug #15 — 4th regression test)
- Old webhook file removed (Bug #16 — 4th regression test)
- ILMSpec as a different type of v1beta1 enhancement (duration strings)
- No new reconciler needed — ILM config is API-level

**Acceptance criteria**:
- [ ] api/v1beta1/ with storageversion, ILMSpec, maxShards
- [ ] v1alpha1 webhook file REMOVED (Bug #16)
- [ ] Only v1beta1 webhook registered in main.go
- [ ] CRD conversion strategy: None (Bug #15)
- [ ] CSV webhookdefinitions only contain v1beta1 paths (Bug #16 CSV)
- [ ] CSV v0.4.0 with replaces v0.3.0, multi-version CRD, maturity=beta
- [ ] Code review: 0 Critical, bundle validates
- [ ] No skill/subagent modifications required

---

## Scenario E: Same-Group CRD — ElasticsearchIndex (Scaffolding Workflow B — Expand)

Builds on Scenario D (v0.4.0). Adds `ElasticsearchIndex` CRD in the **same API group** (`search`) as `ElasticsearchCluster`. This is the 2nd same-group multi-CRD test (after Redis Scenario E).

**Why this matters**: Redis E tested same-group multi-CRD with `RedisUser`. Elasticsearch E validates the same pattern on a 2nd operator — confirming it's reproducible, not a one-off.

```
Prompt: "Add an ElasticsearchIndex CRD to the existing Elasticsearch operator 
at e2e/elasticsearch-operator/ (which already has ElasticsearchCluster at 
v0.4.0 with master nodes, webhooks, and ILM support).

The ElasticsearchIndex represents an index template managed by the operator.

1. Scaffold the new CRD (scaffolding-operator Workflow B, Pattern B — 
   same group):
   Kind: ElasticsearchIndex
   Group: search (same as ElasticsearchCluster)
   Domain: elasticsearch.example.com (same)
   Version: v1beta1 (match the storage version)
   
   This should add types to api/v1beta1/ (same package as ElasticsearchCluster)
   and a new controller file. Do NOT create a separate api/ package —
   same-group resources share the api/<version>/ package.

2. Design the ElasticsearchIndex CRD (designing-operator-api Workflow A):
   Spec:
   - indexName: string (required, the Elasticsearch index name)
   - shards: int32 (1-50, default 1)
   - replicas: int32 (0-5, default 1)
   - clusterRef: string (required, name of the ElasticsearchCluster)
   
   Status:
   - phase: Pending/Active/Failed
   - conditions: Available
   - indexReady: bool

3. Implement the ElasticsearchIndex controller (implementing-reconciliation 
   Workflow A for new controller):
   - reconcileIndexConfigMap() — create ConfigMap with index template JSON
   - Register the new controller in main.go with its own Reconcile loop
   - Add RBAC for the new CRD

4. Generate tests for the ElasticsearchIndex controller
5. Review the complete project
6. Update the OLM bundle from v0.4.0 to v0.5.0:
   - Add ElasticsearchIndex as a second owned CRD in the CSV
   - Same API group as ElasticsearchCluster"
```

**What this specifically validates**:
- `scaffolding-operator` Workflow B (Pattern B: same-group) — 2nd validation (after Redis E)
- Two kinds in same api/v1beta1/ package
- Two controllers registered in main.go from same package
- CSV with two owned CRDs from same group (vs MongoDB E which was different group)
- Cross-CRD reference: ElasticsearchIndex.spec.clusterRef → ElasticsearchCluster

**Acceptance criteria**:
- [ ] ElasticsearchIndex types added to api/v1beta1/ (same package as ElasticsearchCluster)
- [ ] New controller file for ElasticsearchIndex with its own Reconcile()
- [ ] main.go registers BOTH controllers
- [ ] ElasticsearchIndex RBAC markers separate from ElasticsearchCluster
- [ ] reconcileIndexConfigMap() follows check-create pattern
- [ ] Tests: generate-test-matrix shows coverage for ElasticsearchIndex methods
- [ ] CSV v0.5.0 with TWO owned CRDs from SAME group
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

| Bug # | Description | Tested In |
|-------|-------------|-----------|
| 2 | Dockerfile cross-compile | A (image build) |
| 3 | OpenShift image compatibility | A (UBI micro mock) |
| 5 | UBI base image | A (Dockerfile) |
| 10 | manager.yaml namespace | A (scaffold) |
| 11 | Check-update for modified reconcilers | B (master adds to reconciler) |
| 12 | Bundle CRD refresh | B (v0.2.0 bundle) |
| 13 | Audit ALL mutable fields | B (Deployment check-update) |
| 14 | cert-manager kustomize replacements | C (webhook TLS) |
| 15 | CRD conversion strategy: None | D (multi-version CRD) |
| 16 | Multi-version webhook stripping + CSV | D (v1beta1 only webhook) |
| 17 | Third-party image env vars | A (ES image) |
| 18 | check-idempotency.py List() support | N/A (no Job) |

## Version Chain

| Scenario | Bundle | Replaces | Maturity | API Versions | CRDs |
|----------|--------|----------|----------|-------------|------|
| A | 0.1.0 | — | alpha | v1alpha1 | ElasticsearchCluster |
| B | 0.2.0 | 0.1.0 | alpha | v1alpha1 | ElasticsearchCluster |
| C | 0.3.0 | 0.2.0 | alpha | v1alpha1 | ElasticsearchCluster |
| D | 0.4.0 | 0.3.0 | beta | v1alpha1 + v1beta1 | ElasticsearchCluster |
| E | 0.5.0 | 0.4.0 | beta | v1beta1 | ElasticsearchCluster + ElasticsearchIndex |

## Cumulative E2E Statistics (After Elasticsearch)

| Operator | Scenarios | Tests | Skill Fixes | Deploy Paths |
|----------|-----------|-------|-------------|--------------|
| PostgreSQL | A-D | 111 | 17 | Both |
| Redis | A-E | 139 | 0 | Both |
| MongoDB | A-E | 150 | 1 (Bug #18) | Both |
| Elasticsearch | A-E | ~150 (est) | 0 (target) | Both |
| **Total** | **19** | **~550** | **18** | **Both** |
