#!/usr/bin/env bash
# Check test coverage for an operator controller package.
# Usage: check-test-coverage.sh <project-dir>

set -euo pipefail

PROJECT_DIR="${1:?Usage: check-test-coverage.sh <project-directory>}"

echo "=== Test Coverage Check ==="
echo "Project: ${PROJECT_DIR}"
echo ""

cd "$PROJECT_DIR"

# Check if test files exist
TEST_FILES=$(find internal/controller -name '*_test.go' 2>/dev/null | wc -l | tr -d ' ')
if [ "$TEST_FILES" -eq 0 ]; then
    echo -e "\033[0;31mFAIL: No test files found in internal/controller/\033[0m"
    exit 1
fi
echo -e "\033[0;32mPASS: Found $TEST_FILES test file(s)\033[0m"

# Check for suite_test.go
if [ -f internal/controller/suite_test.go ]; then
    echo -e "\033[0;32mPASS: suite_test.go exists\033[0m"
else
    echo -e "\033[0;31mFAIL: suite_test.go missing (envtest setup required)\033[0m"
    exit 1
fi

# Check for controller test
CTRL_TESTS=$(find internal/controller -name '*_controller_test.go' | wc -l | tr -d ' ')
if [ "$CTRL_TESTS" -gt 0 ]; then
    echo -e "\033[0;32mPASS: Controller test file(s) found: $CTRL_TESTS\033[0m"
else
    echo -e "\033[0;33mWARN: No controller_test.go file found\033[0m"
fi

# Count test functions
IT_COUNT=$(grep -rch 'It(' internal/controller/*_test.go 2>/dev/null | awk '{s+=$1} END {print s+0}')
echo "Test cases (It blocks): $IT_COUNT"

# Count tested methods
METHODS=$(grep -rch 'func.*reconcile\|func.*Reconcile\|func.*update\|func.*handle\|func.*labels\|func.*generate' internal/controller/*_test.go 2>/dev/null | awk '{s+=$1} END {print s+0}')
echo "Method references in tests: $METHODS"

echo ""
echo "=== Results ==="
if [ "$IT_COUNT" -ge 3 ]; then
    echo -e "\033[0;32mPASSED: $IT_COUNT test cases across $TEST_FILES file(s)\033[0m"
else
    echo -e "\033[0;33mWARN: Only $IT_COUNT test cases — consider adding more\033[0m"
fi
