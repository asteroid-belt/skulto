#!/bin/bash
#
# validate-phase.sh - Validate Definition of Done for a phase
#
# Usage: ./validate-phase.sh [--stack <js|py|go|rust>] [--skip-tests]
#
# Runs all quality checks and reports pass/fail status.
# Exit code 0 = all checks passed, non-zero = failures detected.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
STACK=""
SKIP_TESTS=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --stack)
            STACK="$2"
            shift 2
            ;;
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        -h|--help)
            echo "Usage: ./validate-phase.sh [--stack <js|py|go|rust>] [--skip-tests]"
            echo ""
            echo "Options:"
            echo "  --stack <type>  Specify stack (js, py, go, rust). Auto-detected if not provided."
            echo "  --skip-tests    Skip test execution (use for phases without code changes)"
            echo "  -h, --help      Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Auto-detect stack if not specified
detect_stack() {
    if [[ -f "package.json" ]]; then
        echo "js"
    elif [[ -f "pyproject.toml" ]] || [[ -f "requirements.txt" ]]; then
        echo "py"
    elif [[ -f "go.mod" ]]; then
        echo "go"
    elif [[ -f "Cargo.toml" ]]; then
        echo "rust"
    else
        echo "unknown"
    fi
}

if [[ -z "$STACK" ]]; then
    STACK=$(detect_stack)
    echo -e "${YELLOW}Auto-detected stack: $STACK${NC}"
fi

# Track overall status
OVERALL_STATUS=0
declare -A RESULTS

# Helper function to run a check
run_check() {
    local name="$1"
    local cmd="$2"

    echo -e "\n${YELLOW}Running: $name${NC}"
    echo "Command: $cmd"
    echo "---"

    if eval "$cmd"; then
        RESULTS["$name"]="PASS"
        echo -e "${GREEN}✓ $name: PASSED${NC}"
    else
        RESULTS["$name"]="FAIL"
        OVERALL_STATUS=1
        echo -e "${RED}✗ $name: FAILED${NC}"
    fi
}

# Stack-specific commands
case $STACK in
    js)
        LINT_CMD="npm run lint 2>&1 || npx eslint . --ext .ts,.tsx,.js,.jsx 2>&1"
        FORMAT_CMD="npm run format:check 2>&1 || npx prettier --check . 2>&1"
        TYPECHECK_CMD="npm run typecheck 2>&1 || npx tsc --noEmit 2>&1"
        TEST_CMD="npm test 2>&1"
        ;;
    py)
        LINT_CMD="ruff check . 2>&1 || pylint src 2>&1"
        FORMAT_CMD="black --check . 2>&1"
        TYPECHECK_CMD="mypy . 2>&1 || pyright 2>&1"
        TEST_CMD="pytest 2>&1"
        ;;
    go)
        LINT_CMD="golangci-lint run 2>&1 || go vet ./... 2>&1"
        FORMAT_CMD="test -z \"\$(gofmt -l .)\" 2>&1"
        TYPECHECK_CMD="go build ./... 2>&1"
        TEST_CMD="go test ./... 2>&1"
        ;;
    rust)
        LINT_CMD="cargo clippy -- -D warnings 2>&1"
        FORMAT_CMD="cargo fmt --check 2>&1"
        TYPECHECK_CMD="cargo check 2>&1"
        TEST_CMD="cargo test 2>&1"
        ;;
    *)
        echo -e "${RED}Unknown or unsupported stack: $STACK${NC}"
        echo "Please specify stack with --stack <js|py|go|rust>"
        exit 1
        ;;
esac

echo "=============================================="
echo "DEFINITION OF DONE VALIDATION"
echo "Stack: $STACK"
echo "=============================================="

# Run checks
run_check "Linter" "$LINT_CMD"
run_check "Formatter" "$FORMAT_CMD"
run_check "Type Checker" "$TYPECHECK_CMD"

if [[ "$SKIP_TESTS" != true ]]; then
    run_check "Tests" "$TEST_CMD"
else
    echo -e "\n${YELLOW}Skipping tests (--skip-tests flag)${NC}"
    RESULTS["Tests"]="SKIPPED"
fi

# Summary
echo ""
echo "=============================================="
echo "VALIDATION SUMMARY"
echo "=============================================="

for check in "Linter" "Formatter" "Type Checker" "Tests"; do
    status="${RESULTS[$check]:-NOT RUN}"
    case $status in
        PASS)
            echo -e "  ${GREEN}✓${NC} $check: $status"
            ;;
        FAIL)
            echo -e "  ${RED}✗${NC} $check: $status"
            ;;
        SKIPPED)
            echo -e "  ${YELLOW}○${NC} $check: $status"
            ;;
        *)
            echo -e "  ${YELLOW}?${NC} $check: $status"
            ;;
    esac
done

echo ""
if [[ $OVERALL_STATUS -eq 0 ]]; then
    echo -e "${GREEN}══════════════════════════════════════════════${NC}"
    echo -e "${GREEN}DEFINITION OF DONE: PASSED${NC}"
    echo -e "${GREEN}══════════════════════════════════════════════${NC}"
else
    echo -e "${RED}══════════════════════════════════════════════${NC}"
    echo -e "${RED}DEFINITION OF DONE: FAILED${NC}"
    echo -e "${RED}══════════════════════════════════════════════${NC}"
    echo ""
    echo "Fix the failing checks before proceeding."
fi

exit $OVERALL_STATUS
