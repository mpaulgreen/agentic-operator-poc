# E2E Validation Guide: Redis Operator on OpenShift

End-to-end validation of the Redis operator on a live OpenShift cluster. This is the **generality test** — validating that the skills produce correct operator code for a non-PostgreSQL workload without any skill modifications.

| Scenario | Feature | Bundle Version | Skills Tested | New Resource |
|----------|---------|----------------|---------------|-------------|
| A | Core operator (from scratch) | v0.1.0 | All 5 + all 3 subagents | StatefulSet, Service×2, Secret, ConfigMap, PDB |
| B | Redis Sentinel HA | v0.2.0 | 4 (Workflow B) + 3 subagents | Deployment + Service (sentinel) |
| C | Webhooks + Network Security | v0.3.0 | 4 (Workflow C) + 3 subagents | NetworkPolicy |
| D | API Maturity + TLS | v0.4.0 | 4 (Workflow D) + 3 subagents | — (TLSSpec) |
| E | Add Second CRD (RedisUser) | v0.5.0 | scaffolding B + all | RedisUser CRD + controller |

**Run scenarios in order** — each builds on the previous.

---

# Scenario A: Core Operator (v0.1.0)

The operator was built using the mandatory workflow from CLAUDE.md:
- **Step 1** (Generate): `scaffolding-operator` SKILL (Workflow A)
- **Step 2** (Generate): `designing-operator-api` SKILL (Workflow A)
- **Step 3** (Generate): `implementing-reconciliation` SKILL (Workflow A)
- **Step 4a** (Test): `operator-test-generator` SUBAGENT
- **Step 4b** (Review): `operator-reviewer` SUBAGENT
- **Step 5** (Generate): `bundling-operator` SKILL (Workflow A)
- **Step 6** (Validate): `operator-bundle-validator` SUBAGENT

**Project stats**: 6 reconciler methods, 3 conditions, 6 owned resources (Secret, ConfigMap, Service×2, StatefulSet, PDB), 9 RBAC markers, all validation scripts pass.

**Key differences from PostgreSQL**: Two Services (headless + client), auto-PDB when replicas > 1, auth Secret with REDIS_PASSWORD, redis.conf ConfigMap, port 6379.

## Prerequisites

- OpenShift 4.14+ cluster with cluster-admin access
- `oc` CLI logged in (`oc whoami` returns a user)
- `podman` for building images
- Access to a container registry (quay.io)
- A default StorageClass available (`oc get sc`)

## Environment Setup

```bash
export IMG=quay.io/mpaulgreen/redis-operator:v0.1.0
export BUNDLE_IMG=quay.io/mpaulgreen/redis-operator-bundle:v0.1.0
export NAMESPACE=redis-operator-system

cd e2e/redis-operator
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
sed -i '' "s|quay.io/mpaulgreen/redis-operator:v0.1.0|$IMG|g" bundle/manifests/redis-operator.clusterserviceversion.yaml

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
oc get crd redisclusters.cache.redis.example.com

# CRD has expected fields
oc get crd redisclusters.cache.redis.example.com -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties}' | python3 -c "import json,sys; print(sorted(json.load(sys.stdin).keys()))"
```

**Expected**:
- [ ] Pod 1/1 Running
- [ ] CRD `redisclusters.cache.redis.example.com` installed
- [ ] CRD fields: auth, replicas, resources, storage, version
- [ ] Controller watching EventSources for Secret, ConfigMap, Service, StatefulSet, PDB

---

## Phase 2: Basic CR Lifecycle

### 2.1 Create a Minimal RedisCluster

```bash
cat <<EOF | oc apply -f -
apiVersion: cache.redis.example.com/v1alpha1
kind: RedisCluster
metadata:
  name: redis-test
  namespace: $NAMESPACE
spec:
  replicas: 3
  version: "7.4"
  storage:
    size: 1Gi
EOF

sleep 30
```

### 2.2 Verify All 6 Managed Resources Created

```bash
echo "=== Managed Resources ==="
oc get secret redis-test-auth -n $NAMESPACE && echo "PASS: Secret" || echo "FAIL: Secret"
oc get configmap redis-test-config -n $NAMESPACE && echo "PASS: ConfigMap" || echo "FAIL: ConfigMap"
oc get service redis-test-headless -n $NAMESPACE && echo "PASS: Headless Service" || echo "FAIL: Headless Service"
oc get service redis-test-client -n $NAMESPACE && echo "PASS: Client Service" || echo "FAIL: Client Service"
oc get statefulset redis-test -n $NAMESPACE && echo "PASS: StatefulSet" || echo "FAIL: StatefulSet"
oc get pdb redis-test-pdb -n $NAMESPACE && echo "PASS: PDB (replicas > 1)" || echo "FAIL: PDB"

echo ""
echo "=== Owner References ==="
oc get secret redis-test-auth -n $NAMESPACE -o jsonpath='{.metadata.ownerReferences[0].kind}' && echo " (should be RedisCluster)"
oc get service redis-test-client -n $NAMESPACE -o jsonpath='{.metadata.ownerReferences[0].kind}' && echo " (should be RedisCluster)"
oc get statefulset redis-test -n $NAMESPACE -o jsonpath='{.metadata.ownerReferences[0].kind}' && echo " (should be RedisCluster)"
oc get pdb redis-test-pdb -n $NAMESPACE -o jsonpath='{.metadata.ownerReferences[0].kind}' && echo " (should be RedisCluster)"

echo ""
echo "=== CR Status ==="
oc get rediscluster redis-test -n $NAMESPACE -o wide
```

