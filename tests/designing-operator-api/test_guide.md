# Sprint 2 Test Guide: `designing-operator-api` Skill

## Prerequisites

- Go 1.22+ installed
- operator-sdk v1.37.0+ installed (for SDK comparison tests)
- A scaffolded operator project at `/tmp/redis-operator-test/` (run Test 1.1 from scaffolding-operator first)
- The skill is at `.claude/skills/designing-operator-api/`

## Test Order

1. **2.1**: Simple CRD design — Workflow A (basic fields + markers)
2. **2.2**: Complex CRD with nested types — Workflow B
3. **I-1.2**: Integration — scaffold + design end-to-end
4. **2.3**: SDK types comparison
5. **2.4**: Add webhooks — Workflow C
6. **2.5**: SDK webhook comparison
7. **2.6**: Add API version — Workflow D

---

## Test 2.1 — Simple CRD Design (Workflow A)

### Step 1: Ensure a scaffolded project exists

Use the project from Test 1.1 at `/tmp/redis-operator-test/`, or scaffold a fresh one.

### Step 2: Prompt

```
Using the designing-operator-api skill, design a CRD for RedisCluster with 
these requirements:
- Spec: replicas (3-7, default 3), version (enum: 6.0/7.0/7.2, default 7.2), 
  storage size (pattern: ^[0-9]+[KMGT]i$), sentinel enabled (default true)
- Status: phase (Pending/Initializing/Running/Failed), readyReplicas, 
  conditions (Available/Progressing/Degraded), endpoint string
- Print columns: Phase, Ready, Version, Age
Generate api/v1alpha1/rediscluster_types.go at /tmp/redis-operator-test/ 
with proper kubebuilder markers.
```

### Step 3: Verify

```bash
# Validation script
python3 .claude/skills/designing-operator-api/scripts/validate-api-types.py \
  /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go

# Specific marker checks
grep 'kubebuilder:validation:Minimum=3' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: Minimum" || echo "FAIL"
grep 'kubebuilder:validation:Maximum=7' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: Maximum" || echo "FAIL"
grep 'kubebuilder:validation:Enum=' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: Enum" || echo "FAIL"
grep 'kubebuilder:default=3' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: Default replicas" || echo "FAIL"
grep 'kubebuilder:default=true' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: Default sentinel" || echo "FAIL"
grep 'kubebuilder:validation:Pattern=' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: Pattern" || echo "FAIL"
grep 'metav1.Condition' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: Conditions" || echo "FAIL"
grep -c 'printcolumn' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go | xargs -I{} echo "Print columns: {}"

# Compilation (regenerate DeepCopy first)
cd /tmp/redis-operator-test && go build ./api/...
```

### Acceptance Criteria

- [ ] validate-api-types.py passes (0 errors)
- [ ] Spec has: replicas (Minimum=3, Maximum=7, default=3), version (Enum, default), storage (Pattern), sentinel (default=true)
- [ ] Status has: phase (Enum), readyReplicas (int32), conditions ([]metav1.Condition), endpoint (string)
- [ ] At least 4 print columns (Phase, Ready, Version, Age)
- [ ] All fields have json tags
- [ ] Code compiles (`go build ./api/...`)

---

## Test 2.2 — Complex CRD with Nested Types (Workflow B)

### Step 1: Ensure Test 2.1 is complete

### Step 2: Prompt

```
Using the designing-operator-api skill, extend the RedisCluster CRD at 
/tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go to add:
- StorageSpec nested type (size string with pattern ^[0-9]+[KMGT]i$, 
  storageClassName *string optional)
- Use corev1.ResourceRequirements for resources (embedded K8s type)
- BackupSpec nested type (schedule cron string with pattern, 
  retentionDays 1-30 default 7, destination enum s3/local default local)
- Make backup optional (pointer type)
Update the Spec to use these nested types instead of flat storage field.
```

### Step 3: Verify

```bash
# Nested types exist
grep 'type StorageSpec struct' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: StorageSpec" || echo "FAIL"
grep 'type BackupSpec struct' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: BackupSpec" || echo "FAIL"

# Embedded K8s type
grep 'corev1.ResourceRequirements' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: ResourceRequirements" || echo "FAIL"

# Optional pointer
grep 'Backup \*BackupSpec' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: optional backup" || echo "FAIL"

# Backup markers
grep 'Minimum=1' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: retention min" || echo "FAIL"
grep 'Maximum=30' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: retention max" || echo "FAIL"
grep 'Enum=.*s3.*local' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: destination enum" || echo "FAIL"

# Validation script
python3 .claude/skills/designing-operator-api/scripts/validate-api-types.py \
  /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go

# Compilation
cd /tmp/redis-operator-test && go build ./api/...
```

