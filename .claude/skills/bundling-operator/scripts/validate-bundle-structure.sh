#!/bin/bash
# Validate OLM bundle directory structure and required files.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
RESET='\033[0m'

if [ $# -lt 1 ]; then
    echo "Usage: validate-bundle-structure.sh <project-dir>"
    echo "  Validates bundle/ directory structure and bundle.Dockerfile"
    exit 1
fi

PROJECT_DIR="${1%/}"
BUNDLE_DIR="$PROJECT_DIR/bundle"
ERRORS=0
WARNINGS=0

echo "=== Validating Bundle Structure: $PROJECT_DIR ==="
echo ""

check() {
    if [ "$1" = "true" ]; then
        echo -e "${GREEN}PASS: $2${RESET}"
    else
        echo -e "${RED}FAIL: $2${RESET}"
        ERRORS=$((ERRORS + 1))
    fi
}

warn() {
    if [ "$1" = "true" ]; then
        echo -e "${GREEN}PASS: $2${RESET}"
    else
        echo -e "\033[0;33mWARN: $2\033[0m"
        WARNINGS=$((WARNINGS + 1))
    fi
}

# 1. bundle/manifests/ directory
check "$([ -d "$BUNDLE_DIR/manifests" ] && echo true || echo false)" \
    "bundle/manifests/ directory exists"

# 2. CSV file exists
CSV_COUNT=$(find "$BUNDLE_DIR/manifests" -name '*.clusterserviceversion.yaml' 2>/dev/null | wc -l | tr -d ' ')
check "$([ "$CSV_COUNT" -gt 0 ] && echo true || echo false)" \
    "CSV file exists in manifests/ ($CSV_COUNT found)"

# 3. CRD file(s) exist
CRD_COUNT=$(find "$BUNDLE_DIR/manifests" -name '*.yaml' ! -name '*.clusterserviceversion.yaml' 2>/dev/null | wc -l | tr -d ' ')
check "$([ "$CRD_COUNT" -gt 0 ] && echo true || echo false)" \
    "At least one non-CSV YAML in manifests/ ($CRD_COUNT found)"

# 4. metadata/annotations.yaml
check "$([ -f "$BUNDLE_DIR/metadata/annotations.yaml" ] && echo true || echo false)" \
    "bundle/metadata/annotations.yaml exists"

# 5. annotations.yaml has required keys
if [ -f "$BUNDLE_DIR/metadata/annotations.yaml" ]; then
    for key in "bundle.mediatype" "bundle.manifests" "bundle.metadata" "bundle.package" "bundle.channels"; do
        found=$(grep -c "$key" "$BUNDLE_DIR/metadata/annotations.yaml" 2>/dev/null || echo 0)
        check "$([ "$found" -gt 0 ] && echo true || echo false)" \
            "  annotations.yaml has $key"
    done
fi

# 6. scorecard config
check "$([ -f "$BUNDLE_DIR/tests/scorecard/config.yaml" ] && echo true || echo false)" \
    "bundle/tests/scorecard/config.yaml exists"

# 7. bundle.Dockerfile
check "$([ -f "$PROJECT_DIR/bundle.Dockerfile" ] && echo true || echo false)" \
    "bundle.Dockerfile exists at project root"

# 8. Dockerfile LABELs match annotations
if [ -f "$PROJECT_DIR/bundle.Dockerfile" ] && [ -f "$BUNDLE_DIR/metadata/annotations.yaml" ]; then
    LABEL_COUNT=$(grep -c '^LABEL ' "$PROJECT_DIR/bundle.Dockerfile" 2>/dev/null || echo 0)
    warn "$([ "$LABEL_COUNT" -ge 5 ] && echo true || echo false)" \
        "bundle.Dockerfile has $LABEL_COUNT LABEL lines (expected >= 5)"
fi

# 9. FROM scratch
if [ -f "$PROJECT_DIR/bundle.Dockerfile" ]; then
    check "$(grep -q '^FROM scratch' "$PROJECT_DIR/bundle.Dockerfile" && echo true || echo false)" \
        "bundle.Dockerfile uses FROM scratch"
fi

echo ""
echo "=== Results ==="
if [ "$ERRORS" -eq 0 ]; then
    echo -e "${GREEN}PASSED: $ERRORS errors, $WARNINGS warnings${RESET}"
else
    echo -e "${RED}FAILED: $ERRORS errors, $WARNINGS warnings${RESET}"
    exit 1
fi