**Expected**:
- [ ] Secret `redis-test-auth` created with `REDIS_PASSWORD` key
- [ ] ConfigMap `redis-test-config` created with `redis.conf`
- [ ] Headless Service `redis-test-headless` (ClusterIP: None, port 6379)
- [ ] Client Service `redis-test-client` (ClusterIP, port 6379)
- [ ] StatefulSet `redis-test` created with 3 replicas
- [ ] PDB `redis-test-pdb` created (because replicas=3 > 1)
- [ ] All resources have ownerReferences pointing to RedisCluster

### 2.3 Verify Two Services (Redis-Specific)

```bash
echo "=== Headless Service ==="
oc get service redis-test-headless -n $NAMESPACE -o jsonpath='{.spec.clusterIP}' && echo " (should be None)"
oc get service redis-test-headless -n $NAMESPACE -o jsonpath='{.spec.ports[0].port}' && echo " port (should be 6379)"

echo ""
echo "=== Client Service ==="
oc get service redis-test-client -n $NAMESPACE -o jsonpath='{.spec.clusterIP}' && echo " (should NOT be None)"
oc get service redis-test-client -n $NAMESPACE -o jsonpath='{.spec.ports[0].port}' && echo " port (should be 6379)"
```

**Expected**:
- [ ] Headless service has `clusterIP: None`
- [ ] Client service has a real ClusterIP (not None)
- [ ] Both on port 6379

### 2.4 Verify StatefulSet Details

```bash
echo "=== StatefulSet Spec ==="
oc get statefulset redis-test -n $NAMESPACE -o jsonpath='{.spec.replicas}' && echo " replicas"
oc get statefulset redis-test -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}' && echo " image"
oc get statefulset redis-test -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].ports[0].containerPort}' && echo " port"
oc get statefulset redis-test -n $NAMESPACE -o jsonpath='{.spec.volumeClaimTemplates[0].spec.resources.requests.storage}' && echo " storage"

echo ""
echo "=== Pod Status ==="
oc get pods -n $NAMESPACE -l app.kubernetes.io/instance=redis-test
```

**Expected**:
- [ ] 3 replicas
- [ ] Image contains `redis`
- [ ] Container port 6379
- [ ] PVC requests 1Gi storage

### 2.5 Verify ConfigMap Content

```bash
oc get configmap redis-test-config -n $NAMESPACE -o jsonpath='{.data.redis\.conf}'
```

**Expected**: Contains `bind 0.0.0.0`, `port 6379`, `maxmemory-policy allkeys-lru`, `appendonly yes`.

### 2.6 Verify Secret Content

```bash
oc get secret redis-test-auth -n $NAMESPACE -o jsonpath='{.data.REDIS_PASSWORD}' | base64 -d && echo ""
```

**Expected**: Random password generated (24 chars).

### 2.7 Verify PDB (Auto-Created When Replicas > 1)

```bash
oc get pdb redis-test-pdb -n $NAMESPACE
oc get pdb redis-test-pdb -n $NAMESPACE -o jsonpath='{.spec.minAvailable}' && echo " minAvailable (should be replicas-1 = 2)"
```

**Expected**:
- [ ] PDB exists with minAvailable = 2 (replicas - 1)

### 2.8 Wait for Running Phase

```bash
oc wait --for=condition=ready pod -l app.kubernetes.io/instance=redis-test -n $NAMESPACE --timeout=300s

oc get rediscluster redis-test -n $NAMESPACE -o jsonpath='{.status}' | python3 -m json.tool
```

**Expected**:
- [ ] `phase: Running`
- [ ] `readyReplicas: 3`
- [ ] `currentVersion: "7.4"`
- [ ] `masterEndpoint: redis-test-client.$NAMESPACE.svc.cluster.local:6379`

### 2.9 Verify Conditions

```bash
oc get rediscluster redis-test -n $NAMESPACE -o jsonpath='{.status.conditions}' | python3 -m json.tool
```

**Expected conditions**:
- [ ] `Available: True`
- [ ] `Progressing: False`
- [ ] `Degraded: False`

### 2.10 Verify Print Columns

```bash
oc get rediscluster -n $NAMESPACE
```

**Expected**: Table with columns Phase, Ready, Version, Age.

### 2.11 Verify Events

```bash
oc get events -n $NAMESPACE --field-selector involvedObject.name=redis-test --sort-by='.lastTimestamp'
```

