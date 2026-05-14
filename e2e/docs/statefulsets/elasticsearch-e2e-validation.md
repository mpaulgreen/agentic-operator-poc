# E2E Validation Guide: Elasticsearch Operator on OpenShift

End-to-end validation of the Elasticsearch operator on a live OpenShift cluster. This is the **N=4 generality proof** — the 4th stateful workload operator validating that skills produce correct code without any further modifications, after PostgreSQL (111 tests, 17 fixes), Redis (139 tests, 0 fixes), and MongoDB (150 tests, 1 fix).

| Scenario | Feature | Bundle Version | Skills Tested | New Resource |
|----------|---------|----------------|---------------|-------------|
| A | Core operator (data nodes) | v0.1.0 | All 5 + all 3 subagents | StatefulSet, Service×2 (HTTP+transport), Secret, ConfigMap, CronJob |
| B | Dedicated master nodes | v0.2.0 | 4 (Workflow B) + 3 subagents | Deployment (master) |
| C | Webhooks + Network Security | v0.3.0 | 4 (Workflow C) + 3 subagents | NetworkPolicy |
| D | API Maturity + ILM Config | v0.4.0 | 4 (Workflow D) + 3 subagents | — (ILMSpec) |
| E | Same-group CRD (ElasticsearchIndex) | v0.5.0 | scaffolding B + all | ElasticsearchIndex CRD |

**Run scenarios in order** — each builds on the previous.

---

# Scenario A: Core Operator — Data Nodes (v0.1.0)

The operator was built using the mandatory workflow from CLAUDE.md:
- **Step 1** (Generate): `scaffolding-operator` SKILL (Workflow A)
- **Step 2** (Generate): `designing-operator-api` SKILL (Workflow A)
- **Step 3** (Generate): `implementing-reconciliation` SKILL (Workflow A)
- **Step 4a** (Test): `operator-test-generator` SUBAGENT
- **Step 4b** (Review): `operator-reviewer` SUBAGENT
- **Step 5** (Generate): `bundling-operator` SKILL (Workflow A)
- **Step 6** (Validate): `operator-bundle-validator` SUBAGENT

**Project stats**: 6 reconciler methods, 4 conditions (Available, Progressing, Degraded, BackupReady), 6 managed resources (Secret, ConfigMap, Service×2, StatefulSet, CronJob), 9 RBAC markers, all validation scripts pass.

**Key differences from PostgreSQL/Redis/MongoDB**: Two Services with different purposes (HTTP API on 9200 + transport/inter-node on 9300), two-port StatefulSet, CronJob backup with schedule (like PostgreSQL but with schedule update), elasticsearch.yml YAML config. Uses UBI micro mock container for E2E.

**Zero skill modifications required** — confirming N=4 generality.

## Prerequisites

- OpenShift 4.14+ cluster with cluster-admin access
- `oc` CLI logged in (`oc whoami` returns a user)
- `podman` for building images
- Access to a container registry (quay.io)
- A default StorageClass available (`oc get sc`)

## Environment Setup

```bash
export IMG=quay.io/mpaulgreen/elasticsearch-operator:v0.1.0
export BUNDLE_IMG=quay.io/mpaulgreen/elasticsearch-operator-bundle:v0.1.0
export NAMESPACE=elasticsearch-operator-system

cd e2e/elasticsearch-operator
```

---

## Phase 1: Build and Deploy

### 1.1 Build the Operator Image

```bash
podman build --platform linux/amd64 -t $IMG .
podman push $IMG
```

**Expected**: Image builds and pushes successfully.

### 1.2 Deploy the Operator

#### Option A: `make deploy` (Development)

```bash
make manifests
make deploy IMG=$IMG
```

#### Option B: OLM

