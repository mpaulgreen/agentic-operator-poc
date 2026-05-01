# E2E Test Patterns

E2E tests run against a real cluster (Kind, OpenShift). They verify the operator works end-to-end.

## Structure

```go
package e2e

var _ = Describe("controller", Ordered, func() {
    BeforeAll(func() {
        // One-time cluster setup
        cmd := exec.Command("kubectl", "create", "ns", namespace)
        _, _ = utils.Run(cmd)
    })

    AfterAll(func() {
        cmd := exec.Command("kubectl", "delete", "ns", namespace)
        _, _ = utils.Run(cmd)
    })

    It("should deploy operator successfully", func() {
        By("building the image")
        cmd := exec.Command("make", "docker-build", "IMG="+image)
        _, err := utils.Run(cmd)
        Expect(err).NotTo(HaveOccurred())

        By("deploying the controller")
        cmd = exec.Command("make", "deploy", "IMG="+image)
        _, err = utils.Run(cmd)
        Expect(err).NotTo(HaveOccurred())

        By("verifying pod is running")
        Eventually(func() error {
            cmd := exec.Command("kubectl", "get", "pods", "-n", namespace,
                "-l", "control-plane=controller-manager", "-o", "jsonpath={.items[0].status.phase}")
            out, err := utils.Run(cmd)
            if string(out) != "Running" { return fmt.Errorf("not running") }
            return err
        }, time.Minute, time.Second).Should(Succeed())
    })
})
```

## Key Differences from Unit Tests

| Aspect | Unit (envtest) | E2E |
|--------|---------------|-----|
| Cluster | Embedded API server | Real Kind/OpenShift |
| Pods run | No | Yes |
| Speed | Fast (~seconds) | Slow (~minutes) |
| Package | `controller` | `e2e` (separate) |
| Interaction | Go client | `kubectl` commands |
| Ordering | Parallel OK | `Ordered` (sequential) |

## When to Use E2E

- Verify operator deploys and runs in a real cluster
- Test webhook behavior (needs real cert-manager)
- Test CRD conversion with multiple versions
- Verify RBAC works with real ServiceAccounts
- Integration with external systems (Prometheus, Istio)

## E2E is Optional for Skill Testing

The testing-operator skill focuses on **unit/integration tests** (envtest). E2E tests require a real cluster and are typically run in CI, not during skill validation.
