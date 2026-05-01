---
name: testing-operator
description: >
  Generates unit, integration, and E2E test suites for operator controllers. Use when
  user asks to write tests, test an operator, add test coverage, generate controller
  tests, create test suite, or test reconciliation logic.
---

# Testing Operator Controllers

Generate comprehensive test suites for operator controllers using envtest and Ginkgo/Gomega. Covers envtest setup, controller lifecycle tests, per-reconciler idempotency tests, helper function tests, and E2E test skeletons.

## Test File Organization

A well-tested controller has these test files:

| File | Content | Tests |
|------|---------|-------|
| `suite_test.go` | envtest setup, Ginkgo entry | Environment, scheme, client |
| `<kind>_controller_test.go` | Reconcile lifecycle | Finalizer, multi-resource create, phase transitions |
| `<kind>_reconcilers_test.go` | Per-method tests | Create/idempotent/error for each reconcileX |
| `<kind>_helpers_test.go` | Utility tests | Labels, password generation |

## Workflow A: Generate Full Test Suite

Use when the user has a controller with reconciler methods and needs a complete test suite.

1. **Read the controller files** to identify:
   - All `reconcileX()` methods (from `_reconcilers.go`)
   - The reconciler struct fields (Client, Scheme, Recorder)
   - What resources each method creates (Secret, Service, StatefulSet, etc.)
   - Status update logic (from `_status.go`)
   - Helper functions (from `_helpers.go`)

2. **Generate `suite_test.go`** — envtest setup:
   - Global vars: `cfg *rest.Config`, `k8sClient client.Client`, `testEnv *envtest.Environment`
   - `TestControllers(t)` — Ginkgo entry point
   - `BeforeSuite` — create envtest.Environment with CRD paths, start, register scheme, create client
   - `AfterSuite` — stop envtest
   - See `assets/templates/suite_test.go.tmpl` and `references/envtest-setup.md`

3. **Generate `<kind>_controller_test.go`** — lifecycle tests:
   - Context: "When reconciling a <Kind>"
   - BeforeEach: create test CR + reconciler with `record.NewFakeRecorder(100)`
   - AfterEach: clean up test CR
   - Tests:
     - "should add finalizer on first reconciliation"
     - "should create all managed resources"
     - "should be idempotent on second reconciliation"
     - "should handle deletion with finalizer cleanup"
   - See `assets/templates/controller_test.go.tmpl`

4. **Generate per-reconciler tests** — for each `reconcileX()` method:
   - "should create <Resource> when absent"
   - "should not recreate existing <Resource>" (idempotency)
   - "should handle <Resource> creation error gracefully" (optional)
   - Verify resource data content, not just existence
   - See `assets/templates/reconciler_test.go.tmpl`

5. **Generate helper tests** — for utility functions:
   - `labelsFor<Kind>()` — verify label keys and values
   - `generatePassword()` — verify length, character set

6. **Verify** with `scripts/generate-test-matrix.py`.

## Workflow B: Generate Tests for Single New Method

Use when the user added a new `reconcileX()` method and needs tests for just that method.

1. Read the new method to understand what resource it creates.
2. Add test cases to existing `_controller_test.go` or `_reconcilers_test.go`:
   - "should create <Resource>"
   - "should not recreate existing <Resource>"
3. Do NOT modify or remove existing tests.

## envtest Limitations

envtest runs a real API server + etcd but has **no kubelet**:
- Pods won't actually run
- Deployments won't create ReplicaSets
- StatefulSets won't create Pods
- `ReadyReplicas` will stay 0

Tests verify the **reconciler creates the right objects** with correct specs, not that they become "Ready". Status tests must account for this.

## Key Patterns

### FakeRecorder for Events
```go
reconciler := &MyReconciler{
    Client:   k8sClient,
    Scheme:   k8sClient.Scheme(),
    Recorder: record.NewFakeRecorder(100),
}
```

### Eventually for Async Assertions
```go
Eventually(func() bool {
    secret := &corev1.Secret{}
    err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, secret)
    return err == nil
}, timeout, interval).Should(BeTrue())
```

### Idempotency Test Pattern
```go
It("should not recreate existing Secret", func() {
    err := reconciler.reconcileSecret(ctx, cr)
    Expect(err).NotTo(HaveOccurred())
    // Get original
    secret := &corev1.Secret{}
    Expect(k8sClient.Get(ctx, key, secret)).To(Succeed())
    original := string(secret.Data["password"])
    // Reconcile again
    err = reconciler.reconcileSecret(ctx, cr)
    Expect(err).NotTo(HaveOccurred())
    // Verify unchanged
    Expect(k8sClient.Get(ctx, key, secret)).To(Succeed())
    Expect(string(secret.Data["password"])).To(Equal(original))
})
```