```bash
# Update CSV image reference
sed -i '' "s|quay.io/mpaulgreen/elasticsearch-operator:v0.1.0|$IMG|g" bundle/manifests/elasticsearch-operator.clusterserviceversion.yaml

# Build and push bundle
podman build -t $BUNDLE_IMG -f bundle.Dockerfile .
podman push $BUNDLE_IMG

# Create namespace first
oc new-project $NAMESPACE || oc create namespace $NAMESPACE

# Deploy via OLM
operator-sdk run bundle $BUNDLE_IMG --namespace $NAMESPACE --timeout 5m
```

### 1.3 Verify Deployment

```bash
oc get pods -n $NAMESPACE -l control-plane=controller-manager
oc logs -n $NAMESPACE -l control-plane=controller-manager --tail=20

# CRD installed
oc get crd elasticsearchclusters.search.elasticsearch.example.com

# CRD has expected fields
oc get crd elasticsearchclusters.search.elasticsearch.example.com -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties}' | python3 -c "import json,sys; print(sorted(json.load(sys.stdin).keys()))"

# Controller watching EventSources
oc logs -n $NAMESPACE -l control-plane=controller-manager --tail=20 | grep -E "Starting EventSource|Starting workers"
```

**Expected**:
- [ ] Pod 1/1 Running
- [ ] CRD `elasticsearchclusters.search.elasticsearch.example.com` installed
- [ ] CRD fields: auth, backup, replicas, resources, storage, version
- [ ] Controller watching EventSources for Secret, ConfigMap, Service, StatefulSet, CronJob

---

## Phase 2: Basic CR Lifecycle

### 2.1 Create a Minimal ElasticsearchCluster

```bash
cat <<EOF | oc apply -f -
apiVersion: search.elasticsearch.example.com/v1alpha1
kind: ElasticsearchCluster
metadata:
  name: es-test
  namespace: $NAMESPACE
spec:
  replicas: 3
  version: "8.14"
  storage:
    size: 1Gi
EOF

sleep 30
```

### 2.2 Verify All 6 Managed Resources Created

```bash
echo "=== Managed Resources ==="
oc get secret es-test-auth -n $NAMESPACE && echo "PASS: Auth Secret" || echo "FAIL: Auth Secret"
oc get configmap es-test-config -n $NAMESPACE && echo "PASS: ConfigMap" || echo "FAIL: ConfigMap"
oc get service es-test-http -n $NAMESPACE && echo "PASS: HTTP Service" || echo "FAIL: HTTP Service"
oc get service es-test-transport -n $NAMESPACE && echo "PASS: Transport Service" || echo "FAIL: Transport Service"
oc get statefulset es-test -n $NAMESPACE && echo "PASS: StatefulSet" || echo "FAIL: StatefulSet"

echo ""
echo "=== No Backup CronJob (backup not enabled) ==="
oc get cronjob es-test-backup -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: No backup CronJob" || echo "FAIL: CronJob exists"

echo ""
echo "=== Owner References ==="
oc get secret es-test-auth -n $NAMESPACE -o jsonpath='{.metadata.ownerReferences[0].kind}' && echo " (should be ElasticsearchCluster)"
oc get service es-test-http -n $NAMESPACE -o jsonpath='{.metadata.ownerReferences[0].kind}' && echo " (should be ElasticsearchCluster)"
oc get statefulset es-test -n $NAMESPACE -o jsonpath='{.metadata.ownerReferences[0].kind}' && echo " (should be ElasticsearchCluster)"

echo ""
echo "=== CR Status ==="
oc get elasticsearchcluster es-test -n $NAMESPACE -o wide
```

**Expected**:
- [ ] Secret `es-test-auth` created with `ELASTIC_USERNAME` and `ELASTIC_PASSWORD` keys
- [ ] ConfigMap `es-test-config` created with `elasticsearch.yml`
- [ ] HTTP Service `es-test-http` (ClusterIP, port 9200)
- [ ] Transport Service `es-test-transport` (ClusterIP: None, port 9300)
- [ ] StatefulSet `es-test` created with 3 replicas
- [ ] No backup CronJob (backup not enabled)
- [ ] All resources have ownerReferences pointing to ElasticsearchCluster