**Expected**: Events for SecretCreated, ConfigMapCreated, HeadlessServiceCreated, ClientServiceCreated, StatefulSetCreated, PDBCreated.

---

## Phase 3: Idempotency

### 3.1 Verify No Duplicate Resources

```bash
oc delete pod -n $NAMESPACE -l control-plane=controller-manager
oc wait --for=condition=available deployment -l control-plane=controller-manager -n $NAMESPACE --timeout=60s
sleep 15

echo "Secrets: $(oc get secret -n $NAMESPACE 2>&1 | grep -c redis-test-auth) (should be 1)"
echo "ConfigMaps: $(oc get configmap -n $NAMESPACE 2>&1 | grep -c redis-test-config) (should be 1)"
echo "Headless Services: $(oc get service -n $NAMESPACE 2>&1 | grep -c redis-test-headless) (should be 1)"
echo "Client Services: $(oc get service -n $NAMESPACE 2>&1 | grep -c redis-test-client) (should be 1)"
echo "StatefulSets: $(oc get statefulset -n $NAMESPACE 2>&1 | grep -c redis-test) (should be 1)"
echo "PDBs: $(oc get pdb -n $NAMESPACE 2>&1 | grep -c redis-test-pdb) (should be 1)"
```

**Expected**: Exactly 1 of each resource, no duplicates.

### 3.2 Verify Password Unchanged After Reconciliation

```bash
PASS_BEFORE=$(oc get secret redis-test-auth -n $NAMESPACE -o jsonpath='{.data.REDIS_PASSWORD}')

oc label rediscluster redis-test -n $NAMESPACE test-reconcile=true
sleep 10

PASS_AFTER=$(oc get secret redis-test-auth -n $NAMESPACE -o jsonpath='{.data.REDIS_PASSWORD}')
[ "$PASS_BEFORE" = "$PASS_AFTER" ] && echo "PASS: Password unchanged (idempotent)" || echo "FAIL: Password changed!"
```

---

## Phase 4: Scaling

### 4.1 Scale Up

```bash
oc patch rediscluster redis-test -n $NAMESPACE --type merge -p '{"spec":{"replicas":5}}'
sleep 15

oc get statefulset redis-test -n $NAMESPACE -o jsonpath='{.spec.replicas}' && echo " replicas (should be 5)"
oc get pdb redis-test-pdb -n $NAMESPACE -o jsonpath='{.spec.minAvailable}' && echo " PDB minAvailable (should be 4)"
```

**Expected**:
- [ ] StatefulSet replicas updated to 5
- [ ] PDB minAvailable updated to 4 (replicas - 1)

### 4.2 Scale Down to 1 (PDB Should Be Deleted)

```bash
oc patch rediscluster redis-test -n $NAMESPACE --type merge -p '{"spec":{"replicas":1}}'
sleep 15

oc get statefulset redis-test -n $NAMESPACE -o jsonpath='{.spec.replicas}' && echo " replicas (should be 1)"
oc get pdb redis-test-pdb -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: PDB deleted (replicas <= 1)" || echo "FAIL: PDB still exists"
```

**Expected**:
- [ ] Replicas updated to 1
- [ ] PDB deleted (not needed when replicas ≤ 1)

### 4.3 Scale Back to 3 (PDB Should Be Re-Created)

```bash
oc patch rediscluster redis-test -n $NAMESPACE --type merge -p '{"spec":{"replicas":3}}'
sleep 15

oc get pdb redis-test-pdb -n $NAMESPACE && echo "PASS: PDB re-created" || echo "FAIL: PDB not re-created"
oc get pdb redis-test-pdb -n $NAMESPACE -o jsonpath='{.spec.minAvailable}' && echo " minAvailable (should be 2)"
```

**Expected**:
- [ ] PDB re-created with minAvailable = 2

---

## Phase 5: Finalizer and Deletion

### 5.1 Verify Finalizer Exists

```bash
oc get rediscluster redis-test -n $NAMESPACE -o jsonpath='{.metadata.finalizers}' && echo ""
```

**Expected**: `["cache.redis.example.com/finalizer"]`

### 5.2 Delete the CR

```bash
oc delete rediscluster redis-test -n $NAMESPACE
sleep 15

oc get secret redis-test-auth -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Secret cleaned"
oc get configmap redis-test-config -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: ConfigMap cleaned"
oc get service redis-test-headless -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Headless Service cleaned"
oc get service redis-test-client -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Client Service cleaned"
oc get statefulset redis-test -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: StatefulSet cleaned"
oc get pdb redis-test-pdb -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: PDB cleaned"
```

**Expected**:
- [ ] All 6 managed resources garbage collected
- [ ] No orphaned resources remain

---

## Phase 6: Validation Markers

### 6.1 Reject Invalid Replicas

