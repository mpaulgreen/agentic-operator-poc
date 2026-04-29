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

    # Check each method has Get() before Create()
    create_count = len(re.findall(r'r\.Create\(', content))
    get_count = len(re.findall(r'r\.Get\(', content))

    if create_count > 0 and get_count >= create_count:
        ok(f"Get() calls ({get_count}) >= Create() calls ({create_count}) — idempotency guard present")
    elif create_count > 0 and get_count < create_count:
        fail(f"Create() calls ({create_count}) > Get() calls ({get_count}) — possible non-idempotent creates")
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

    # Check for status updates
    status_updates = len(re.findall(r'r\.Status\(\)\.Update\(', content))
    if status_updates > 0:
        ok(f"Status updates: {status_updates}")

    print(f"\n=== Results ===")
    if errors == 0:
        print(f"\033[0;32mPASSED: {errors} errors, {warnings} warnings\033[0m")
    else:
        print(f"\033[0;31mFAILED: {errors} errors, {warnings} warnings\033[0m")
    sys.exit(1 if errors > 0 else 0)

if __name__ == "__main__":
    main()