### 2.3 Verify Two Services (Elasticsearch-Specific)

```bash
echo "=== HTTP Service ==="
oc get service es-test-http -n $NAMESPACE -o jsonpath='{.spec.clusterIP}' && echo " (should NOT be None)"
oc get service es-test-http -n $NAMESPACE -o jsonpath='{.spec.ports[0].port}' && echo " port (should be 9200)"

echo ""
echo "=== Transport Service ==="
oc get service es-test-transport -n $NAMESPACE -o jsonpath='{.spec.clusterIP}' && echo " (should be None — headless)"
oc get service es-test-transport -n $NAMESPACE -o jsonpath='{.spec.ports[0].port}' && echo " port (should be 9300)"
```

**Expected**:
- [ ] HTTP service has a real ClusterIP (not None), port 9200
- [ ] Transport service has `clusterIP: None` (headless), port 9300

### 2.4 Verify StatefulSet Details

```bash
echo "=== StatefulSet Spec ==="
oc get statefulset es-test -n $NAMESPACE -o jsonpath='{.spec.replicas}' && echo " replicas"
oc get statefulset es-test -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}' && echo " image"
oc get statefulset es-test -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].ports[*].containerPort}' && echo " ports (should be 9200 9300)"
oc get statefulset es-test -n $NAMESPACE -o jsonpath='{.spec.volumeClaimTemplates[0].spec.resources.requests.storage}' && echo " storage"

echo ""
echo "=== Pod Status ==="
oc get pods -n $NAMESPACE -l app.kubernetes.io/instance=es-test
```

**Expected**:
- [ ] 3 replicas
- [ ] Image: `registry.access.redhat.com/ubi9/ubi-micro:latest` (E2E mock)
- [ ] Two container ports: 9200 + 9300
- [ ] PVC requests 1Gi storage

### 2.5 Verify Auth Secret Content

```bash
oc get secret es-test-auth -n $NAMESPACE -o jsonpath='{.data.ELASTIC_USERNAME}' | base64 -d && echo ""
oc get secret es-test-auth -n $NAMESPACE -o jsonpath='{.data.ELASTIC_PASSWORD}' | base64 -d && echo ""
```

**Expected**: Username=elastic, Password is random (24 chars).

### 2.6 Verify ConfigMap Content

```bash
oc get configmap es-test-config -n $NAMESPACE -o jsonpath='{.data.elasticsearch\.yml}'
```

**Expected**: Contains `cluster.name: es-test`, `http.port: 9200`, `transport.port: 9300`, `discovery.seed_hosts`.

### 2.7 Wait for Running Phase

```bash
oc wait --for=condition=ready pod -l app.kubernetes.io/instance=es-test -n $NAMESPACE --timeout=300s

oc get elasticsearchcluster es-test -n $NAMESPACE -o jsonpath='{.status}' | python3 -m json.tool
```

**Expected**:
- [ ] `phase: Running`
- [ ] `readyReplicas: 3`
- [ ] `currentVersion: "8.14"`
- [ ] `httpEndpoint: es-test-http.$NAMESPACE.svc.cluster.local:9200`

### 2.8 Verify Conditions

```bash
oc get elasticsearchcluster es-test -n $NAMESPACE -o jsonpath='{.status.conditions}' | python3 -m json.tool
```

**Expected conditions**:
- [ ] `Available: True`
- [ ] `Progressing: False`
- [ ] `Degraded: False`
- [ ] `BackupReady: False` (backup not enabled)

### 2.9 Verify Print Columns

```bash
oc get elasticsearchcluster -n $NAMESPACE
```

**Expected**: Table with columns Phase, Ready, Version, Age.

### 2.10 Verify Events

```bash
oc get events -n $NAMESPACE --field-selector involvedObject.name=es-test --sort-by='.lastTimestamp'
```

