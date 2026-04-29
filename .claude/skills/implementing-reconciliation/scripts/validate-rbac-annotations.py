#!/usr/bin/env python3
"""Validate RBAC annotations in controller files match managed resources."""

import re
import sys

def main():
    if len(sys.argv) < 2:
        print("Usage: validate-rbac-annotations.py <controller-file>")
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

    print(f"=== Validating RBAC Annotations: {filepath} ===\n")

    # Extract RBAC markers
    rbac_markers = re.findall(r'//\s*\+kubebuilder:rbac:(.+)', content)
    if rbac_markers:
        ok(f"RBAC markers found: {len(rbac_markers)}")
    else:
        fail("No RBAC markers found")
        print(f"\n=== Results ===\n\033[0;31mFAILED: {errors} errors\033[0m")
        sys.exit(1)

    # Parse each marker
    managed_resources = set()
    has_status = False
    has_finalizers = False
    has_events = False
    has_wildcard = False

    for marker in rbac_markers:
        parts = dict(p.split('=', 1) for p in marker.split(',') if '=' in p)
        resources = parts.get('resources', '')
        verbs = parts.get('verbs', '')

        if resources.endswith('/status'):
            has_status = True
        elif resources.endswith('/finalizers'):
            has_finalizers = True
        elif resources == 'events':
            has_events = True
        else:
            managed_resources.add(resources)

        if '*' in verbs:
            has_wildcard = True

    # Check for wildcard verbs
    if has_wildcard:
        fail("Wildcard verbs (*) found — use explicit verbs for least privilege")
    else:
        ok("No wildcard verbs")

    # Check for status subresource
    if has_status:
        ok("Status subresource RBAC present")
    else:
        warn("No /status subresource RBAC — needed if updating status")

    # Check for events permission
    if has_events:
        ok("Events permission present")
    else:
        warn("No events permission — needed for r.Recorder.Event()")

    # Check for finalizer permission
    if 'controllerutil.ContainsFinalizer' in content or 'AddFinalizer' in content:
        if has_finalizers:
            ok("Finalizer RBAC matches finalizer usage")
        else:
            fail("Uses finalizers but missing /finalizers RBAC marker")
    elif has_finalizers:
        warn("Has /finalizers RBAC but no finalizer usage detected")

    # Check Create() calls have matching RBAC
    create_calls = re.findall(r'r\.Create\(ctx,\s*(\w+)', content)
    for var_name in create_calls:
        ok(f"Create() call found for variable: {var_name}")

    # Check Get() before Create() (idempotency)
    reconcile_methods = re.findall(r'func \(r \*\w+\) reconcile\w+', content)
    for method in reconcile_methods:
        ok(f"Reconciler method found: {method}")

    print(f"\n=== Summary ===")
    print(f"RBAC markers: {len(rbac_markers)}")
    print(f"Managed resources: {', '.join(sorted(managed_resources))}")

    print(f"\n=== Results ===")
    if errors == 0:
        print(f"\033[0;32mPASSED: {errors} errors, {warnings} warnings\033[0m")
    else:
        print(f"\033[0;31mFAILED: {errors} errors, {warnings} warnings\033[0m")
    sys.exit(1 if errors > 0 else 0)

if __name__ == "__main__":
    main()
