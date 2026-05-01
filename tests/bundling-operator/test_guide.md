# Sprint 5 Test Guide: `bundling-operator` Skill

## Prerequisites

- Go 1.22+ installed
- operator-sdk v1.37.0+ installed (for SDK comparison)
- PyYAML installed (`pip install pyyaml`)
- A scaffolded + designed + implemented + tested operator project (Tests 1.1 + 2.1 + 2.2 + 3.1 + 4.1)
- The skill is at `.claude/skills/bundling-operator/`

## Test Order

1. **5.1**: Generate initial OLM bundle for redis-operator
2. **5.2**: Update bundle from v0.1.0 to v0.2.0
3. **5.3**: SDK bundle comparison
4. **I-1.2.3.4.5**: Integration — scaffold + design + reconcile + test + bundle

---

## Test 5.1 — Generate Initial OLM Bundle (Workflow A)

### Step 1: Ensure /tmp/redis-operator-test/ has controller files from Tests 1.1 + 2.1 + 2.2 + 3.1

### Step 2: Prompt

```
Using the bundling-operator skill, create an OLM bundle for the
redis-operator v0.1.0 at /tmp/redis-operator-test/. The operator:
- Manages RedisCluster CRD (cache.redis.example.com/v1alpha1)
- CRD spec fields: replicas, version, storage, sentinel, resources, backup
- CRD status fields: phase, readyReplicas, conditions, endpoint
- Needs RBAC for: statefulsets, services, secrets, configmaps
- Install modes: OwnNamespace, SingleNamespace, AllNamespaces
- Channel: alpha
- Category: Database
- Display name: Redis Operator
- Description: Manages Redis clusters on Kubernetes with HA via Sentinel
- Image: quay.io/example/redis-operator:v0.1.0

Generate:
- bundle/manifests/redis-operator.clusterserviceversion.yaml
- bundle/manifests/cache.redis.example.com_redisclusters.yaml (CRD)
- bundle/metadata/annotations.yaml
- bundle/tests/scorecard/config.yaml
- bundle.Dockerfile
```

### Step 3: Verify

```bash
# Bundle structure
bash .claude/skills/bundling-operator/scripts/validate-bundle-structure.sh /tmp/redis-operator-test/

# CSV validation
python3 .claude/skills/bundling-operator/scripts/validate-csv.py \
  /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml

# Scorecard readiness
python3 .claude/skills/bundling-operator/scripts/check-scorecard-readiness.py /tmp/redis-operator-test/bundle/

# Key checks
grep 'redis-operator.v0.1.0' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: CSV name" || echo "FAIL"
grep 'version: 0.1.0' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: version" || echo "FAIL"
grep 'RedisCluster' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: owned CRD" || echo "FAIL"
grep 'alm-examples' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: alm-examples" || echo "FAIL"
grep 'specDescriptors' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: specDescriptors" || echo "FAIL"
grep 'statusDescriptors' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: statusDescriptors" || echo "FAIL"
```

### Acceptance Criteria

- [ ] Bundle structure valid (validate-bundle-structure.sh passes)
- [ ] CSV has all required sections (validate-csv.py passes)
- [ ] Scorecard readiness passes (check-scorecard-readiness.py)
- [ ] CSV name is `redis-operator.v0.1.0`
- [ ] spec.version is `0.1.0`
- [ ] Owned CRD is RedisCluster
- [ ] alm-examples has valid sample CR
- [ ] specDescriptors match CRD spec fields
- [ ] statusDescriptors match CRD status fields
- [ ] RBAC includes secrets, services, statefulsets, configmaps
- [ ] Deployment has manager container with correct image

---

## Test 5.2 — Update Bundle for New Version (Workflow B)

### Step 1: Ensure Test 5.1 is complete

### Step 2: Prompt

```
Using the bundling-operator skill, update the redis-operator bundle at
/tmp/redis-operator-test/ from v0.1.0 to v0.2.0. Changes:
- Added BackupSpec to CRD (schedule, retentionDays, destination fields)
- New reconcileCronJob method added (needs RBAC for batch/cronjobs)
- Update the CSV with:
  - New version 0.2.0
  - replaces: redis-operator.v0.1.0
  - New RBAC for batch/cronjobs
  - New specDescriptors for backup fields
  - Updated alm-examples showing backup configuration
```