**Expected**: Events for SecretCreated, ConfigMapCreated, HTTPServiceCreated, TransportServiceCreated, StatefulSetCreated.

---

## Phase 3: Idempotency

### 3.1 Verify No Duplicate Resources

```bash
oc delete pod -n $NAMESPACE -l control-plane=controller-manager
oc wait --for=condition=available deployment -l control-plane=controller-manager -n $NAMESPACE --timeout=60s
sleep 15

echo "Auth Secrets: $(oc get secret -n $NAMESPACE 2>&1 | grep -c es-test-auth) (should be 1)"
echo "ConfigMaps: $(oc get configmap -n $NAMESPACE 2>&1 | grep -c es-test-config) (should be 1)"
echo "HTTP Services: $(oc get service -n $NAMESPACE 2>&1 | grep -c es-test-http) (should be 1)"
echo "Transport Services: $(oc get service -n $NAMESPACE 2>&1 | grep -c es-test-transport) (should be 1)"
echo "StatefulSets: $(oc get statefulset -n $NAMESPACE 2>&1 | grep -c es-test) (should be 1)"
```

**Expected**: Exactly 1 of each resource, no duplicates.

### 3.2 Verify Password Unchanged After Reconciliation

```bash
PASS_BEFORE=$(oc get secret es-test-auth -n $NAMESPACE -o jsonpath='{.data.ELASTIC_PASSWORD}')

oc label elasticsearchcluster es-test -n $NAMESPACE test-reconcile=true
sleep 10

PASS_AFTER=$(oc get secret es-test-auth -n $NAMESPACE -o jsonpath='{.data.ELASTIC_PASSWORD}')
[ "$PASS_BEFORE" = "$PASS_AFTER" ] && echo "PASS: Password unchanged (idempotent)" || echo "FAIL: Password changed!"
```

---

## Phase 4: Scaling

### 4.1 Scale Up

```bash
oc patch elasticsearchcluster es-test -n $NAMESPACE --type merge -p '{"spec":{"replicas":5}}'
sleep 15

oc get statefulset es-test -n $NAMESPACE -o jsonpath='{.spec.replicas}' && echo " replicas (should be 5)"
```

**Expected**: StatefulSet replicas updated to 5.

### 4.2 Scale Down

```bash
oc patch elasticsearchcluster es-test -n $NAMESPACE --type merge -p '{"spec":{"replicas":1}}'
sleep 15

oc get statefulset es-test -n $NAMESPACE -o jsonpath='{.spec.replicas}' && echo " replicas (should be 1)"
```

**Expected**: Replicas updated to 1.

### 4.3 Scale Back to 3

```bash
oc patch elasticsearchcluster es-test -n $NAMESPACE --type merge -p '{"spec":{"replicas":3}}'
sleep 15

oc get statefulset es-test -n $NAMESPACE -o jsonpath='{.spec.replicas}' && echo " replicas (should be 3)"
```

---

## Phase 5: Backup CronJob

### 5.1 Enable Backup

```bash
oc patch elasticsearchcluster es-test -n $NAMESPACE --type merge -p '{"spec":{"backup":{"enabled":true,"schedule":"0 2 * * *","retentionDays":7}}}'
sleep 15
```

### 5.2 Verify Backup CronJob Created

```bash
echo "=== Backup CronJob ==="
oc get cronjob es-test-backup -n $NAMESPACE
oc get cronjob es-test-backup -n $NAMESPACE -o jsonpath='{.spec.schedule}' && echo " schedule (should be 0 2 * * *)"
oc get cronjob es-test-backup -n $NAMESPACE -o jsonpath='{.metadata.ownerReferences[0].kind}' && echo " owner (should be ElasticsearchCluster)"
```

**Expected**:
- [ ] CronJob `es-test-backup` created with schedule `0 2 * * *`
- [ ] Owner reference → ElasticsearchCluster

