# Test Scenarios

## Per-Reconciler Method (Minimum 2, Recommended 3)

For each `reconcileX()` method, test:

### 1. Create When Absent (Required)
```go
It("should create Secret when absent", func() {
    err := reconciler.reconcileSecret(ctx, cr)
    Expect(err).NotTo(HaveOccurred())

    secret := &corev1.Secret{}
    Expect(k8sClient.Get(ctx, key, secret)).To(Succeed())
    Expect(secret.Data).To(HaveKey("password"))
})
```

### 2. Idempotent When Exists (Required)
```go
It("should not recreate existing Secret", func() {
    Expect(reconciler.reconcileSecret(ctx, cr)).To(Succeed())
    secret := &corev1.Secret{}
    Expect(k8sClient.Get(ctx, key, secret)).To(Succeed())
    original := string(secret.Data["password"])

    Expect(reconciler.reconcileSecret(ctx, cr)).To(Succeed())
    Expect(k8sClient.Get(ctx, key, secret)).To(Succeed())
    Expect(string(secret.Data["password"])).To(Equal(original))
})
```

### 3. Update When Spec Changes (For mutable resources)
```go
It("should update StatefulSet replicas when spec changes", func() {
    Expect(reconciler.reconcileStatefulSet(ctx, cr)).To(Succeed())
    cr.Spec.Replicas = 5
    Expect(reconciler.reconcileStatefulSet(ctx, cr)).To(Succeed())

    sts := &appsv1.StatefulSet{}
    Expect(k8sClient.Get(ctx, key, sts)).To(Succeed())
    Expect(*sts.Spec.Replicas).To(Equal(int32(5)))
})
```

## Controller Lifecycle Tests

### Finalizer Added
```go
It("should add finalizer", func() {
    reconciler.Reconcile(ctx, req)
    Expect(k8sClient.Get(ctx, key, cr)).To(Succeed())
    Expect(cr.Finalizers).To(ContainElement("my.group/finalizer"))
})
```

### Resources Created in Order
```go
It("should create all resources", func() {
    reconciler.Reconcile(ctx, req)
    // Verify each resource exists
    Expect(k8sClient.Get(ctx, secretKey, &corev1.Secret{})).To(Succeed())
    Expect(k8sClient.Get(ctx, svcKey, &corev1.Service{})).To(Succeed())
    Expect(k8sClient.Get(ctx, stsKey, &appsv1.StatefulSet{})).To(Succeed())
})
```

### Owner References Set
```go
It("should set owner references", func() {
    reconciler.Reconcile(ctx, req)
    secret := &corev1.Secret{}
    Expect(k8sClient.Get(ctx, key, secret)).To(Succeed())
    Expect(secret.OwnerReferences).To(HaveLen(1))
    Expect(secret.OwnerReferences[0].Name).To(Equal(cr.Name))
})
```

## Helper Function Tests

```go
It("should generate password of correct length", func() {
    Expect(len(generatePassword(16))).To(Equal(16))
})

It("should return correct labels", func() {
    labels := labelsForRedisCluster("test")
    Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/instance", "test"))
})
```