```bash
cat <<EOF | oc apply -f - 2>&1
apiVersion: cache.redis.example.com/v1alpha1
kind: RedisCluster
metadata:
  name: redis-invalid
  namespace: $NAMESPACE
spec:
  replicas: 10
  version: "7.4"
  storage:
    size: 1Gi
EOF
```

**Expected**: Rejected — `replicas` max is 6.

### 6.2 Reject Invalid Version

```bash
cat <<EOF | oc apply -f - 2>&1
apiVersion: cache.redis.example.com/v1alpha1
kind: RedisCluster
metadata:
  name: redis-invalid
  namespace: $NAMESPACE
spec:
  replicas: 3
  version: "6.2"
  storage:
    size: 1Gi
EOF
```

**Expected**: Rejected — version enum only allows "7.2", "7.4".

### 6.3 Reject Invalid Storage Size Pattern

```bash
cat <<EOF | oc apply -f - 2>&1
apiVersion: cache.redis.example.com/v1alpha1
kind: RedisCluster
metadata:
  name: redis-invalid
  namespace: $NAMESPACE
spec:
  replicas: 3
  version: "7.4"
  storage:
    size: "100MB"
EOF
```

**Expected**: Rejected — storage size must match pattern `^[0-9]+[KMGT]i$`.

### 6.4 Verify Defaults Applied

```bash
cat <<EOF | oc apply -f -
apiVersion: cache.redis.example.com/v1alpha1
kind: RedisCluster
metadata:
  name: redis-defaults
  namespace: $NAMESPACE
spec:
  storage:
    size: 5Gi
EOF

sleep 5
oc get rediscluster redis-defaults -n $NAMESPACE -o jsonpath='{.spec.replicas}' && echo " (should be 3)"
oc get rediscluster redis-defaults -n $NAMESPACE -o jsonpath='{.spec.version}' && echo " (should be 7.4)"

oc delete rediscluster redis-defaults -n $NAMESPACE
```

**Expected**: Defaults applied — replicas=3, version="7.4".

---

## Phase 7: RBAC Verification

### 7.1 Verify Operator RBAC Works

```bash
oc auth can-i get secrets --as=system:serviceaccount:$NAMESPACE:redis-operator-controller-manager && echo "PASS" || echo "FAIL"
oc auth can-i create statefulsets --as=system:serviceaccount:$NAMESPACE:redis-operator-controller-manager && echo "PASS" || echo "FAIL"
oc auth can-i create poddisruptionbudgets --as=system:serviceaccount:$NAMESPACE:redis-operator-controller-manager && echo "PASS" || echo "FAIL"
oc auth can-i update redisclusters/status --as=system:serviceaccount:$NAMESPACE:redis-operator-controller-manager && echo "PASS" || echo "FAIL"
```

### 7.2 Verify No Excess Permissions

```bash
oc auth can-i create namespaces --as=system:serviceaccount:$NAMESPACE:redis-operator-controller-manager && echo "FAIL: excess perms" || echo "PASS: no excess"
oc auth can-i delete nodes --as=system:serviceaccount:$NAMESPACE:redis-operator-controller-manager && echo "FAIL: excess perms" || echo "PASS: no excess"
```

---

## Phase 8: Security Posture

### 8.1 Verify Non-Root Container

```bash
oc get deployment redis-operator-controller-manager -n $NAMESPACE -o jsonpath='{.spec.template.spec.securityContext.runAsNonRoot}' && echo " (should be true)"
```

### 8.2 Verify Capabilities Dropped

```bash
oc get deployment redis-operator-controller-manager -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].securityContext.capabilities.drop}' && echo ""
```

**Expected**: `["ALL"]`

---

## Phase 9: Health Probes

### 9.1 Verify Liveness Probe

```bash
oc get deployment redis-operator-controller-manager -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}' | python3 -m json.tool
```

**Expected**: httpGet on /healthz:8081

### 9.2 Verify Readiness Probe

```bash
oc get deployment redis-operator-controller-manager -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].readinessProbe}' | python3 -m json.tool
```

**Expected**: httpGet on /readyz:8081

---

## Phase 10: OLM Bundle Validation (Optional)

### 10.1 Bundle Validate

```bash
operator-sdk bundle validate bundle/
```

**Expected**: No errors.

---

## Phase 11: Multi-Instance

### 11.1 Create Multiple RedisClusters

```bash
for i in 1 2 3; do
cat <<EOF | oc apply -f -
apiVersion: cache.redis.example.com/v1alpha1
kind: RedisCluster
metadata:
  name: redis-multi-$i
  namespace: $NAMESPACE
spec:
  replicas: 1
  version: "7.4"
  storage:
    size: 1Gi
EOF
done

sleep 30
oc get rediscluster -n $NAMESPACE
```

**Expected**:
- [ ] 3 independent RedisClusters created
- [ ] Each has its own Secret, ConfigMap, 2 Services, StatefulSet
- [ ] No PDB (replicas=1 for each)
- [ ] No cross-contamination

### 11.2 Delete One, Others Unaffected