### 5.3 Verify BackupReady Condition

```bash
oc get elasticsearchcluster es-test -n $NAMESPACE -o jsonpath='{.status.conditions}' | python3 -c "
import json, sys
conditions = json.load(sys.stdin)
for c in conditions:
    if c['type'] == 'BackupReady':
        print(f\"BackupReady: {c['status']} (reason: {c['reason']})\")
"
```

**Expected**: BackupReady: True (BackupConfigured).

### 5.4 Verify Schedule Update

```bash
oc patch elasticsearchcluster es-test -n $NAMESPACE --type merge -p '{"spec":{"backup":{"schedule":"0 4 * * *"}}}'
sleep 15

oc get cronjob es-test-backup -n $NAMESPACE -o jsonpath='{.spec.schedule}' && echo " (should be 0 4 * * *)"
```

**Expected**: CronJob schedule updated to `0 4 * * *` (check-update pattern).

### 5.5 Disable Backup

```bash
oc patch elasticsearchcluster es-test -n $NAMESPACE --type merge -p '{"spec":{"backup":{"enabled":false}}}'
sleep 15

echo "=== BackupReady After Disable ==="
oc get elasticsearchcluster es-test -n $NAMESPACE -o jsonpath='{.status.conditions}' | python3 -c "
import json, sys
conditions = json.load(sys.stdin)
for c in conditions:
    if c['type'] == 'BackupReady':
        print(f\"BackupReady: {c['status']} (reason: {c['reason']})\")
"
```

**Expected**: BackupReady: False (BackupDisabled).

---

## Phase 6: Finalizer and Deletion

### 6.1 Verify Finalizer Exists

```bash
oc get elasticsearchcluster es-test -n $NAMESPACE -o jsonpath='{.metadata.finalizers}' && echo ""
```

**Expected**: `["search.elasticsearch.example.com/finalizer"]`

### 6.2 Delete the CR

```bash
oc delete elasticsearchcluster es-test -n $NAMESPACE
sleep 15

oc get secret es-test-auth -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Auth Secret cleaned"
oc get configmap es-test-config -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: ConfigMap cleaned"
oc get service es-test-http -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: HTTP Service cleaned"
oc get service es-test-transport -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Transport Service cleaned"
oc get statefulset es-test -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: StatefulSet cleaned"
oc get cronjob es-test-backup -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Backup CronJob cleaned"
```

**Expected**:
- [ ] All 6 managed resources garbage collected
- [ ] No orphaned resources remain

---

## Phase 7: Validation Markers

### 7.1 Reject Invalid Replicas

```bash
cat <<EOF | oc apply -f - 2>&1
apiVersion: search.elasticsearch.example.com/v1alpha1
kind: ElasticsearchCluster
metadata:
  name: es-invalid
  namespace: $NAMESPACE
spec:
  replicas: 15
  version: "8.14"
  storage:
    size: 1Gi
EOF
```

**Expected**: Rejected — `replicas` max is 9.

### 7.2 Reject Invalid Version

```bash
cat <<EOF | oc apply -f - 2>&1
apiVersion: search.elasticsearch.example.com/v1alpha1
kind: ElasticsearchCluster
metadata:
  name: es-invalid
  namespace: $NAMESPACE
spec:
  replicas: 3
  version: "7.17"
  storage:
    size: 1Gi
EOF
```

**Expected**: Rejected — version enum only allows "8.12", "8.14".

### 7.3 Reject Invalid Storage Size Pattern

```bash
cat <<EOF | oc apply -f - 2>&1
apiVersion: search.elasticsearch.example.com/v1alpha1
kind: ElasticsearchCluster
metadata:
  name: es-invalid
  namespace: $NAMESPACE
spec:
  replicas: 3
  version: "8.14"
  storage:
    size: "100MB"
EOF
```

**Expected**: Rejected — storage size must match pattern `^[0-9]+[KMGT]i$`.

