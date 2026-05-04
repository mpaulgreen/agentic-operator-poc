# Sprint 6 Test Guide: `operator-reviewer` Subagent

## Prerequisites

- Sprints 1-5 complete (all 5 skills built)
- The agent is at `.claude/agents/operator-reviewer.md`
- A working redis-operator project (from previous tests, or rebuild)
- The database-operator at `../go-operator/operators/database-operator/`

## Test Order

1. **6.1**: Review operator with known issues (5 planted bugs)
2. **6.2**: Review clean operator (database-operator)
3. **I-6**: Review then fix

---

## Test 6.1 — Review Operator with Known Issues

### Step 1: Create flawed operator

Copy the redis-operator-test project, then plant 5 deliberate issues in the controller files.

```bash
rm -rf /tmp/redis-operator-flawed
cp -r /tmp/redis-operator-test /tmp/redis-operator-flawed
```

Then apply these 5 modifications to plant the issues:

**Issue 1 — Non-idempotent reconcileService() (Critical)**

In `/tmp/redis-operator-flawed/internal/controller/rediscluster_reconcilers.go`, replace the `reconcileService` method with one that calls `r.Create()` directly without checking if the Service exists:

```go
func (r *RedisClusterReconciler) reconcileService(ctx context.Context, cr *cachev1alpha1.RedisCluster) error {
	logger := log.FromContext(ctx)
	name := fmt.Sprintf("%s-service", cr.Name)

	// BUG: No Get() + IsNotFound check — creates duplicate on every reconcile
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labelsForRedisCluster(cr),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Selector:  labelsForRedisCluster(cr),
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(cr, svc, r.Scheme); err != nil {
		return err
	}

	logger.Info("Creating Service", "name", name)
	if err := r.Create(ctx, svc); err != nil {
		return err
	}

	r.Recorder.Event(cr, corev1.EventTypeNormal, "ServiceCreated",
		fmt.Sprintf("Created Service %s", name))
	return nil
}
```

**Issue 2 — Missing SetControllerReference in reconcileConfigMap() (Critical)**

In the same file, modify `reconcileConfigMap` to remove the `SetControllerReference` call:

```go
// In reconcileConfigMap(), REMOVE this block:
//	if err := controllerutil.SetControllerReference(cr, cm, r.Scheme); err != nil {
//		return err
//	}
// The ConfigMap is created without owner references — will be orphaned on CR deletion.
```

**Issue 3 — Wildcard RBAC (Critical)**

In `/tmp/redis-operator-flawed/internal/controller/rediscluster_controller.go`, change one RBAC marker to use wildcard:

```go
// Change:
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// To:
// +kubebuilder:rbac:groups="",resources=secrets,verbs=*
```

**Issue 4 — Missing status condition update in error handler (Warning)**

In the same controller file, modify `handleError` to NOT update conditions:

```go
func (r *RedisClusterReconciler) handleError(ctx context.Context, cr *cachev1alpha1.RedisCluster, err error, msg string) (ctrl.Result, error) {
	log.FromContext(ctx).Error(err, msg)
	r.Recorder.Event(cr, corev1.EventTypeWarning, "ReconcileError", fmt.Sprintf("%s: %v", msg, err))
	cr.Status.Phase = "Failed"
	_ = r.Status().Update(ctx, cr)
	// BUG: No setDegradedCondition() call — conditions don't reflect error state
	return ctrl.Result{RequeueAfter: 30 * time.Second}, err
}
```

**Issue 5 — Finalizer removal without re-fetching CR (Critical)**

In the same controller file, modify `handleDeletion` so it doesn't re-fetch before removing the finalizer:

```go
func (r *RedisClusterReconciler) handleDeletion(ctx context.Context, cr *cachev1alpha1.RedisCluster) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(cr, redisClusterFinalizer) {
		return ctrl.Result{}, nil
	}

	r.Recorder.Event(cr, corev1.EventTypeNormal, "Deleting", "Starting cleanup")

	// BUG: Should re-fetch CR here with r.Get() to avoid stale ResourceVersion
	// The cr object may be stale if another controller modified it since we last fetched it.
	controllerutil.RemoveFinalizer(cr, redisClusterFinalizer)
	if err := r.Update(ctx, cr); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(cr, corev1.EventTypeNormal, "Deleted", "Cleanup completed")
	return ctrl.Result{}, nil
}
```

