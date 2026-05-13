#!/usr/bin/env python3
"""Check reconciler methods for idempotency patterns."""

import re
import sys

def main():
    if len(sys.argv) < 2:
        print("Usage: check-idempotency.py <reconcilers-file>")
        sys.exit(1)

    filepath = sys.argv[1]
    try:
        with open(filepath) as f:
            content = f.read()
    except FileNotFoundError:
        print(f"FAIL: File not found: {filepath}")
        sys.exit(1)

    errors = 0
    warnings = 0

    def fail(msg):
        nonlocal errors
        print(f"\033[0;31mFAIL: {msg}\033[0m")
        errors += 1

    def warn(msg):
        nonlocal warnings
        print(f"\033[0;33mWARN: {msg}\033[0m")
        warnings += 1

    def ok(msg):
        print(f"\033[0;32mPASS: {msg}\033[0m")

    print(f"=== Checking Idempotency: {filepath} ===\n")

    # Find reconciler methods
    methods = re.findall(r'func \(r \*\w+\) (reconcile\w+)\(', content)
    if methods:
        ok(f"Reconciler methods found: {len(methods)} ({', '.join(methods)})")
    else:
        warn("No reconcileX() methods found")

    # Check each method has Get() or List() before Create()
    # List() is equivalent to Get() for resources that use label selectors
    # (e.g., Jobs are immutable and have timestamp-based names, so List() with
    # label selector is the correct idempotency guard)
    create_count = len(re.findall(r'r\.Create\(', content))
    get_count = len(re.findall(r'r\.Get\(', content))
    list_count = len(re.findall(r'r\.List\(', content))
    existence_checks = get_count + list_count

    if create_count > 0 and existence_checks >= create_count:
        ok(f"Get()/List() calls ({existence_checks}) >= Create() calls ({create_count}) — idempotency guard present")
    elif create_count > 0 and existence_checks < create_count:
        fail(f"Create() calls ({create_count}) > Get()/List() calls ({existence_checks}) — possible non-idempotent creates")
    elif create_count == 0:
        warn("No Create() calls found")

    # Check IsNotFound pattern
    is_not_found = len(re.findall(r'errors\.IsNotFound\(', content))
    if is_not_found > 0:
        ok(f"IsNotFound checks: {is_not_found}")
    elif create_count > 0:
        fail("No errors.IsNotFound() checks — creates may not be guarded")

    # Check SetControllerReference usage
    owner_refs = len(re.findall(r'SetControllerReference\(|SetOwnerReference\(', content))
    if owner_refs > 0:
        ok(f"Owner reference calls: {owner_refs}")
    elif create_count > 0:
        fail("No SetControllerReference() calls — created resources won't have owner references")

    # Check event recording
    events = len(re.findall(r'r\.Recorder\.Event\(', content))
    if events > 0:
        ok(f"Event recordings: {events}")
    else:
        warn("No event recordings found")

    # Check for Update() with spec comparison
    updates = len(re.findall(r'r\.Update\(', content))
    if updates > 0:
        ok(f"Update() calls: {updates}")

    # Check that mutable resources have field comparisons before Update()
    # For each reconciler method with r.Update(), check it also has field comparison logic
    for method in methods:
        # Extract method body (rough heuristic: from method name to next "func " or EOF)
        method_pattern = rf'func \(r \*\w+\) {method}\(.*?\n(.*?)(?=\nfunc |\Z)'
        method_match = re.search(method_pattern, content, re.DOTALL)
        if method_match:
            method_body = method_match.group(1)
            has_update = 'r.Update(' in method_body
            has_create = 'r.Create(' in method_body
            if has_update and has_create:
                # Count field comparisons (!=, DeepEqual, Equal patterns)
                comparisons = len(re.findall(r'!=|DeepEqual|Equal\(', method_body))
                # Count field assignments in builder section (after "BUILD" comment or resource construction)
                if comparisons < 1:
                    warn(f"{method}() has Update() but no field comparisons — mutable fields may not be reconciled")

    # Check for status updates
    status_updates = len(re.findall(r'r\.Status\(\)\.Update\(', content))
    if status_updates > 0:
        ok(f"Status updates: {status_updates}")

    # Check for context.TODO() or context.Background() usage in reconciler methods
    ctx_todo = re.findall(r'context\.TODO\(\)', content)
    ctx_bg = re.findall(r'context\.Background\(\)', content)
    bad_ctx = len(ctx_todo) + len(ctx_bg)
    if bad_ctx > 0:
        warn(f"context.TODO()/Background() used {bad_ctx} time(s) — use ctx parameter from Reconcile() instead")
    else:
        ok("No context.TODO()/Background() usage — correctly uses passed ctx")

    # Check for boolean logic errors in error handling
    bad_logic = re.findall(r'!(?:errors\.)?IsNotFound\([^)]+\)\s*\|\|\s*!(?:errors\.)?Is(?:Gone|NotFound)', content)
    bad_logic += re.findall(r'!(?:errors\.)?IsGone\([^)]+\)\s*\|\|\s*!(?:errors\.)?Is(?:NotFound|Gone)', content)
    if bad_logic:
        warn(f"Possible boolean logic error: negated error checks combined with || instead of && ({len(bad_logic)} occurrence(s))")

    print(f"\n=== Results ===")
    if errors == 0:
        print(f"\033[0;32mPASSED: {errors} errors, {warnings} warnings\033[0m")
    else:
        print(f"\033[0;31mFAILED: {errors} errors, {warnings} warnings\033[0m")
    sys.exit(1 if errors > 0 else 0)

if __name__ == "__main__":
    main()
