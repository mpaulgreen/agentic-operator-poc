# Sprint 3 Test Guide: `implementing-reconciliation` Skill

## Prerequisites

- Go 1.22+ installed
- operator-sdk v1.37.0+ installed (for SDK comparison)
- A scaffolded + designed operator project at `/tmp/redis-operator-test/` (run Tests 1.1 + 2.1 + 2.2 first)
- The skill is at `.claude/skills/implementing-reconciliation/`

## Test Order

1. **3.1**: Simple reconciler (Secret + Service + StatefulSet)
2. **3.2**: Add finalizer lifecycle
3. **3.3**: Add resource to existing controller (ConfigMap)
4. **I-1.2.3**: Integration — scaffold + design + reconcile end-to-end
5. **3.4**: SDK comparison

---

## Test 3.1 — Simple Reconciler (Workflow A)

### Step 1: Ensure Tests 1.1 + 2.1 + 2.2 project exists at /tmp/redis-operator-test/

### Step 2: Prompt

```
Using the implementing-reconciliation skill, implement reconciliation for 
the RedisCluster controller at /tmp/redis-operator-test/. The controller 
should reconcile:
1. Secret (redis credentials with generated password)
2. Service (headless service on port 6379)
3. StatefulSet (redis containers using spec.version for image, spec.replicas for count)

Use the three-phase pattern (fetch, orchestrate, status), check-create 
idempotency, owner references, event recording, and status updates 
(readyReplicas from StatefulSet, endpoint, phase, conditions).

Split across: controller.go, reconcilers.go, status.go, conditions.go, helpers.go
```

### Step 3: Verify

```bash
# RBAC validation
python3 .claude/skills/implementing-reconciliation/scripts/validate-rbac-annotations.py \
  /tmp/redis-operator-test/internal/controller/rediscluster_controller.go

# Idempotency check
python3 .claude/skills/implementing-reconciliation/scripts/check-idempotency.py \
  /tmp/redis-operator-test/internal/controller/rediscluster_reconcilers.go

# File structure
for f in rediscluster_controller.go rediscluster_reconcilers.go rediscluster_status.go rediscluster_conditions.go rediscluster_helpers.go; do
  test -f /tmp/redis-operator-test/internal/controller/$f && echo "PASS: $f" || echo "FAIL: $f"
done

# Key patterns
grep 'SetControllerReference' /tmp/redis-operator-test/internal/controller/rediscluster_reconcilers.go > /dev/null && echo "PASS: owner refs" || echo "FAIL"
grep 'r.Recorder.Event' /tmp/redis-operator-test/internal/controller/rediscluster_reconcilers.go > /dev/null && echo "PASS: event recording" || echo "FAIL"
grep 'r.Status().Update' /tmp/redis-operator-test/internal/controller/rediscluster_status.go > /dev/null && echo "PASS: status update" || echo "FAIL"
grep 'errors.IsNotFound' /tmp/redis-operator-test/internal/controller/rediscluster_reconcilers.go > /dev/null && echo "PASS: idempotency guard" || echo "FAIL"

# Compilation
cd /tmp/redis-operator-test && go build -o bin/manager ./cmd/main.go && echo "COMPILE: PASS" || echo "COMPILE: FAIL"
```

### Acceptance Criteria

- [ ] 5 controller files exist (controller, reconcilers, status, conditions, helpers)
- [ ] validate-rbac-annotations.py passes (RBAC matches resources)
- [ ] check-idempotency.py passes (Get before Create, owner refs, events)
- [ ] Three-phase pattern in Reconcile() (fetch, orchestrate, status)
- [ ] Check-create pattern in each reconcileX method
- [ ] SetControllerReference on all created resources
- [ ] Event recording on create success/failure
- [ ] Status update reads StatefulSet readyReplicas
- [ ] Conditions updated (Available, Progressing, Degraded)
- [ ] Compiles

---

## Test 3.2 — Finalizer Lifecycle

### Step 1: Ensure Test 3.1 is complete

### Step 2: Prompt

```
Using the implementing-reconciliation skill, add finalizer support to the 
RedisCluster controller at /tmp/redis-operator-test/. Add a finalizer 
on first reconcile, handle deletion with cleanup, and remove the finalizer.
```

### Step 3: Verify

```bash
grep -E 'finalizerName|Finalizer.*=.*"' /tmp/redis-operator-test/internal/controller/rediscluster_controller.go > /dev/null && echo "PASS: finalizer const" || echo "FAIL"
grep 'ContainsFinalizer' /tmp/redis-operator-test/internal/controller/rediscluster_controller.go > /dev/null && echo "PASS: check finalizer" || echo "FAIL"
grep 'AddFinalizer' /tmp/redis-operator-test/internal/controller/rediscluster_controller.go > /dev/null && echo "PASS: add finalizer" || echo "FAIL"
grep 'RemoveFinalizer' /tmp/redis-operator-test/internal/controller/rediscluster_controller.go > /dev/null && echo "PASS: remove finalizer" || echo "FAIL"
grep 'handleDeletion' /tmp/redis-operator-test/internal/controller/rediscluster_controller.go > /dev/null && echo "PASS: handleDeletion" || echo "FAIL"
grep 'DeletionTimestamp' /tmp/redis-operator-test/internal/controller/rediscluster_controller.go > /dev/null && echo "PASS: deletion check" || echo "FAIL"

cd /tmp/redis-operator-test && go build -o bin/manager ./cmd/main.go && echo "COMPILE: PASS" || echo "COMPILE: FAIL"
```

### Acceptance Criteria