### Step 2: Prompt

```
Using the operator-reviewer agent, review the operator code at 
/tmp/redis-operator-flawed/ for best practices and common mistakes. 
Check the controller, API types, and RBAC.
```

### Step 3: Verify

Check that the review identifies all 5 planted issues:

```bash
# The review output should mention:
# 1. reconcileService — non-idempotent (Create without Get/IsNotFound)
# 2. reconcileConfigMap — missing SetControllerReference/owner ref
# 3. RBAC — wildcard verbs=*
# 4. handleError — missing condition update (setDegradedCondition)
# 5. handleDeletion — stale CR / missing re-fetch before finalizer removal
```

### Acceptance Criteria

- [ ] Issue 1 detected: non-idempotent Service creation (Critical)
- [ ] Issue 2 detected: missing owner reference on ConfigMap (Critical)
- [ ] Issue 3 detected: wildcard RBAC verbs (Critical)
- [ ] Issue 4 detected: missing status condition in error path (Warning)
- [ ] Issue 5 detected: stale CR in finalizer removal (Critical)
- [ ] Each finding has file path and line reference
- [ ] Each finding has a concrete fix suggestion
- [ ] No false positives on correct code
- [ ] Automated scripts run (validate-api-types, validate-rbac, check-idempotency)

---

## Test 6.2 — Review Clean Operator

### Step 1: Ensure database-operator exists

```bash
test -d ../go-operator/operators/database-operator/ && echo "PASS" || echo "FAIL: database-operator not found"
```

### Step 2: Prompt

```
Using the operator-reviewer agent, review the database-operator at 
go-operator/operators/database-operator/ for best practices.
```

### Step 3: Verify

```bash
# Review should:
# - Run all 3 validation scripts
# - Not report any false Critical findings
# - May report genuine Warnings (e.g., missing print columns, no specDescriptors)
# - Complete without errors
```

### Acceptance Criteria

- [ ] No false Critical findings on production code
- [ ] Warnings are genuine improvement suggestions
- [ ] Review completes without errors
- [ ] All 3 automated scripts run

---

## Test I-6 — Review Then Fix (Integration)

### Step 1: Ensure Test 6.1 flawed operator exists

### Step 2: Prompt

```
Review the operator at /tmp/redis-operator-flawed/. For any Critical 
findings, fix them using the implementing-reconciliation skill patterns.
```

### Step 3: Verify

```bash
# After fixes:
# 1. reconcileService should have Get+IsNotFound guard
# 2. reconcileConfigMap should have SetControllerReference
# 3. RBAC should have explicit verbs (no wildcard)
# 4. handleDeletion should re-fetch CR before removing finalizer

# Run validation scripts to confirm fixes
python3 .claude/skills/implementing-reconciliation/scripts/check-idempotency.py \
  /tmp/redis-operator-flawed/internal/controller/rediscluster_reconcilers.go

python3 .claude/skills/implementing-reconciliation/scripts/validate-rbac-annotations.py \
  /tmp/redis-operator-flawed/internal/controller/rediscluster_controller.go

# Should compile
cd /tmp/redis-operator-flawed && go build -o bin/manager ./cmd/main.go
```

### Acceptance Criteria

- [ ] Review identifies Critical issues
- [ ] Fixes follow implementing-reconciliation skill patterns
- [ ] check-idempotency.py passes after fixes
- [ ] validate-rbac-annotations.py passes after fixes
- [ ] Code compiles after fixes
- [ ] Re-review shows Critical issues resolved

---

## Cleanup

```bash
rm -rf /tmp/redis-operator-flawed
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| Script not found | Wrong path to skill scripts | Use `.claude/skills/<skill>/scripts/` |
| False positive on production code | Script too strict | Check if pattern is genuinely bad or acceptable variant |
| Review misses planted issue | Agent didn't inspect that file | Ensure prompt mentions "Check the controller, API types, and RBAC" |
| Flawed operator doesn't compile | Planted issue broke imports | Ensure planted issues are logic bugs, not syntax errors |
