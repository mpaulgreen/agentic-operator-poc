#!/usr/bin/env python3
"""Validate a ClusterServiceVersion YAML for required sections and consistency."""

import sys
import os
import json
import re

try:
    import yaml
except ImportError:
    print("FAIL: PyYAML not installed. Run: pip install pyyaml")
    sys.exit(1)

RED = "\033[0;31m"
GREEN = "\033[0;32m"
RESET = "\033[0m"

def main():
    if len(sys.argv) < 2:
        print("Usage: validate-csv.py <csv-yaml-path>")
        sys.exit(1)

    csv_path = sys.argv[1]
    if not os.path.isfile(csv_path):
        print(f"FAIL: File not found: {csv_path}")
        sys.exit(1)

    print(f"=== Validating CSV: {csv_path} ===\n")

    with open(csv_path) as f:
        csv = yaml.safe_load(f)

    errors = 0
    warnings = 0

    def check(condition, msg, is_warning=False):
        nonlocal errors, warnings
        if condition:
            print(f"{GREEN}PASS: {msg}{RESET}")
        elif is_warning:
            print(f"\033[0;33mWARN: {msg}\033[0m")
            warnings += 1
        else:
            print(f"{RED}FAIL: {msg}{RESET}")
            errors += 1

    # 1. Top-level fields
    check(csv.get("apiVersion") == "operators.coreos.com/v1alpha1",
          "apiVersion is operators.coreos.com/v1alpha1")
    check(csv.get("kind") == "ClusterServiceVersion",
          "kind is ClusterServiceVersion")

    # 2. metadata.name matches <package>.v<version>
    name = csv.get("metadata", {}).get("name", "")
    check(re.match(r'^[\w-]+\.v\d+\.\d+\.\d+$', name),
          f"metadata.name matches <package>.v<version> pattern: {name}")

    # 3. spec.version
    spec = csv.get("spec", {})
    version = spec.get("version", "")
    check(version, f"spec.version present: {version}")

    # 4. Version consistency
    if name and version:
        expected_suffix = f"v{version}"
        check(name.endswith(expected_suffix),
              f"metadata.name version matches spec.version ({expected_suffix})")

    # 5. alm-examples
    annotations = csv.get("metadata", {}).get("annotations", {})
    alm = annotations.get("alm-examples", "")
    if alm:
        try:
            examples = json.loads(alm)
            check(isinstance(examples, list) and len(examples) > 0,
                  f"alm-examples is valid JSON array with {len(examples)} entry(ies)")
        except json.JSONDecodeError:
            check(False, "alm-examples is valid JSON")
    else:
        check(False, "alm-examples annotation present")

    # 6. customresourcedefinitions.owned
    crds = spec.get("customresourcedefinitions", {}).get("owned", [])
    check(len(crds) > 0, f"At least one owned CRD: {len(crds)} found")

    for crd in crds:
        check(crd.get("kind"), f"  Owned CRD has kind: {crd.get('kind', 'MISSING')}")
        check(crd.get("name"), f"  Owned CRD has name: {crd.get('name', 'MISSING')}")
        check(crd.get("version"), f"  Owned CRD has version: {crd.get('version', 'MISSING')}")

    # 7. install.spec permissions
    install = spec.get("install", {}).get("spec", {})
    cluster_perms = install.get("clusterPermissions", [])
    ns_perms = install.get("permissions", [])
    check(len(cluster_perms) > 0 or len(ns_perms) > 0,
          f"Permissions present (cluster: {len(cluster_perms)}, namespace: {len(ns_perms)})")

    if cluster_perms:
        rules = cluster_perms[0].get("rules", [])
        check(len(rules) > 0, f"  clusterPermissions has {len(rules)} rules")
        sa = cluster_perms[0].get("serviceAccountName", "")
        check(sa, f"  clusterPermissions serviceAccountName: {sa}")

    # 8. deployments
    deployments = install.get("deployments", [])
    check(len(deployments) > 0, f"At least one deployment: {len(deployments)} found")

    # 9. installModes
    modes = spec.get("installModes", [])
    supported = [m for m in modes if m.get("supported")]
    check(len(modes) > 0, f"installModes present: {len(modes)} modes")
    check(len(supported) > 0, f"At least one installMode supported: {len(supported)}")

    # 10. displayName and description
    check(spec.get("displayName"), f"displayName present: {spec.get('displayName', 'MISSING')}")
    check(spec.get("description"), "description present")

    # 11. icon
    icons = spec.get("icon", [])
    check(len(icons) > 0, "icon field exists", is_warning=True)

    # 12. maturity
    check(spec.get("maturity"), f"maturity present: {spec.get('maturity', 'MISSING')}")

    # 13. install strategy
    strategy = spec.get("install", {}).get("strategy")
    check(strategy == "deployment", f"install strategy is 'deployment': {strategy}")

    # 14. specDescriptors (warning only)
    has_spec_desc = any(crd.get("specDescriptors") for crd in crds)
    check(has_spec_desc, "specDescriptors present on owned CRDs", is_warning=True)

    # 15. statusDescriptors (warning only)
    has_status_desc = any(crd.get("statusDescriptors") for crd in crds)
    check(has_status_desc, "statusDescriptors present on owned CRDs", is_warning=True)

    print(f"\n=== Results ===")
    if errors == 0:
        print(f"{GREEN}PASSED: {errors} errors, {warnings} warnings{RESET}")
    else:
        print(f"{RED}FAILED: {errors} errors, {warnings} warnings{RESET}")
        sys.exit(1)

if __name__ == "__main__":
    main()