```bash
oc delete rediscluster redis-multi-2 -n $NAMESPACE
sleep 10

oc get rediscluster -n $NAMESPACE
oc get statefulset -n $NAMESPACE
```

**Expected**: redis-multi-1 and redis-multi-3 still Running, redis-multi-2 fully cleaned up.

---

## Scenario A Cleanup

```bash
oc delete rediscluster --all -n $NAMESPACE
sleep 15

# If NOT continuing to Scenario B, undeploy the operator:
# make undeploy                                                    # if deployed with make deploy
# operator-sdk cleanup redis-operator --namespace $NAMESPACE       # if deployed with OLM
# oc delete project $NAMESPACE
```

---

## Scenario A Summary Checklist

| # | Test | Phase | Expected |
|---|------|-------|----------|
| 1 | Operator deploys and starts | 1 | Pod Running |
| 2 | CRD installed with expected fields | 1 | auth, replicas, resources, storage, version |
| 3 | CR creates all 6 managed resources | 2.2 | Secret, ConfigMap, Service×2, StatefulSet, PDB |
| 4 | Owner references set on all resources | 2.2 | All point to RedisCluster |
| 5 | Headless service is ClusterIP None | 2.3 | clusterIP: None, port 6379 |
| 6 | Client service has real ClusterIP | 2.3 | Not None, port 6379 |
| 7 | StatefulSet has correct image and replicas | 2.4 | redis image, 3 replicas |
| 8 | ConfigMap has redis.conf content | 2.5 | maxmemory-policy, appendonly |
| 9 | Secret has generated REDIS_PASSWORD | 2.6 | 24-char random |
| 10 | PDB created with minAvailable=replicas-1 | 2.7 | minAvailable=2 |
| 11 | Status reaches Running phase | 2.8 | phase=Running, readyReplicas=3 |
| 12 | MasterEndpoint points to client service | 2.8 | redis-test-client...6379 |
| 13 | Conditions set correctly | 2.9 | Available=True, Degraded=False |
| 14 | Print columns display | 2.10 | Phase, Ready, Version, Age |
| 15 | Events recorded for each resource | 2.11 | 6 create events |
| 16 | Idempotent — no duplicates | 3.1 | Exactly 1 of each |
| 17 | Password unchanged on re-reconcile | 3.2 | Same base64 value |
| 18 | Scale up works | 4.1 | 5 replicas, PDB minAvailable=4 |
| 19 | Scale to 1 deletes PDB | 4.2 | PDB removed |
| 20 | Scale to 3 re-creates PDB | 4.3 | PDB with minAvailable=2 |
| 21 | Finalizer present | 5.1 | Finalizer in metadata |
| 22 | Deletion cleans all 6 resources | 5.2 | No orphans |
| 23 | Invalid replicas rejected | 6.1 | Validation error |
| 24 | Invalid version rejected | 6.2 | Validation error |
| 25 | Invalid storage pattern rejected | 6.3 | Validation error |
| 26 | Defaults applied | 6.4 | replicas=3, version=7.4 |
| 27 | RBAC allows needed operations | 7.1 | can-i returns yes |
| 28 | No excess RBAC permissions | 7.2 | can-i returns no |
| 29 | Container runs as non-root | 8.1 | runAsNonRoot=true |
| 30 | Capabilities dropped | 8.2 | drop=[ALL] |
| 31 | Health probes configured | 9 | /healthz, /readyz |
| 32 | Bundle validates | 10 | No errors |
| 33 | Multiple instances independent | 11.1 | No cross-contamination |
| 34 | Deleting one doesn't affect others | 11.2 | Others still Running |

---
---

# Scenario B: Redis Sentinel HA (v0.2.0)

Adds Redis Sentinel for automatic failover — a Deployment + Service pair managed conditionally. Built using:
- **Step 1** (Generate): `designing-operator-api` SKILL (Workflow B) — Added SentinelSpec to types
- **Step 2** (Generate): `implementing-reconciliation` SKILL (Workflow B) — Added reconcileSentinel (Deployment + Service)
- **Step 3a** (Test): `operator-test-generator` SUBAGENT (Workflow B) — Added sentinel tests
- **Step 3b** (Review): `operator-reviewer` SUBAGENT — Reviewed modified code
- **Step 4** (Generate): `bundling-operator` SKILL (Workflow B) — Updated CSV v0.1.0 → v0.2.0
- **Step 5** (Validate): `operator-bundle-validator` SUBAGENT — Validated updated bundle

**Changes**: SentinelSpec (enabled, replicas, image), reconcileSentinel creating Deployment + ClusterIP Service on port 26379, SentinelReady condition, sentinelEndpoint in status, CSV v0.2.0 with replaces.

**Prerequisites**: Scenario A completed successfully. All Scenario A CRs deleted.

## Scenario B Environment Setup

```bash
export IMG=quay.io/mpaulgreen/redis-operator:v0.2.0
export BUNDLE_IMG=quay.io/mpaulgreen/redis-operator-bundle:v0.2.0
export NAMESPACE=redis-operator-system

cd e2e/redis-operator
```

