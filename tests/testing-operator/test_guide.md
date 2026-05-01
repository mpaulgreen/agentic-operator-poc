# Sprint 4 Test Guide: `testing-operator` Skill

## Prerequisites

- Go 1.22+ installed
- A scaffolded + designed + implemented operator project (Tests 1.1 + 2.1 + 2.2 + 3.1)
- The skill is at `.claude/skills/testing-operator/`

## Test Order

1. **4.1**: Generate full test suite for RedisCluster controller
2. **4.2**: Generate tests for single new method (ConfigMap)
3. **4.3**: SDK test comparison
4. **I-1.2.3.4**: Integration — scaffold + design + reconcile + test

---

## Test 4.1 — Generate Full Test Suite (Workflow A)

### Step 1: Ensure /tmp/redis-operator-test/ has controller files from Test 3.1

### Step 2: Prompt

```
Using the testing-operator skill, generate a complete test suite for the 
RedisCluster controller at /tmp/redis-operator-test/. The controller has:
- reconcileSecret (creates credentials Secret)
- reconcileService (creates headless Service)
- reconcileStatefulSet (creates StatefulSet, updates replicas)
- reconcileConfigMap (creates redis.conf ConfigMap)
- Finalizer lifecycle
- Status updates from StatefulSet

Generate:
- internal/controller/suite_test.go (envtest setup)
- internal/controller/rediscluster_controller_test.go (lifecycle + per-method tests)
```

### Step 3: Verify

```bash
# Files exist
test -f /tmp/redis-operator-test/internal/controller/suite_test.go && echo "PASS: suite_test.go" || echo "FAIL"
test -f /tmp/redis-operator-test/internal/controller/rediscluster_controller_test.go && echo "PASS: controller_test.go" || echo "FAIL"

# Test coverage script
bash .claude/skills/testing-operator/scripts/check-test-coverage.sh /tmp/redis-operator-test/

# Test matrix
python3 .claude/skills/testing-operator/scripts/generate-test-matrix.py /tmp/redis-operator-test/internal/controller/

# Compilation (tests compile even if envtest binaries not available)
cd /tmp/redis-operator-test && go vet ./internal/controller/...
```

### Acceptance Criteria

- [ ] suite_test.go exists with envtest setup (BeforeSuite, AfterSuite)
- [ ] controller_test.go exists with test cases
- [ ] check-test-coverage.sh passes (finds test files + test cases)
- [ ] generate-test-matrix.py passes (all reconcileX methods covered)
- [ ] `go vet ./internal/controller/...` passes (tests compile)

---

## Test 4.2 — Generate Tests for Single New Method (Workflow B)

### Step 1: Ensure Test 4.1 is complete

### Step 2: Prompt

```
Using the testing-operator skill, add tests for the reconcileConfigMap 
method in the RedisCluster controller at /tmp/redis-operator-test/. 
Add to the existing test file. Test that ConfigMap is created with 
redis.conf content, and that it's idempotent.
```

### Step 3: Verify

```bash
grep 'reconcileConfigMap\|ConfigMap' /tmp/redis-operator-test/internal/controller/rediscluster_controller_test.go > /dev/null && echo "PASS: ConfigMap tests present" || echo "FAIL"
grep 'redis.conf\|maxmemory' /tmp/redis-operator-test/internal/controller/rediscluster_controller_test.go > /dev/null && echo "PASS: verifies content" || echo "FAIL"

cd /tmp/redis-operator-test && go vet ./internal/controller/...
```

### Acceptance Criteria

- [ ] ConfigMap test cases added to existing file
- [ ] Tests verify ConfigMap data content (redis.conf)
- [ ] Existing tests unchanged
- [ ] `go vet` passes

---

## Test 4.3 — SDK Test Comparison

### Step 1: Create SDK project (if not exists)

```bash
rm -rf /tmp/redis-operator-sdk
mkdir -p /tmp/redis-operator-sdk && cd /tmp/redis-operator-sdk
operator-sdk init --domain redis.example.com --repo github.com/example/redis-operator --plugins=go/v4
operator-sdk create api --group cache --version v1alpha1 --kind RedisCluster --resource --controller
```

### Step 2: Compare test structure

```bash
echo "=== SDK test files ==="
find /tmp/redis-operator-sdk -name '*_test.go' -type f | sed 's|/tmp/redis-operator-sdk/||' | sort

echo ""
echo "=== SKILL test files ==="
find /tmp/redis-operator-test -name '*_test.go' -type f | sed 's|/tmp/redis-operator-test/||' | sort

echo ""
echo "=== SDK test cases ==="
grep -c 'It(' /tmp/redis-operator-sdk/internal/controller/rediscluster_controller_test.go 2>/dev/null || echo "0"

echo "=== SKILL test cases ==="
grep -rc 'It(' /tmp/redis-operator-test/internal/controller/*_test.go 2>/dev/null | awk -F: '{s+=$2} END {print s}'
```

### Expected Differences

| Aspect | SDK | Skill | Why |
|--------|-----|-------|-----|
| Test files | 3 (suite, controller, e2e) | 2+ (suite, controller) | Skill focuses on unit/integration |
| Test cases | 1 (basic reconcile) | 5+ (lifecycle, per-method, idempotency) | Skill tests real patterns |
| Reconciler method tests | 0 | 3+ per method | Skill verifies each resource |
| FakeRecorder | Not used | Used | Skill captures events |
| E2E tests | Present (skeleton) | Optional | E2E needs real cluster |

---

## Cleanup

```bash
rm -rf /tmp/redis-operator-test /tmp/redis-operator-sdk
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| `go test` fails: no envtest binaries | Need to download k8s binaries | Run `make envtest` or set `KUBEBUILDER_ASSETS` |
| `go vet` fails: import cycle | Test imports wrong package | Tests must be in `controller` package, not `controller_test` |
| Suite hangs | envtest can't start etcd | Check binary path in suite_test.go |
| Tests pass but coverage low | Not enough test scenarios | Use generate-test-matrix.py to find gaps |