### Acceptance Criteria

- [ ] StorageSpec struct with size (Pattern) and storageClassName (*string)
- [ ] BackupSpec struct with schedule (Pattern), retentionDays (Min=1, Max=30, default=7), destination (Enum)
- [ ] Backup field is `*BackupSpec` (pointer, optional)
- [ ] Resources uses `corev1.ResourceRequirements` (import present)
- [ ] validate-api-types.py passes
- [ ] Code compiles

---

## Test I-1.2 — Integration: Scaffold then Design

### Step 1: Clean up

```bash
rm -rf /tmp/mq-operator-test
```

### Step 2: Prompt (two-step)

```
Step 1: Using the scaffolding-operator skill, create a new operator project 
called 'message-queue-operator' with domain 'messaging.example.com' and 
group 'queue'. The operator will manage MessageQueue resources. Generate 
under /tmp/mq-operator-test/

Step 2: Using the designing-operator-api skill, design the MessageQueue CRD 
with these fields:
- Spec: queueType (enum: kafka/rabbitmq/redis), replicas (1-10, default 1), 
  persistentStorage (bool, default false), retentionHours (1-720, default 24)
- Status: phase (Pending/Running/Failed), readyReplicas, conditions, 
  endpoint string
- Print columns: Phase, Type, Replicas, Age
```

### Step 3: Verify

```bash
# Scaffolding valid
bash .claude/skills/scaffolding-operator/scripts/validate-project-structure.sh /tmp/mq-operator-test/

# Types valid
python3 .claude/skills/designing-operator-api/scripts/validate-api-types.py \
  /tmp/mq-operator-test/api/v1alpha1/messagequeue_types.go

# End-to-end compilation
cd /tmp/mq-operator-test && go mod tidy && go build ./...
```

### Acceptance Criteria

- [ ] Project structure valid (48/48 scaffolding checks)
- [ ] Types valid (validation script passes)
- [ ] PROJECT file matches types (same group, version, kind)
- [ ] Whole project compiles end-to-end
- [ ] Spec has all requested fields with correct markers

---

## Test 2.3 — SDK Comparison

### Step 1: Generate types with operator-sdk

The SDK generates stub types with a `Foo` example field. Our skill generates types with actual user-requested fields and markers. The comparison focuses on structure, not content.

```bash
rm -rf /tmp/types-sdk-test
mkdir -p /tmp/types-sdk-test && cd /tmp/types-sdk-test
operator-sdk init --domain redis.example.com --repo github.com/example/redis-operator --plugins=go/v4
operator-sdk create api --group cache --version v1alpha1 --kind RedisCluster --resource --controller
```

### Step 2: Compare structural elements

```bash
echo "=== Root type markers ==="
echo "SDK:" && grep '+kubebuilder:object\|+kubebuilder:subresource' /tmp/types-sdk-test/api/v1alpha1/rediscluster_types.go
echo "SKILL:" && grep '+kubebuilder:object\|+kubebuilder:subresource' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go

echo ""
echo "=== Status struct ==="
echo "SDK has Conditions:" && grep -c 'Condition' /tmp/types-sdk-test/api/v1alpha1/rediscluster_types.go
echo "SKILL has Conditions:" && grep -c 'Condition' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go

echo ""
echo "=== Validation markers ==="
echo "SDK markers:" && grep -c 'kubebuilder:validation' /tmp/types-sdk-test/api/v1alpha1/rediscluster_types.go
echo "SKILL markers:" && grep -c 'kubebuilder:validation' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go

echo ""
echo "=== Print columns ==="
echo "SDK columns:" && grep -c 'printcolumn' /tmp/types-sdk-test/api/v1alpha1/rediscluster_types.go
echo "SKILL columns:" && grep -c 'printcolumn' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go
```

### Expected Differences

| Aspect | SDK | Skill | Why |
|--------|-----|-------|-----|
| Spec fields | `Foo string` (example) | Actual fields with markers | Skill designs from requirements |
| Status fields | Empty | Phase, conditions, counters | Skill includes production patterns |
| Validation markers | 0 | Multiple (Minimum, Maximum, Enum, etc.) | Skill adds real validation |
| Print columns | 0 | 4+ | Skill adds kubectl visibility |
| Conditions | Not present | `[]metav1.Condition` | Skill follows K8s conventions |
| Nested types | None | StorageSpec, BackupSpec | Skill groups related fields |

### Acceptance Criteria