### 7.4 Verify Defaults Applied

```bash
cat <<EOF | oc apply -f -
apiVersion: search.elasticsearch.example.com/v1alpha1
kind: ElasticsearchCluster
metadata:
  name: es-defaults
  namespace: $NAMESPACE
spec:
  storage:
    size: 5Gi
EOF

sleep 5
oc get elasticsearchcluster es-defaults -n $NAMESPACE -o jsonpath='{.spec.replicas}' && echo " (should be 3)"
oc get elasticsearchcluster es-defaults -n $NAMESPACE -o jsonpath='{.spec.version}' && echo " (should be 8.14)"

oc delete elasticsearchcluster es-defaults -n $NAMESPACE
```

**Expected**: Defaults applied — replicas=3, version="8.14".

---

## Phase 8: RBAC Verification

### 8.1 Verify Operator RBAC Works

```bash
oc auth can-i get secrets --as=system:serviceaccount:$NAMESPACE:elasticsearch-operator-controller-manager && echo "PASS" || echo "FAIL"
oc auth can-i create statefulsets --as=system:serviceaccount:$NAMESPACE:elasticsearch-operator-controller-manager && echo "PASS" || echo "FAIL"
oc auth can-i create cronjobs --as=system:serviceaccount:$NAMESPACE:elasticsearch-operator-controller-manager && echo "PASS: CronJobs RBAC" || echo "FAIL"
oc auth can-i update elasticsearchclusters/status --as=system:serviceaccount:$NAMESPACE:elasticsearch-operator-controller-manager && echo "PASS" || echo "FAIL"
```

### 8.2 Verify No Excess Permissions

```bash
oc auth can-i create namespaces --as=system:serviceaccount:$NAMESPACE:elasticsearch-operator-controller-manager && echo "FAIL: excess perms" || echo "PASS: no excess"
oc auth can-i delete nodes --as=system:serviceaccount:$NAMESPACE:elasticsearch-operator-controller-manager && echo "FAIL: excess perms" || echo "PASS: no excess"
```

---

## Phase 9: Security Posture

### 9.1 Verify Non-Root Container

```bash
oc get deployment elasticsearch-operator-controller-manager -n $NAMESPACE -o jsonpath='{.spec.template.spec.securityContext.runAsNonRoot}' && echo " (should be true)"
```

### 9.2 Verify Capabilities Dropped

```bash
oc get deployment elasticsearch-operator-controller-manager -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].securityContext.capabilities.drop}' && echo ""
```

**Expected**: `["ALL"]`

---

## Phase 10: Health Probes

### 10.1 Verify Liveness Probe

```bash
oc get deployment elasticsearch-operator-controller-manager -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}' | python3 -m json.tool
```

**Expected**: httpGet on /healthz:8081

### 10.2 Verify Readiness Probe

```bash
oc get deployment elasticsearch-operator-controller-manager -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].readinessProbe}' | python3 -m json.tool
```

**Expected**: httpGet on /readyz:8081

---

## Phase 11: OLM Bundle Validation (Optional)

### 11.1 Bundle Validate

```bash
operator-sdk bundle validate bundle/
```

**Expected**: No errors.

---

## Phase 12: Multi-Instance

### 12.1 Create Multiple ElasticsearchClusters

```bash
for i in 1 2 3; do
cat <<EOF | oc apply -f -
apiVersion: search.elasticsearch.example.com/v1alpha1
kind: ElasticsearchCluster
metadata:
  name: es-multi-$i
  namespace: $NAMESPACE
spec:
  replicas: 1
  version: "8.14"
  storage:
    size: 1Gi
EOF
done

sleep 30
oc get elasticsearchcluster -n $NAMESPACE
```

**Expected**:
- [ ] 3 independent ElasticsearchClusters created
- [ ] Each has its own Secret, ConfigMap, 2 Services, StatefulSet
- [ ] No cross-contamination