---

## Phase B.1: Build and Deploy v0.2.0

### B.1.1 Build the Operator Image

```bash
podman build --platform linux/amd64 -t $IMG .
podman push $IMG
```

### B.1.2 Deploy the Operator

#### Option A: `make deploy` (Development)

```bash
make manifests
make deploy IMG=$IMG
```

#### Option B: OLM

```bash
# Update CSV image reference
sed -i '' "s|quay.io/mpaulgreen/redis-operator:v0.2.0|$IMG|g" bundle/manifests/redis-operator.clusterserviceversion.yaml

# Refresh CRD in bundle
make manifests
cp config/crd/bases/cache.redis.example.com_redisclusters.yaml bundle/manifests/

# Build and push bundle
podman build -t $BUNDLE_IMG -f bundle.Dockerfile .
podman push $BUNDLE_IMG

# Create namespace first
oc new-project $NAMESPACE || oc create namespace $NAMESPACE

# Deploy via OLM
operator-sdk run bundle $BUNDLE_IMG --namespace $NAMESPACE --timeout 5m
```

### B.1.3 Verify Deployment

```bash
oc get pods -n $NAMESPACE -l control-plane=controller-manager
oc logs -n $NAMESPACE -l control-plane=controller-manager --tail=20

# CRD has sentinel fields
oc get crd redisclusters.cache.redis.example.com -o jsonpath='{.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties}' | python3 -c "import json,sys; print(sorted(json.load(sys.stdin).keys()))"
```

**Expected**:
- [ ] Pod 1/1 Running with v0.2.0 image
- [ ] CRD fields: auth, replicas, resources, sentinel, storage, version
- [ ] Controller watching 7 EventSources (added Deployment)

---

## Phase B.2: Existing Features Regression

### B.2.1 Create CR Without Sentinel

```bash
cat <<EOF | oc apply -f -
apiVersion: cache.redis.example.com/v1alpha1
kind: RedisCluster
metadata:
  name: redis-test
  namespace: $NAMESPACE
spec:
  replicas: 3
  version: "7.4"
  storage:
    size: 1Gi
EOF

sleep 30
```

### B.2.2 Verify All Scenario A Resources Created

```bash
echo "=== Managed Resources ==="
oc get secret redis-test-auth -n $NAMESPACE && echo "PASS: Secret" || echo "FAIL: Secret"
oc get configmap redis-test-config -n $NAMESPACE && echo "PASS: ConfigMap" || echo "FAIL: ConfigMap"
oc get service redis-test-headless -n $NAMESPACE && echo "PASS: Headless Service" || echo "FAIL: Headless Service"
oc get service redis-test-client -n $NAMESPACE && echo "PASS: Client Service" || echo "FAIL: Client Service"
oc get statefulset redis-test -n $NAMESPACE && echo "PASS: StatefulSet" || echo "FAIL: StatefulSet"
oc get pdb redis-test-pdb -n $NAMESPACE && echo "PASS: PDB" || echo "FAIL: PDB"

echo ""
echo "=== No Sentinel (not configured) ==="
oc get deployment redis-test-sentinel -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: No Sentinel Deployment" || echo "FAIL"
oc get service redis-test-sentinel -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: No Sentinel Service" || echo "FAIL"

echo ""
echo "=== Status ==="
oc get rediscluster redis-test -n $NAMESPACE -o wide
```

**Expected**:
- [ ] All 6 existing resources created (Secret, ConfigMap, Service×2, StatefulSet, PDB)
- [ ] No Sentinel Deployment or Service (sentinel not configured)
- [ ] Status shows Running

---

## Phase B.3: Enable Sentinel

### B.3.1 Enable Sentinel HA

```bash
oc patch rediscluster redis-test -n $NAMESPACE --type merge -p '{"spec":{"sentinel":{"enabled":true,"replicas":3}}}'
sleep 15
```

### B.3.2 Verify Sentinel Deployment Created

```bash
echo "=== Sentinel Deployment ==="
oc get deployment redis-test-sentinel -n $NAMESPACE
oc get deployment redis-test-sentinel -n $NAMESPACE -o jsonpath='{.spec.replicas}' && echo " replicas (should be 3)"
oc get deployment redis-test-sentinel -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].ports[0].containerPort}' && echo " port (should be 26379)"

echo ""
echo "=== Sentinel Deployment Owner Reference ==="
oc get deployment redis-test-sentinel -n $NAMESPACE -o jsonpath='{.metadata.ownerReferences[0].kind}' && echo " (should be RedisCluster)"

echo ""
echo "=== Sentinel Deployment Labels ==="
oc get deployment redis-test-sentinel -n $NAMESPACE -o jsonpath='{.metadata.labels}' | python3 -m json.tool
```

**Expected**:
- [ ] Deployment `redis-test-sentinel` created with 3 replicas
- [ ] Port 26379
- [ ] Owner reference → RedisCluster
- [ ] Labels include `component: sentinel`

