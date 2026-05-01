# envtest Setup

envtest provides a real Kubernetes API server + etcd for testing controllers without a full cluster. No kubelet — Pods won't run, but CRUD on API objects works.

## suite_test.go Structure

```go
var (
    cfg       *rest.Config
    k8sClient client.Client
    testEnv   *envtest.Environment
)

func TestControllers(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
    logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

    testEnv = &envtest.Environment{
        CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
        ErrorIfCRDPathMissing: true,
    }

    var err error
    cfg, err = testEnv.Start()
    Expect(err).NotTo(HaveOccurred())

    err = myv1alpha1.AddToScheme(scheme.Scheme)
    Expect(err).NotTo(HaveOccurred())

    k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
    Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
    err := testEnv.Stop()
    Expect(err).NotTo(HaveOccurred())
})
```

## CRD Path

Tests run from `internal/controller/`, so CRDs are at `../../config/crd/bases/`. Must exist — `ErrorIfCRDPathMissing: true` prevents silent failures.

## Running Tests

```bash
# Via Makefile (downloads envtest binaries automatically)
make test

# Directly (requires KUBEBUILDER_ASSETS)
KUBEBUILDER_ASSETS="$(setup-envtest use 1.29.0 -p path)" go test ./internal/controller/... -v
```

## Limitations

- No kubelet: Pods, ReplicaSets won't be created from Deployments/StatefulSets
- No built-in controllers: Deployment controller won't update status
- `ReadyReplicas` stays 0 — test that objects are created, not that they're ready
- Can't test webhooks without extra setup (WebhookInstallOptions)