### 12.2 Delete One, Others Unaffected

```bash
oc delete elasticsearchcluster es-multi-2 -n $NAMESPACE
sleep 10

oc get elasticsearchcluster -n $NAMESPACE
oc get statefulset -n $NAMESPACE
```

**Expected**: es-multi-1 and es-multi-3 still Running, es-multi-2 fully cleaned up.

---

## Scenario A Cleanup

```bash
oc delete elasticsearchcluster --all -n $NAMESPACE
sleep 15

# If NOT continuing to Scenario B, undeploy the operator:
# make undeploy                                                    # if deployed with make deploy
# operator-sdk cleanup elasticsearch-operator --namespace $NAMESPACE  # if deployed with OLM
# oc delete project $NAMESPACE
```

---

## Scenario A Summary Checklist

| # | Test | Phase | Expected |
|---|------|-------|----------|
| 1 | Operator deploys and starts | 1 | Pod Running |
| 2 | CRD installed with expected fields | 1 | auth, backup, replicas, resources, storage, version |
| 3 | CR creates all 6 managed resources | 2.2 | Secret, ConfigMap, Service×2, StatefulSet, no CronJob |
| 4 | Owner references set on all resources | 2.2 | All point to ElasticsearchCluster |
| 5 | HTTP service has real ClusterIP, port 9200 | 2.3 | Not None |
| 6 | Transport service is headless, port 9300 | 2.3 | clusterIP: None |
| 7 | StatefulSet has correct image and 2 ports | 2.4 | UBI micro, 9200+9300 |
| 8 | Auth Secret has username + password | 2.5 | elastic, 24-char random |
| 9 | ConfigMap has elasticsearch.yml | 2.6 | cluster.name, http.port, transport.port |
| 10 | Status reaches Running phase | 2.7 | phase=Running, readyReplicas=3 |
| 11 | HttpEndpoint points to HTTP service | 2.7 | es-test-http...9200 |
| 12 | Conditions set correctly | 2.8 | Available=True, BackupReady=False |
| 13 | Print columns display | 2.9 | Phase, Ready, Version, Age |
| 14 | Events recorded for each resource | 2.10 | 5 create events |
| 15 | Idempotent — no duplicates | 3.1 | Exactly 1 of each |
| 16 | Password unchanged on re-reconcile | 3.2 | Same base64 value |
| 17 | Scale up works | 4.1 | 5 replicas |
| 18 | Scale down works | 4.2 | 1 replica |
| 19 | Backup CronJob created when enabled | 5.2 | schedule=0 2 * * *, ownerRef |
| 20 | BackupReady condition set | 5.3 | BackupConfigured |
| 21 | CronJob schedule updated | 5.4 | 0 4 * * * (check-update) |
| 22 | BackupReady False when disabled | 5.5 | BackupDisabled |
| 23 | Finalizer present | 6.1 | Finalizer in metadata |
| 24 | Deletion cleans all 6 resources | 6.2 | No orphans |
| 25 | Invalid replicas rejected | 7.1 | Validation error |
| 26 | Invalid version rejected | 7.2 | Validation error |
| 27 | Invalid storage pattern rejected | 7.3 | Validation error |
| 28 | Defaults applied | 7.4 | replicas=3, version=8.14 |
| 29 | RBAC allows needed operations (incl. batch/cronjobs) | 8.1 | can-i returns yes |
| 30 | No excess RBAC permissions | 8.2 | can-i returns no |
| 31 | Container runs as non-root | 9.1 | runAsNonRoot=true |
| 32 | Capabilities dropped | 9.2 | drop=[ALL] |
| 33 | Health probes configured | 10 | /healthz, /readyz |
| 34 | Bundle validates | 11 | No errors |
| 35 | Multiple instances independent | 12.1 | No cross-contamination |
| 36 | Deleting one doesn't affect others | 12.2 | Others still Running |