### B.3.3 Verify Sentinel Service Created

```bash
echo "=== Sentinel Service ==="
oc get service redis-test-sentinel -n $NAMESPACE
oc get service redis-test-sentinel -n $NAMESPACE -o jsonpath='{.spec.type}' && echo " type (should be ClusterIP)"
oc get service redis-test-sentinel -n $NAMESPACE -o jsonpath='{.spec.ports[0].port}' && echo " port (should be 26379)"
oc get service redis-test-sentinel -n $NAMESPACE -o jsonpath='{.metadata.ownerReferences[0].kind}' && echo " owner (should be RedisCluster)"
```

**Expected**:
- [ ] Service `redis-test-sentinel` created (ClusterIP, port 26379)
- [ ] Owner reference → RedisCluster

### B.3.4 Verify SentinelReady Condition

```bash
oc get rediscluster redis-test -n $NAMESPACE -o jsonpath='{.status.conditions}' | python3 -c "
import json, sys
conditions = json.load(sys.stdin)
for c in conditions:
    if c['type'] == 'SentinelReady':
        print(f\"SentinelReady: {c['status']} (reason: {c['reason']})\")
"
echo ""
oc get rediscluster redis-test -n $NAMESPACE -o jsonpath='{.status.sentinelEndpoint}' && echo " sentinelEndpoint"
```

**Expected**:
- [ ] SentinelReady: True
- [ ] sentinelEndpoint: `redis-test-sentinel.<namespace>.svc.cluster.local:26379`

### B.3.5 Verify Existing Resources Unaffected

```bash
oc get statefulset redis-test -n $NAMESPACE && echo "PASS: StatefulSet still exists"
oc get pdb redis-test-pdb -n $NAMESPACE && echo "PASS: PDB still exists"
oc get rediscluster redis-test -n $NAMESPACE -o jsonpath='{.status.phase}' && echo " (should still be Running)"
```

---

## Phase B.4: Disable Sentinel

### B.4.1 Disable Sentinel

```bash
oc patch rediscluster redis-test -n $NAMESPACE --type merge -p '{"spec":{"sentinel":{"enabled":false}}}'
sleep 15

echo "=== Sentinel After Disable ==="
oc get deployment redis-test-sentinel -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Deployment deleted" || echo "FAIL: Deployment still exists"
oc get service redis-test-sentinel -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Service deleted" || echo "FAIL: Service still exists"

echo ""
echo "=== SentinelReady After Disable ==="
oc get rediscluster redis-test -n $NAMESPACE -o jsonpath='{.status.conditions}' | python3 -c "
import json, sys
conditions = json.load(sys.stdin)
for c in conditions:
    if c['type'] == 'SentinelReady':
        print(f\"SentinelReady: {c['status']} (reason: {c['reason']})\")
"
```

**Expected**:
- [ ] Sentinel Deployment deleted
- [ ] Sentinel Service deleted
- [ ] SentinelReady: False (SentinelDisabled)

---

## Phase B.5: Idempotency

### B.5.1 Re-enable Sentinel and Re-reconcile

```bash
# Re-enable sentinel
oc patch rediscluster redis-test -n $NAMESPACE --type merge -p '{"spec":{"sentinel":{"enabled":true,"replicas":3}}}'
sleep 15

# Restart controller
oc delete pod -n $NAMESPACE -l control-plane=controller-manager
oc wait --for=condition=available deployment -l control-plane=controller-manager -n $NAMESPACE --timeout=60s
sleep 15

echo "Sentinel Deployments: $(oc get deployment -n $NAMESPACE 2>&1 | grep -c redis-test-sentinel) (should be 1)"
echo "Sentinel Services: $(oc get service -n $NAMESPACE 2>&1 | grep -c redis-test-sentinel) (should be 1)"
echo "StatefulSets: $(oc get statefulset -n $NAMESPACE 2>&1 | grep -c redis-test) (should be 1)"
```

**Expected**: Exactly 1 of each, no duplicates.

---

## Phase B.6: Validation Markers

### B.6.1 Verify Sentinel Defaults Applied

```bash
cat <<EOF | oc apply -f -
apiVersion: cache.redis.example.com/v1alpha1
kind: RedisCluster
metadata:
  name: redis-sentinel-defaults
  namespace: $NAMESPACE
spec:
  replicas: 3
  version: "7.4"
  storage:
    size: 1Gi
  sentinel:
    enabled: true
EOF

sleep 5
oc get rediscluster redis-sentinel-defaults -n $NAMESPACE -o jsonpath='{.spec.sentinel.replicas}' && echo " (should be 3)"

oc delete rediscluster redis-sentinel-defaults -n $NAMESPACE
```

**Expected**: `sentinel.replicas` defaults to 3.

---

## Phase B.7: Delete CR with Sentinel

### B.7.1 Delete and Verify All Resources Cleaned

