# Webhook Patterns

## Three Webhook Types

### Defaulting Webhook (Mutating)
Sets default values programmatically. Use when defaults are computed, conditional, or depend on other fields.

```go
//+kubebuilder:webhook:path=/mutate-<group-path>-<version>-<kind>,mutating=true,failurePolicy=fail,sideEffects=None,groups=<group>,resources=<plural>,verbs=create;update,versions=<version>,name=m<kind>.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &RedisCluster{}

func (r *RedisCluster) Default() {
    if r.Spec.Replicas == 0 {
        r.Spec.Replicas = 3
    }
    if r.Spec.Version == "" {
        r.Spec.Version = "7.2"
    }
}
```

### Validating Webhook
Custom validation beyond kubebuilder markers. Use for cross-field rules, OneOf patterns, business logic.

```go
//+kubebuilder:webhook:path=/validate-<group-path>-<version>-<kind>,mutating=false,failurePolicy=fail,sideEffects=None,groups=<group>,resources=<plural>,verbs=create;update,versions=<version>,name=v<kind>.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &RedisCluster{}

func (r *RedisCluster) ValidateCreate() (admission.Warnings, error) {
    return r.validate()
}

func (r *RedisCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
    return r.validate()
}

func (r *RedisCluster) ValidateDelete() (admission.Warnings, error) {
    return nil, nil
}

func (r *RedisCluster) validate() (admission.Warnings, error) {
    if r.Spec.Replicas > 7 {
        return nil, fmt.Errorf("replicas must not exceed 7, got %d", r.Spec.Replicas)
    }
    return nil, nil
}
```

### Conversion Webhook (Pattern I)
Converts between API versions. Uses hub-and-spoke pattern where one version is the "hub" and others convert to/from it.

```go
//+kubebuilder:webhook:path=/convert,mutating=false,failurePolicy=fail,sideEffects=None,name=cversion.kb.io,admissionReviewVersions=v1
```

## SetupWebhookWithManager

```go
func (r *RedisCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
    return ctrl.NewWebhookManagedBy(mgr).
        For(r).
        Complete()
}
```

## Main.go Registration

```go
if err = (&cachev1alpha1.RedisCluster{}).SetupWebhookWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create webhook", "webhook", "RedisCluster")
    os.Exit(1)
}
```

## Config Files Required

| File | Purpose |
|------|---------|
| `config/webhook/service.yaml` | Service exposing webhook on 443→9443 |
| `config/webhook/kustomization.yaml` | Combines manifests + service |
| `config/webhook/kustomizeconfig.yaml` | Kustomize substitution rules |
| `config/certmanager/certificate.yaml` | Self-signed TLS certificate |
| `config/certmanager/kustomization.yaml` | Cert-manager resources |
| `config/default/manager_webhook_patch.yaml` | Adds port 9443 + cert volume |
| `config/default/webhookcainjection_patch.yaml` | CA injection annotations |
| `config/crd/patches/webhook_in_<kind>.yaml` | CRD conversion config |

## Webhook Path Convention

- Mutating: `/mutate-<group-with-dashes>-<version>-<kind>`
- Validating: `/validate-<group-with-dashes>-<version>-<kind>`
- Conversion: `/convert`

Example: `/mutate-cache-redis-example-com-v1alpha1-rediscluster`
