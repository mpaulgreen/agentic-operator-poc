#!/usr/bin/env python3
"""Validate operator API types file for common issues."""

import re
import sys

def main():
    if len(sys.argv) < 2:
        print("Usage: validate-api-types.py <types-file>")
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

    print(f"=== Validating API Types: {filepath} ===\n")

    # 1. Package declaration
    if re.search(r'^package \w+', content, re.MULTILINE):
        ok("Package declaration found")
    else:
        fail("No package declaration")

    # 2. License header
    if "Licensed under the Apache License" in content or "Copyright" in content:
        ok("License header present")
    else:
        warn("No license header")

    # 3. Imports
    if 'metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"' in content:
        ok("metav1 import present")
    else:
        fail("Missing metav1 import")

    # 4. Spec struct
    spec_match = re.search(r'type \w+Spec struct', content)
    if spec_match:
        ok(f"Spec struct found: {spec_match.group()}")
    else:
        fail("No Spec struct found (expected <Kind>Spec)")

    # 5. Status struct
    status_match = re.search(r'type \w+Status struct', content)
    if status_match:
        ok(f"Status struct found: {status_match.group()}")
    else:
        fail("No Status struct found (expected <Kind>Status)")

    # 6. Root type markers
    if "+kubebuilder:object:root=true" in content:
        ok("Root type marker (+kubebuilder:object:root=true)")
    else:
        fail("Missing +kubebuilder:object:root=true on root type")

    if "+kubebuilder:subresource:status" in content:
        ok("Status subresource marker")
    else:
        fail("Missing +kubebuilder:subresource:status")

    # 7. List type
    if re.search(r'type \w+List struct', content):
        ok("List type found")
    else:
        fail("No List type (expected <Kind>List)")

    # 8. SchemeBuilder.Register
    if "SchemeBuilder.Register" in content:
        ok("SchemeBuilder.Register in init()")
    else:
        fail("Missing SchemeBuilder.Register in init()")

    # 9. JSON tags on fields
    # Only check lines inside struct bodies, not in comments or license headers
    in_struct = False
    brace_depth = 0
    fields_without_json = []
    for line in content.split('\n'):
        stripped = line.strip()
        if re.match(r'^type \w+ struct', stripped):
            in_struct = True
            brace_depth = 0
        if in_struct:
            brace_depth += stripped.count('{') - stripped.count('}')
            if brace_depth <= 0 and '{' not in stripped:
                in_struct = False
                continue
        if not in_struct:
            continue
        # Skip non-field lines
        if stripped.startswith("//") or stripped.startswith("/*") or stripped.startswith("*") or stripped == '{' or stripped == '}':
            continue
        if "TypeMeta" in stripped or "ObjectMeta" in stripped or "ListMeta" in stripped:
            continue
        if "Items" in stripped and "[]" in stripped:
            continue
        if "TODO" in stripped or stripped == "":
            continue
        # Check exported fields (start with uppercase after optional whitespace)
        if re.match(r'^[A-Z]\w+\s+\S+', stripped):
            if '`json:"' not in stripped:
                fields_without_json.append(stripped)

    if fields_without_json:
        for f in fields_without_json[:5]:
            fail(f"Field missing json tag: {f.strip()}")
    else:
        ok("All fields have json tags")

    # 10. Conditions in Status
    if "Conditions" in content and "metav1.Condition" in content:
        ok("Status has Conditions []metav1.Condition")
    elif "Conditions" in content:
        warn("Conditions field exists but may not use metav1.Condition")
    else:
        warn("No Conditions field in Status (recommended for production)")

    # 11. Validation markers
    marker_count = len(re.findall(r'\+kubebuilder:validation:', content))
    if marker_count > 0:
        ok(f"Validation markers found: {marker_count}")
    else:
        warn("No validation markers found")

    # 12. Print columns
    printcol_count = len(re.findall(r'\+kubebuilder:printcolumn:', content))
    if printcol_count >= 2:
        ok(f"Print columns found: {printcol_count}")
    elif printcol_count == 1:
        warn("Only 1 print column (recommend at least 2: a status field + Age)")
    else:
        warn("No print columns (recommend adding for kubectl get output)")

    # 13. Default values
    default_count = len(re.findall(r'\+kubebuilder:default=', content))
    if default_count > 0:
        ok(f"Default values found: {default_count}")

    # Summary
    print(f"\n=== Results ===")
    if errors == 0:
        print(f"\033[0;32mPASSED: {errors} errors, {warnings} warnings\033[0m")
    else:
        print(f"\033[0;31mFAILED: {errors} errors, {warnings} warnings\033[0m")
    sys.exit(1 if errors > 0 else 0)

if __name__ == "__main__":
    main()