```bash
oc delete rediscluster redis-test -n $NAMESPACE
sleep 15

oc get secret redis-test-auth -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Secret cleaned"
oc get configmap redis-test-config -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: ConfigMap cleaned"
oc get service redis-test-headless -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Headless Service cleaned"
oc get service redis-test-client -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Client Service cleaned"
oc get statefulset redis-test -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: StatefulSet cleaned"
oc get pdb redis-test-pdb -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: PDB cleaned"
oc get deployment redis-test-sentinel -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Sentinel Deployment cleaned"
oc get service redis-test-sentinel -n $NAMESPACE 2>&1 | grep "not found" && echo "PASS: Sentinel Service cleaned"
```

**Expected**:
- [ ] All 8 managed resources garbage collected (Secret, ConfigMap, Service×2, StatefulSet, PDB, Sentinel Deployment, Sentinel Service)

---

## Phase B.8: RBAC Verification

### B.8.1 Verify Deployment RBAC

```bash
oc auth can-i create deployments --as=system:serviceaccount:redis-operator-system:redis-operator-controller-manager -n redis-operator-system && echo "PASS: Can create Deployments" || echo "FAIL"
oc auth can-i delete deployments --as=system:serviceaccount:redis-operator-system:redis-operator-controller-manager -n redis-operator-system && echo "PASS: Can delete Deployments" || echo "FAIL"
```

**Expected**: Both return "yes".

---

## Phase B.9: OLM Bundle Validation

### B.9.1 Verify Bundle Version

```bash
echo "=== CSV Version ==="
grep 'name:.*redis-operator.v' bundle/manifests/redis-operator.clusterserviceversion.yaml | head -1
grep 'replaces:' bundle/manifests/redis-operator.clusterserviceversion.yaml
grep '^  version:' bundle/manifests/redis-operator.clusterserviceversion.yaml
```

**Expected**:
- [ ] CSV name: `redis-operator.v0.2.0`
- [ ] replaces: `redis-operator.v0.1.0`
- [ ] version: `0.2.0`

### B.9.2 Verify Sentinel Descriptors

```bash
grep -E 'sentinel' bundle/manifests/redis-operator.clusterserviceversion.yaml | grep 'path:' | head -5
```

**Expected**: specDescriptors for sentinel, sentinel.enabled, sentinel.replicas.

### B.9.3 Verify Deployment RBAC in CSV

```bash
grep -A3 'deployments' bundle/manifests/redis-operator.clusterserviceversion.yaml | head -5
```

**Expected**: `apps/deployments` with CRUD verbs.

### B.9.4 Bundle Validate

```bash
operator-sdk bundle validate bundle/
```

**Expected**: No errors.

---

## Scenario B Cleanup

```bash
oc delete rediscluster --all -n $NAMESPACE
sleep 15

# If NOT continuing to Scenario C, undeploy:
# make undeploy                                                    # if deployed with make deploy
# operator-sdk cleanup redis-operator --namespace $NAMESPACE       # if deployed with OLM
# oc delete project $NAMESPACE
```

---

## Scenario B Summary Checklist

| # | Test | Phase | Expected |
|---|------|-------|----------|
| 1 | Operator deploys with v0.2.0 image | B.1 | Pod Running, CRD has sentinel field |
| 2 | All Scenario A resources work without sentinel | B.2 | 6 resources created |
| 3 | No sentinel Deployment/Service when not configured | B.2 | Not found |
| 4 | Sentinel Deployment created when enabled | B.3 | 3 replicas, port 26379 |
| 5 | Sentinel Deployment has correct owner ref | B.3 | RedisCluster |
| 6 | Sentinel Deployment has component=sentinel label | B.3 | Labels correct |
| 7 | Sentinel Service created (ClusterIP, 26379) | B.3 | Port 26379 |
| 8 | Sentinel Service has correct owner ref | B.3 | RedisCluster |
| 9 | SentinelReady condition True | B.3 | SentinelReady |
| 10 | sentinelEndpoint in status | B.3 | Correct DNS:26379 |
| 11 | Existing resources unaffected | B.3 | StatefulSet/PDB ok |
| 12 | Sentinel Deployment deleted when disabled | B.4 | Not found |
| 13 | Sentinel Service deleted when disabled | B.4 | Not found |
| 14 | SentinelReady False when disabled | B.4 | SentinelDisabled |
| 15 | Idempotent — no duplicate sentinel resources | B.5 | Exactly 1 each |
| 16 | sentinel.replicas defaults to 3 | B.6 | Default applied |
| 17 | All 8 resources cleaned on CR delete | B.7 | Including sentinel |
| 18 | Deployment RBAC works | B.8 | can-i returns yes |
| 19 | CSV version 0.2.0 with replaces | B.9 | Correct upgrade path |
| 20 | Sentinel descriptors in CSV | B.9 | sentinel.* fields present |
| 21 | Deployment RBAC in CSV | B.9 | apps/deployments |
| 22 | Bundle validates | B.9 | No errors |