### Step 3: Verify

```bash
# CSV version updated
grep 'redis-operator.v0.2.0' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: CSV name v0.2.0" || echo "FAIL"
grep 'version: 0.2.0' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: version 0.2.0" || echo "FAIL"

# replaces field
grep 'replaces: redis-operator.v0.1.0' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: replaces" || echo "FAIL"

# New RBAC
grep 'cronjobs' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: cronjobs RBAC" || echo "FAIL"

# Backup descriptors
grep -i 'backup\|schedule\|retention' /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml > /dev/null && echo "PASS: backup descriptors" || echo "FAIL"

# All scripts still pass
python3 .claude/skills/bundling-operator/scripts/validate-csv.py \
  /tmp/redis-operator-test/bundle/manifests/redis-operator.clusterserviceversion.yaml
```

### Acceptance Criteria

- [ ] CSV version updated to 0.2.0
- [ ] `spec.replaces` set to redis-operator.v0.1.0
- [ ] New RBAC entry for batch/cronjobs
- [ ] New specDescriptors for backup fields
- [ ] alm-examples includes backup configuration
- [ ] All 3 validation scripts still pass

---

## Test 5.3 — SDK Bundle Comparison

### Step 1: Generate SDK bundle

```bash
rm -rf /tmp/redis-operator-sdk
mkdir -p /tmp/redis-operator-sdk && cd /tmp/redis-operator-sdk
operator-sdk init --domain redis.example.com --repo github.com/example/redis-operator --plugins=go/v4
operator-sdk create api --group cache --version v1alpha1 --kind RedisCluster --resource --controller
make manifests
make bundle IMG=quay.io/example/redis-operator:v0.1.0 BUNDLE_CHANNELS=alpha VERSION=0.1.0
```

### Step 2: Compare

```bash
echo "=== SDK bundle files ==="
find /tmp/redis-operator-sdk/bundle -type f | sed 's|/tmp/redis-operator-sdk/||' | sort

echo ""
echo "=== SKILL bundle files ==="
find /tmp/redis-operator-test/bundle -type f | sed 's|/tmp/redis-operator-test/||' | sort

echo ""
echo "=== SDK CSV sections ==="
grep -c 'specDescriptors\|statusDescriptors\|alm-examples\|clusterPermissions\|installModes' /tmp/redis-operator-sdk/bundle/manifests/*.clusterserviceversion.yaml 2>/dev/null

echo "=== SKILL CSV sections ==="
grep -c 'specDescriptors\|statusDescriptors\|alm-examples\|clusterPermissions\|installModes' /tmp/redis-operator-test/bundle/manifests/*.clusterserviceversion.yaml 2>/dev/null
```

### Expected Differences

| Aspect | SDK (`make bundle`) | Skill | Why |
|--------|---------------------|-------|-----|
| Bundle structure | Same layout | Same layout | Both follow OLM spec |
| CSV specDescriptors | 0 (empty) | Multiple per field | Skill extracts from types.go |
| CSV statusDescriptors | 0 (empty) | Multiple per field | Skill extracts from types.go |
| alm-examples | Minimal stub | Rich with all fields | Skill builds from Spec |
| RBAC rules | CRD-only (3 rules) | All managed resources | Skill reads controller RBAC markers |
| Scorecard config | Generated | Same structure | Match |

---

## Cleanup

```bash
rm -rf /tmp/redis-operator-test /tmp/redis-operator-sdk
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| validate-csv.py fails: no yaml module | PyYAML not installed | `pip install pyyaml` |
| CSV name mismatch | metadata.name doesn't match `<pkg>.v<ver>` | Ensure both match |
| scorecard fails: no alm-examples | Missing annotation | Add alm-examples JSON to CSV metadata.annotations |
| Bundle structure fails: no Dockerfile | bundle.Dockerfile at wrong path | Must be at project root, not inside bundle/ |
| RBAC incomplete | Controller markers not read | Check controller for `//+kubebuilder:rbac` markers |
