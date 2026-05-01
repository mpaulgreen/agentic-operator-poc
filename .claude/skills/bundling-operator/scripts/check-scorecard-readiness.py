#!/usr/bin/env python3
"""Check scorecard readiness: verify config, alm-examples coverage, and descriptor paths."""

import sys
import os
import json
import glob

try:
    import yaml
except ImportError:
    print("FAIL: PyYAML not installed. Run: pip install pyyaml")
    sys.exit(1)

RED = "\033[0;31m"
GREEN = "\033[0;32m"
YELLOW = "\033[0;33m"
RESET = "\033[0m"

def main():
    if len(sys.argv) < 2:
        print("Usage: check-scorecard-readiness.py <bundle-dir>")
        print("  Checks scorecard config, alm-examples, and descriptors")
        sys.exit(1)

    bundle_dir = sys.argv[1].rstrip("/")
    if not os.path.isdir(bundle_dir):
        print(f"FAIL: Directory not found: {bundle_dir}")
        sys.exit(1)

    print(f"=== Scorecard Readiness Check: {bundle_dir} ===\n")

    errors = 0
    warnings = 0

    def check(condition, msg, is_warning=False):
        nonlocal errors, warnings
        if condition:
            print(f"{GREEN}PASS: {msg}{RESET}")
        elif is_warning:
            print(f"{YELLOW}WARN: {msg}{RESET}")
            warnings += 1
        else:
            print(f"{RED}FAIL: {msg}{RESET}")
            errors += 1

    # 1. Scorecard config exists and has required tests
    sc_path = os.path.join(bundle_dir, "tests", "scorecard", "config.yaml")
    check(os.path.isfile(sc_path), "scorecard config.yaml exists")

    required_tests = ["basic-check-spec", "olm-bundle-validation"]
    if os.path.isfile(sc_path):
        with open(sc_path) as f:
            sc = yaml.safe_load(f)

        all_tests = []
        for stage in sc.get("stages", []):
            for test in stage.get("tests", []):
                ep = test.get("entrypoint", [])
                if len(ep) >= 2:
                    all_tests.append(ep[1])

        for rt in required_tests:
            check(rt in all_tests, f"scorecard has {rt} test")

        total = len(all_tests)
        check(total >= 2, f"scorecard has {total} tests total")

    # 2. Load CSV
    csv_files = glob.glob(os.path.join(bundle_dir, "manifests", "*.clusterserviceversion.yaml"))
    if not csv_files:
        check(False, "CSV file found in manifests/")
        print(f"\n=== Results ===\n{RED}FAILED: {errors} errors, {warnings} warnings{RESET}")
        sys.exit(1)

    with open(csv_files[0]) as f:
        csv = yaml.safe_load(f)

    # 3. alm-examples covers all owned CRDs
    owned_crds = csv.get("spec", {}).get("customresourcedefinitions", {}).get("owned", [])
    alm_str = csv.get("metadata", {}).get("annotations", {}).get("alm-examples", "[]")
    try:
        alm_examples = json.loads(alm_str)
    except json.JSONDecodeError:
        alm_examples = []

    alm_kinds = {ex.get("kind", "") for ex in alm_examples}
    for crd in owned_crds:
        kind = crd.get("kind", "")
        check(kind in alm_kinds,
              f"alm-examples has entry for {kind}")

    # 4. specDescriptors present
    for crd in owned_crds:
        kind = crd.get("kind", "")
        spec_desc = crd.get("specDescriptors", [])
        check(len(spec_desc) > 0,
              f"specDescriptors present for {kind} ({len(spec_desc)} descriptors)",
              is_warning=True)

    # 5. statusDescriptors present
    for crd in owned_crds:
        kind = crd.get("kind", "")
        status_desc = crd.get("statusDescriptors", [])
        check(len(status_desc) > 0,
              f"statusDescriptors present for {kind} ({len(status_desc)} descriptors)",
              is_warning=True)

    # 6. Check descriptor paths against CRD schema if available
    crd_files = [f for f in glob.glob(os.path.join(bundle_dir, "manifests", "*.yaml"))
                 if "clusterserviceversion" not in f]

    for crd_file in crd_files:
        with open(crd_file) as f:
            crd_yaml = yaml.safe_load(f)

        if not crd_yaml or crd_yaml.get("kind") != "CustomResourceDefinition":
            continue

        crd_kind = crd_yaml.get("spec", {}).get("names", {}).get("kind", "")
        versions = crd_yaml.get("spec", {}).get("versions", [])
        if not versions:
            continue

        schema = versions[0].get("schema", {}).get("openAPIV3Schema", {})
        spec_props = schema.get("properties", {}).get("spec", {}).get("properties", {})
        status_props = schema.get("properties", {}).get("status", {}).get("properties", {})

        for crd in owned_crds:
            if crd.get("kind") != crd_kind:
                continue

            for desc in crd.get("specDescriptors", []):
                path = desc.get("path", "")
                top_field = path.split(".")[0] if path else ""
                if top_field and spec_props:
                    check(top_field in spec_props,
                          f"  specDescriptor path '{path}' exists in CRD schema",
                          is_warning=True)

            for desc in crd.get("statusDescriptors", []):
                path = desc.get("path", "")
                top_field = path.split(".")[0] if path else ""
                if top_field and status_props:
                    check(top_field in status_props,
                          f"  statusDescriptor path '{path}' exists in CRD schema",
                          is_warning=True)

    print(f"\n=== Results ===")
    if errors == 0:
        print(f"{GREEN}PASSED: {errors} errors, {warnings} warnings{RESET}")
    else:
        print(f"{RED}FAILED: {errors} errors, {warnings} warnings{RESET}")
        sys.exit(1)

if __name__ == "__main__":
    main()
