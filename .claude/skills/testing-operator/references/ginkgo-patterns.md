# Ginkgo/Gomega Test Patterns

## Structure

```go
var _ = Describe("RedisCluster Controller", func() {
    const (
        timeout  = time.Second * 10
        interval = time.Millisecond * 250
    )

    Context("When reconciling a RedisCluster", func() {
        var (
            ctx        context.Context
            cr         *v1alpha1.RedisCluster
            reconciler *RedisClusterReconciler
        )

        BeforeEach(func() {
            ctx = context.Background()
            cr = &v1alpha1.RedisCluster{...}
            reconciler = &RedisClusterReconciler{
                Client:   k8sClient,
                Scheme:   k8sClient.Scheme(),
                Recorder: record.NewFakeRecorder(100),
            }
            Expect(k8sClient.Create(ctx, cr)).To(Succeed())
        })

        AfterEach(func() {
            resource := &v1alpha1.RedisCluster{}
            if err := k8sClient.Get(ctx, key, resource); err == nil {
                Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
            }
        })

        It("should create resources", func() { ... })
    })
})
```

## Key Constructs

| Construct | Purpose |
|-----------|---------|
| `Describe()` | Top-level test suite |
| `Context()` | Group related tests |
| `It()` | Individual test case |
| `BeforeEach()` | Per-test setup |
| `AfterEach()` | Per-test cleanup |
| `By()` | Narrative step in test output |
| `Eventually()` | Poll until condition met or timeout |
| `Expect()` | Assertion |

## FakeRecorder

```go
recorder := record.NewFakeRecorder(100) // buffer for 100 events
```

Captures events without needing a real event sink.

## Eventually Pattern

```go
Eventually(func() bool {
    obj := &corev1.Secret{}
    err := k8sClient.Get(ctx, key, obj)
    return err == nil
}, timeout, interval).Should(BeTrue())
```

Use for anything that happens asynchronously (resource creation, status updates).

## Unique Test Names

Use timestamps or random suffixes to avoid name collisions:
```go
name := fmt.Sprintf("test-cluster-%d", time.Now().UnixNano())
```