- [ ] Finalizer constant defined
- [ ] Finalizer added on first reconcile
- [ ] DeletionTimestamp checked in Reconcile()
- [ ] handleDeletion() removes finalizer after cleanup
- [ ] Compiles

---

## Test 3.3 — Add Resource to Existing Controller (Workflow B)

### Step 1: Ensure Test 3.2 is complete

### Step 2: Prompt

```
Using the implementing-reconciliation skill, add a reconcileConfigMap 
method to the RedisCluster controller. The ConfigMap should contain 
redis.conf with settings: maxmemory 256mb, maxmemory-policy allkeys-lru. 
Follow the same check-create pattern as existing methods. Add RBAC marker 
and Owns() for ConfigMap.
```

### Step 3: Verify

```bash
grep 'func.*reconcileConfigMap' /tmp/redis-operator-test/internal/controller/rediscluster_reconcilers.go > /dev/null && echo "PASS: method exists" || echo "FAIL"
grep 'configmaps' /tmp/redis-operator-test/internal/controller/rediscluster_controller.go > /dev/null && echo "PASS: RBAC for configmaps" || echo "FAIL"
grep 'ConfigMap' /tmp/redis-operator-test/internal/controller/rediscluster_controller.go | grep -i 'owns' > /dev/null && echo "PASS: Owns ConfigMap" || echo "FAIL"

cd /tmp/redis-operator-test && go build -o bin/manager ./cmd/main.go && echo "COMPILE: PASS" || echo "COMPILE: FAIL"
```

### Acceptance Criteria

- [ ] reconcileConfigMap() method exists with check-create pattern
- [ ] RBAC annotation for configmaps added
- [ ] Owns(&corev1.ConfigMap{}) in SetupWithManager
- [ ] Called in correct dependency position (after Secret, before StatefulSet)
- [ ] Compiles

---

## Test I-1.2.3 — Integration: Scaffold + Design + Reconcile

### Step 1: Clean up
```bash
rm -rf /tmp/notification-operator-test
```

### Step 2: Prompt

```
Build a notification-operator end-to-end:
1. Scaffold project (domain: notify.example.com, group: notify, kind: NotificationChannel)
2. Design CRD (type enum email/slack/pagerduty, endpoint string, retryCount 1-5 default 3)
3. Implement controller reconciling: Secret (API credentials), Deployment (notification worker)
Generate under /tmp/notification-operator-test/
```

### Step 3: Verify

```bash
bash .claude/skills/scaffolding-operator/scripts/validate-project-structure.sh /tmp/notification-operator-test/
python3 .claude/skills/designing-operator-api/scripts/validate-api-types.py /tmp/notification-operator-test/api/v1alpha1/notificationchannel_types.go
python3 .claude/skills/implementing-reconciliation/scripts/check-idempotency.py /tmp/notification-operator-test/internal/controller/notificationchannel_reconcilers.go
cd /tmp/notification-operator-test && go mod tidy && go build ./...
```

### Acceptance Criteria

- [ ] Scaffolding checks pass
- [ ] Types validation passes
- [ ] Idempotency checks pass
- [ ] Full project compiles end-to-end

---

## Test 3.4 — SDK Comparison

The SDK generates a stub controller with empty Reconcile(). Our skill generates a full controller with reconciler methods, status updates, conditions, and event recording.

### Step 1: Create SDK project

```bash
rm -rf /tmp/redis-operator-sdk
mkdir -p /tmp/redis-operator-sdk && cd /tmp/redis-operator-sdk
operator-sdk init --domain redis.example.com --repo github.com/example/redis-operator --plugins=go/v4
operator-sdk create api --group cache --version v1alpha1 --kind RedisCluster --resource --controller
```

### Step 2: Compare structural elements

```bash
echo "=== SDK controller ==="
grep -c 'func.*Reconcile\|func.*Setup\|func.*reconcile\|func.*handle\|func.*update' /tmp/redis-operator-sdk/internal/controller/rediscluster_controller.go 2>/dev/null || echo "Run SDK scaffold first"

echo "=== SKILL controller ==="
find /tmp/redis-operator-test/internal/controller -name 'rediscluster_*.go' ! -name '*_test.go' | wc -l | xargs -I{} echo "{} controller files"
grep -rch 'func ' /tmp/redis-operator-test/internal/controller/rediscluster_*.go 2>/dev/null | awk '{s+=$1} END {print s " total methods"}'
```

### Expected Differences

| Aspect | SDK | Skill | Why |
|--------|-----|-------|-----|
| Controller files | 1 | 5+ | Skill splits by concern |
| Reconcile() body | Empty (TODO) | Three-phase pattern | Skill implements real logic |
| reconcileX methods | 0 | 3-4 | Skill creates resources |
| Status updates | 0 | updateStatus + conditions | Skill reports state |
| Event recording | 0 | Multiple | Skill records actions |
| RBAC markers | 3 (CRD only) | 8+ (all managed resources) | Skill matches resources |
| Owner references | 0 | On all created resources | Skill enables GC |

---

## Cleanup

```bash
rm -rf /tmp/redis-operator-test /tmp/notification-operator-test
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| `go build` fails: undefined controllerutil | Missing import | Add `"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"` |
| `go build` fails: undefined record | Missing import | Add `"k8s.io/client-go/tools/record"` |
| RBAC script fails: no markers | Markers use wrong spacing | Both `//+kubebuilder:rbac` and `// +kubebuilder:rbac` work |
| Idempotency fails: Create without Get | Check-create pattern missing | Add Get() + IsNotFound check before Create() |
| Status update conflict | Stale resource version | Re-fetch CR before status update |
