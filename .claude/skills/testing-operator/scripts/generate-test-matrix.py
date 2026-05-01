#!/usr/bin/env python3
"""Generate a test coverage matrix: which reconciler methods have tests."""

import re
import sys
import os

def main():
    if len(sys.argv) < 2:
        print("Usage: generate-test-matrix.py <controller-dir>")
        print("  Scans for reconcileX methods and checks if tests exist.")
        sys.exit(1)

    ctrl_dir = sys.argv[1]
    if not os.path.isdir(ctrl_dir):
        print(f"FAIL: Directory not found: {ctrl_dir}")
        sys.exit(1)

    print(f"=== Test Coverage Matrix: {ctrl_dir} ===\n")

    # Find all reconcileX methods in non-test Go files
    methods = set()
    for f in os.listdir(ctrl_dir):
        if f.endswith('.go') and not f.endswith('_test.go'):
            filepath = os.path.join(ctrl_dir, f)
            with open(filepath) as fh:
                for match in re.finditer(r'func \(r \*\w+\) (reconcile\w+)\(', fh.read()):
                    methods.add(match.group(1))

    if not methods:
        print("No reconcileX methods found.")
        sys.exit(0)

    print(f"Reconciler methods found: {len(methods)}")
    for m in sorted(methods):
        print(f"  - {m}")

    # Find all test files and check for method references
    test_content = ""
    test_files = []
    for f in os.listdir(ctrl_dir):
        if f.endswith('_test.go'):
            test_files.append(f)
            with open(os.path.join(ctrl_dir, f)) as fh:
                test_content += fh.read()

    print(f"\nTest files found: {len(test_files)}")
    for f in sorted(test_files):
        print(f"  - {f}")

    # Check coverage
    print(f"\n{'Method':<30} {'Tested?':<10} {'Create':<8} {'Idempotent':<12}")
    print("-" * 60)

    # Map method names to resource types for indirect testing detection
    method_resources = {}
    for m in methods:
        resource = m.replace('reconcile', '')
        method_resources[m] = resource

    covered = 0
    for method in sorted(methods):
        resource = method_resources[method]
        # Direct: test calls reconcileSecret() / or indirect: test verifies Secret was created
        has_test = method in test_content or resource in test_content
        has_create = ("should create" in test_content or "Created" in test_content) and (method in test_content or resource in test_content)
        has_idempotent = ("idempotent" in test_content.lower() or "not recreate" in test_content.lower() or "repeated reconciliation" in test_content.lower()) and (method in test_content or resource in test_content)

        status = "\033[0;32mYES\033[0m" if has_test else "\033[0;31mNO\033[0m"
        create = "\033[0;32mY\033[0m" if has_create else "\033[0;31mN\033[0m"
        idemp = "\033[0;32mY\033[0m" if has_idempotent else "\033[0;31mN\033[0m"

        print(f"  {method:<28} {status:<18} {create:<16} {idemp}")
        if has_test:
            covered += 1

    print(f"\n=== Results ===")
    pct = (covered / len(methods) * 100) if methods else 0
    if covered == len(methods):
        print(f"\033[0;32mPASSED: {covered}/{len(methods)} methods tested ({pct:.0f}%)\033[0m")
    else:
        print(f"\033[0;31mFAILED: {covered}/{len(methods)} methods tested ({pct:.0f}%)\033[0m")
        sys.exit(1)

if __name__ == "__main__":
    main()
