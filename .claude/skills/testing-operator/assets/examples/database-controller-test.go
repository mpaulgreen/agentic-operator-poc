// Example: Controller test patterns from database-operator.
// Shows the key testing patterns for operator reconciliation.

package controller

// Test file organization (7 files, ~2200 lines):
//   suite_test.go              — envtest setup
//   controller_test.go         — reconcile lifecycle, finalizers
//   reconcilers_test.go        — per-method create/idempotent tests
//   resources_test.go          — builder function verification
//   status_test.go             — condition updates, phase transitions
//   conditions_test.go         — setCondition, findCondition
//   helpers_test.go            — labels, password generation
//
// Key patterns:
//
// 1. FakeRecorder:
//    reconciler := &Reconciler{
//        Client:   k8sClient,
//        Scheme:   k8sClient.Scheme(),
//        Recorder: record.NewFakeRecorder(100),
//    }
//
// 2. Unique names per test:
//    name := fmt.Sprintf("test-cluster-%d", time.Now().UnixNano())
//
// 3. Create + multiple reconciliations:
//    Expect(k8sClient.Create(ctx, cr)).To(Succeed())
//    for i := 0; i < 3; i++ {
//        _, _ = reconciler.Reconcile(ctx, req)
//    }
//
// 4. Verify child resource exists and has correct data:
//    secret := &corev1.Secret{}
//    Expect(k8sClient.Get(ctx, key, secret)).To(Succeed())
//    Expect(secret.Data).To(HaveKey("password"))
//
// 5. Idempotency — reconcile twice, verify unchanged:
//    reconciler.reconcileSecret(ctx, cr)
//    original := secret.Data["password"]
//    reconciler.reconcileSecret(ctx, cr)
//    Expect(secret.Data["password"]).To(Equal(original))
//
// 6. Finalizer verification:
//    reconciler.Reconcile(ctx, req)
//    Expect(cr.Finalizers).To(ContainElement(finalizerName))
//
// 7. Owner reference verification:
//    Expect(secret.OwnerReferences).To(HaveLen(1))
//    Expect(secret.OwnerReferences[0].Name).To(Equal(cr.Name))
//
// 8. envtest limitation — ReadyReplicas stays 0:
//    // Can verify StatefulSet was created with correct spec
//    // Cannot verify pods are running (no kubelet)
//    Expect(*sts.Spec.Replicas).To(Equal(int32(3)))  // OK
//    // sts.Status.ReadyReplicas will be 0             // expected