- [ ] Both compile
- [ ] Skill output has significantly more markers and structure than SDK stub
- [ ] Skill output has conditions in Status (SDK doesn't)
- [ ] Skill output has print columns (SDK doesn't)
- [ ] Root markers match (object:root, subresource:status)

---

---

## Test 2.4 — Add Webhooks (Workflow C)

### Step 1: Ensure Test 2.2 project exists

### Step 2: Prompt

```
Using the designing-operator-api skill, add validating and defaulting webhooks 
to the RedisCluster CRD at /tmp/redis-operator-test/. The defaulting webhook 
should set replicas=3 and version=7.2 if not specified. The validating webhook 
should reject replicas > 7.
Generate the webhook handler, all config files (webhook, certmanager, patches), 
and update main.go.
```

### Step 3: Verify

```bash
# Webhook handler
test -f /tmp/redis-operator-test/api/v1alpha1/rediscluster_webhook.go && echo "PASS" || echo "FAIL"
grep 'webhook.Defaulter' /tmp/redis-operator-test/api/v1alpha1/rediscluster_webhook.go && echo "PASS: Defaulter" || echo "FAIL"
grep 'webhook.Validator' /tmp/redis-operator-test/api/v1alpha1/rediscluster_webhook.go && echo "PASS: Validator" || echo "FAIL"

# Config files (9 total)
for f in config/webhook/service.yaml config/webhook/kustomization.yaml config/webhook/kustomizeconfig.yaml config/certmanager/certificate.yaml config/certmanager/kustomization.yaml config/certmanager/kustomizeconfig.yaml config/default/manager_webhook_patch.yaml config/default/webhookcainjection_patch.yaml config/crd/patches/webhook_in_redisclusters.yaml; do
  test -f /tmp/redis-operator-test/$f && echo "PASS: $f" || echo "FAIL: $f"
done

# main.go
grep 'SetupWebhookWithManager' /tmp/redis-operator-test/cmd/main.go && echo "PASS: webhook in main.go" || echo "FAIL"

# Compilation
cd /tmp/redis-operator-test && go build -o bin/manager ./cmd/main.go && echo "PASS" || echo "FAIL"
```

### Acceptance Criteria

- [ ] `rediscluster_webhook.go` exists with Default(), ValidateCreate/Update/Delete()
- [ ] 9 config files created (webhook/, certmanager/, patches)
- [ ] main.go updated with SetupWebhookWithManager() call
- [ ] Compiles with webhook handler

---

## Test 2.5 — SDK Webhook Comparison

### Step 1: Create SDK webhook project

```bash
rm -rf /tmp/webhook-test
mkdir -p /tmp/webhook-test && cd /tmp/webhook-test
operator-sdk init --domain example.com --repo github.com/example/webhook-test --plugins=go/v4
operator-sdk create api --group cache --version v1alpha1 --kind Redis --resource --controller
operator-sdk create webhook --group cache --version v1alpha1 --kind Redis --defaulting --programmatic-validation
```

### Step 2: Compare

```bash
# Both have same structure
echo "Methods: SDK=$(grep -c 'func.*Default\|func.*Validate\|func.*SetupWebhook' /tmp/webhook-test/api/v1alpha1/redis_webhook.go) SKILL=$(grep -c 'func.*Default\|func.*Validate\|func.*SetupWebhook' /tmp/redis-operator-test/api/v1alpha1/rediscluster_webhook.go)"
echo "Markers: SDK=$(grep -c 'kubebuilder:webhook' /tmp/webhook-test/api/v1alpha1/redis_webhook.go) SKILL=$(grep -c 'kubebuilder:webhook' /tmp/redis-operator-test/api/v1alpha1/rediscluster_webhook.go)"

# Config files present in both
for f in config/webhook/service.yaml config/webhook/kustomization.yaml config/webhook/kustomizeconfig.yaml config/certmanager/certificate.yaml config/certmanager/kustomization.yaml config/default/manager_webhook_patch.yaml config/default/webhookcainjection_patch.yaml; do
  sdk=$(test -f /tmp/webhook-test/$f && echo "Y" || echo "N")
  skill=$(test -f /tmp/redis-operator-test/$f && echo "Y" || echo "N")
  printf "  %-50s SDK:%-1s SKILL:%-1s\n" "$f" "$sdk" "$skill"
done

# main.go webhook registration
grep 'SetupWebhookWithManager' /tmp/webhook-test/cmd/main.go > /dev/null && echo "SDK: webhook in main.go" || echo "SDK: MISSING"
grep 'SetupWebhookWithManager' /tmp/redis-operator-test/cmd/main.go > /dev/null && echo "SKILL: webhook in main.go" || echo "SKILL: MISSING"

# Both compile
cd /tmp/webhook-test && go build -o bin/manager ./cmd/main.go && echo "SDK: PASS" || echo "SDK: FAIL"
cd /tmp/redis-operator-test && go build -o bin/manager ./cmd/main.go && echo "SKILL: PASS" || echo "SKILL: FAIL"
```

### Expected Differences

| Aspect | SDK | Skill | Why |
|--------|-----|-------|-----|
| Default() body | Empty (TODO) | Sets replicas + version | Skill implements real defaults |
| Validate*() body | Empty (TODO) | Checks replicas > 7 | Skill implements real validation |
| Config files | All present | All present | Match |
| Test files | webhook_test.go, suite_test.go | Not generated | Sprint 4 |

---

## Test 2.6 — Add API Version (Workflow D)

Tests version progression: creating a v1beta1 version of an existing v1alpha1 CRD.

### Step 1: Ensure Test 2.4 project exists

### Step 2: Prompt

```
Using the designing-operator-api skill, add a new API version v1beta1 for 
the RedisCluster CRD at /tmp/redis-operator-test/. The v1beta1 version should 
be the new storage version. Copy the same Spec/Status fields from v1alpha1 
but add a new field 'maxMemory string' to the v1beta1 Spec. Generate the 
new version directory with groupversion_info.go, types, and DeepCopy.
```

### Step 3: Verify

```bash
# New version directory exists
test -d /tmp/redis-operator-test/api/v1beta1 && echo "PASS: api/v1beta1/ exists" || echo "FAIL"
test -f /tmp/redis-operator-test/api/v1beta1/groupversion_info.go && echo "PASS: groupversion_info" || echo "FAIL"
test -f /tmp/redis-operator-test/api/v1beta1/rediscluster_types.go && echo "PASS: types" || echo "FAIL"
test -f /tmp/redis-operator-test/api/v1beta1/zz_generated.deepcopy.go && echo "PASS: deepcopy" || echo "FAIL"

# Storage version marker on v1beta1
grep 'storageversion' /tmp/redis-operator-test/api/v1beta1/rediscluster_types.go && echo "PASS: storageversion on v1beta1" || echo "FAIL"

# Storage version marker NOT on v1alpha1
! grep 'storageversion' /tmp/redis-operator-test/api/v1alpha1/rediscluster_types.go && echo "PASS: no storageversion on v1alpha1" || echo "FAIL"

# New field exists in v1beta1
grep 'MaxMemory' /tmp/redis-operator-test/api/v1beta1/rediscluster_types.go && echo "PASS: maxMemory field" || echo "FAIL"

# Both versions compile
cd /tmp/redis-operator-test && go build ./api/... && echo "PASS: compiles" || echo "FAIL"
```

### Acceptance Criteria

- [ ] `api/v1beta1/` directory with groupversion_info.go, types.go, zz_generated.deepcopy.go
- [ ] `+kubebuilder:storageversion` on v1beta1 root type
- [ ] No `+kubebuilder:storageversion` on v1alpha1 root type
- [ ] v1beta1 has maxMemory field that v1alpha1 doesn't have
- [ ] GroupVersion in v1beta1 uses `Version: "v1beta1"` (not v1alpha1)
- [ ] Both API versions compile

---

## Cleanup

```bash
rm -rf /tmp/redis-operator-test /tmp/mq-operator-test /tmp/types-sdk-test /tmp/webhook-test
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| validate-api-types.py fails on json tags | Field missing `json:"fieldName"` | Add json tag to every exported field |
| `go build ./api/...` fails: undefined Condition | Missing metav1 import | Add `metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"` |
| `go build ./api/...` fails: undefined corev1 | Missing core import | Add `corev1 "k8s.io/api/core/v1"` |
| DeepCopy errors after adding nested types | New types with slices need DeepCopy | Regenerate zz_generated.deepcopy.go or run `make generate` |
| Pattern validation not working | Missing backticks around regex | Use `` `^regex$` `` not `"^regex$"` |
| Webhook compile error: missing admission import | Import not added | Add `"sigs.k8s.io/controller-runtime/pkg/webhook/admission"` |

| Issue | Cause | Fix |
|-------|-------|-----|
| validate-api-types.py fails on json tags | Field missing `json:"fieldName"` | Add json tag to every exported field |
| `go build ./api/...` fails: undefined Condition | Missing metav1 import | Add `metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"` |
| `go build ./api/...` fails: undefined corev1 | Missing core import | Add `corev1 "k8s.io/api/core/v1"` |
| DeepCopy errors after adding nested types | New types with slices need DeepCopy | Regenerate zz_generated.deepcopy.go or run `make generate` |
| Pattern validation not working | Missing backticks around regex | Use `` `^regex$` `` not `"^regex$"` |
