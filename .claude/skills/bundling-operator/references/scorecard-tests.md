# Scorecard Tests

The operator-sdk scorecard runs automated tests against an operator bundle. Tests execute as containers in the target cluster.

## Test Suites

### Basic Suite

| Test | What It Checks | Common Failure |
|------|---------------|----------------|
| `basic-check-spec` | CSV has required fields, valid structure | Missing displayName, description, or icon |

### OLM Suite

| Test | What It Checks | Common Failure |
|------|---------------|----------------|
| `olm-bundle-validation` | Bundle directory structure, annotations | Missing metadata/annotations.yaml |
| `olm-crds-have-validation` | CRDs have OpenAPI validation schemas | CRD generated without markers |
| `olm-crds-have-resources` | alm-examples references all owned CRDs | Missing CRD kind in alm-examples |
| `olm-spec-descriptors` | specDescriptors cover Spec fields | Missing descriptors (warning) |
| `olm-status-descriptors` | statusDescriptors cover Status fields | Missing descriptors (warning) |

## Scorecard Config Structure

```yaml
apiVersion: scorecard.operatorframework.io/v1alpha3
kind: Configuration
metadata:
  name: config
stages:
- parallel: true          # Run all tests in parallel
  tests:
  - entrypoint:
    - scorecard-test      # Container command
    - basic-check-spec    # Test name
    image: quay.io/operator-framework/scorecard-test:v1.37.0
    labels:
      suite: basic
      test: basic-check-spec-test
    storage:
      spec:
        mountPath: {}
```

## Running Scorecard

```bash
# Run all tests
operator-sdk scorecard bundle/ --kubeconfig ~/.kube/config

# Run specific suite
operator-sdk scorecard bundle/ --selector=suite=olm

# Output as JSON
operator-sdk scorecard bundle/ --output json
```

## Common Fixes

| Issue | Fix |
|-------|-----|
| `olm-spec-descriptors` fails | Add specDescriptors to each owned CRD in CSV |
| `olm-status-descriptors` fails | Add statusDescriptors to each owned CRD in CSV |
| `olm-crds-have-resources` fails | Add sample CR for each CRD kind to alm-examples |
| `basic-check-spec` fails | Ensure displayName, description, icon fields exist |
| `olm-crds-have-validation` fails | Regenerate CRD with kubebuilder markers (`make manifests`) |

## Pre-flight Checks

Before running scorecard on a live cluster, use `check-scorecard-readiness.py` to catch common issues locally:
1. Verify scorecard config has required tests
2. Verify alm-examples covers all owned CRDs
3. Verify descriptor paths match CRD schema
