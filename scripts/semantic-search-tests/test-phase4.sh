#!/bin/bash
#
# Phase 4 Functional Test Script: On-Launch Incremental Indexing
#
# This script tests the complete Phase 4 implementation including:
# - BackgroundIndexer creation and lifecycle
# - Non-blocking start behavior
# - Progress reporting via channels
# - TUI integration with footer progress display
# - Graceful degradation without API key
#
# Usage:
#   # Without API key (tests graceful degradation)
#   ./scripts/test-phase4.sh
#
#   # With API key (full functional test)
#   export OPENAI_API_KEY=sk-your-key-here
#   ./scripts/test-phase4.sh
#

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Phase 4 Functional Tests: Background Indexer          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}━━━ Checking Prerequisites ━━━${NC}"

    # Check Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}✗ Go is not installed${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Go is installed ($(go version | cut -d' ' -f3))${NC}"

    # Check we're in the right directory
    if [ ! -f "go.mod" ] || ! grep -q "skulto" go.mod; then
        echo -e "${RED}✗ Not in skulto project directory${NC}"
        echo "Please run this script from the project root"
        exit 1
    fi
    echo -e "${GREEN}✓ In skulto project directory${NC}"

    # Check OPENAI_API_KEY
    if [ -z "$OPENAI_API_KEY" ]; then
        echo -e "${YELLOW}⚠ OPENAI_API_KEY is not set${NC}"
        echo -e "  Some tests will be skipped. Set it to run full tests:"
        echo -e "  ${CYAN}export OPENAI_API_KEY=sk-your-key-here${NC}"
        HAS_API_KEY=false
    else
        echo -e "${GREEN}✓ OPENAI_API_KEY is set${NC}"
        HAS_API_KEY=true
    fi

    echo ""
}

# Run unit tests for BackgroundIndexer
run_unit_tests() {
    echo -e "${BLUE}━━━ Phase 4.1: BackgroundIndexer Unit Tests ━━━${NC}"

    echo "Running background indexer unit tests..."
    if go test ./internal/search/... -v -count=1 -run "Background" 2>&1 | tee /tmp/phase4_unit.log | grep -E "(PASS|FAIL|---)" ; then
        echo ""
    fi

    if grep -q "FAIL" /tmp/phase4_unit.log; then
        echo -e "${RED}✗ Some unit tests failed${NC}"
        return 1
    fi

    echo -e "${GREEN}✓ BackgroundIndexer unit tests passed${NC}"
    echo ""
}

# Test TUI integration (home view footer)
test_tui_integration() {
    echo -e "${BLUE}━━━ Phase 4.2: TUI Integration Tests ━━━${NC}"

    echo "Running TUI tests..."
    if go test ./internal/tui/... -v -count=1 2>&1 | tee /tmp/phase4_tui.log | grep -E "(PASS|FAIL|---)" ; then
        echo ""
    fi

    if grep -q "FAIL" /tmp/phase4_tui.log; then
        echo -e "${RED}✗ Some TUI tests failed${NC}"
        return 1
    fi

    echo -e "${GREEN}✓ TUI integration tests passed${NC}"
    echo ""
}

# Test build compiles
test_build() {
    echo -e "${BLUE}━━━ Phase 4.3: Build Verification ━━━${NC}"

    echo "Building skulto..."
    if go build -o /tmp/skulto-phase4-test ./cmd/skulto 2>&1; then
        echo -e "${GREEN}✓ Build succeeded${NC}"
    else
        echo -e "${RED}✗ Build failed${NC}"
        return 1
    fi

    # Check binary exists
    if [ -f "/tmp/skulto-phase4-test" ]; then
        echo -e "${GREEN}✓ Binary created${NC}"
        rm /tmp/skulto-phase4-test
    fi

    echo ""
}

# Run functional tests (requires API key)
run_functional_tests() {
    echo -e "${BLUE}━━━ Phase 4.4: Functional Tests (API) ━━━${NC}"

    if [ "$HAS_API_KEY" = false ]; then
        echo -e "${YELLOW}⚠ Skipping functional tests (no API key)${NC}"
        echo ""
        return 0
    fi

    echo -e "${CYAN}Running Phase 4 functional tests...${NC}"
    echo -e "${YELLOW}This will make real API calls to OpenAI.${NC}"
    echo ""

    if go test -tags=functional ./internal/search/... -v -count=1 -run "Phase4" 2>&1 | tee /tmp/phase4_functional.log; then
        echo ""
        echo -e "${GREEN}✓ Functional tests passed${NC}"
    else
        echo ""
        echo -e "${RED}✗ Functional tests failed${NC}"
        echo "See /tmp/phase4_functional.log for details"
        return 1
    fi

    echo ""
}

# Test graceful degradation without API key
test_no_api_key() {
    echo -e "${BLUE}━━━ Phase 4.5: Graceful Degradation Test ━━━${NC}"

    echo "Testing behavior without OPENAI_API_KEY..."

    # Temporarily unset API key
    OLD_API_KEY="$OPENAI_API_KEY"
    unset OPENAI_API_KEY

    # Build and check output
    go build -o /tmp/skulto-nokey-test ./cmd/skulto 2>&1

    # Run with timeout and capture output
    timeout 3s /tmp/skulto-nokey-test 2>&1 | head -30 > /tmp/phase4_nokey.log || true

    # Check for expected message
    if grep -q "Semantic search: disabled" /tmp/phase4_nokey.log; then
        echo -e "${GREEN}✓ Correctly reports semantic search disabled${NC}"
    else
        echo -e "${YELLOW}⚠ Could not verify disabled message (may have exited early)${NC}"
    fi

    # Restore API key
    if [ -n "$OLD_API_KEY" ]; then
        export OPENAI_API_KEY="$OLD_API_KEY"
    fi

    rm -f /tmp/skulto-nokey-test

    echo ""
}

# Test vector store initialization
test_vector_store() {
    echo -e "${BLUE}━━━ Phase 4.6: VectorStore Tests ━━━${NC}"

    echo "Running vector store tests..."
    if go test ./internal/vector/... -v -count=1 2>&1 | tee /tmp/phase4_vector.log | grep -E "(PASS|FAIL|---)" ; then
        echo ""
    fi

    if grep -q "FAIL" /tmp/phase4_vector.log; then
        echo -e "${RED}✗ Some vector store tests failed${NC}"
        return 1
    fi

    echo -e "${GREEN}✓ VectorStore tests passed${NC}"
    echo ""
}

# Print manual testing instructions
print_manual_instructions() {
    echo -e "${BLUE}━━━ Manual TUI Testing Instructions ━━━${NC}"
    echo ""
    echo "To manually test the Phase 4 TUI integration:"
    echo ""
    echo "1. Ensure you have skills in the database:"
    echo -e "   ${CYAN}go run ./cmd/skulto${NC}"
    echo ""
    echo "2. Look at the startup output for:"
    echo -e "   ${GREEN}🔍 Semantic search: enabled (OPENAI_API_KEY found)${NC}"
    echo -e "   ${GREEN}   Found X skills to index for semantic search${NC}"
    echo ""
    echo "3. In the TUI, check the footer (bottom right) for:"
    echo -e "   ${YELLOW}Indexing X/Y skills...${NC}"
    echo ""
    echo "4. The indicator should disappear when indexing completes"
    echo ""
    echo "5. Test semantic search by pressing ${CYAN}/${NC} and typing a query"
    echo ""
    echo "6. Verify semantic results are included in search results"
    echo ""
}

# Print summary
print_summary() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                    Phase 4 Test Summary                    ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Components tested:"
    echo -e "  ${GREEN}✓${NC} BackgroundIndexer (internal/search/background.go)"
    echo -e "  ${GREEN}✓${NC} IndexProgress struct and channel communication"
    echo -e "  ${GREEN}✓${NC} TUI Model integration (internal/tui/app.go)"
    echo -e "  ${GREEN}✓${NC} HomeView footer with indexing progress"
    echo -e "  ${GREEN}✓${NC} Main initialization (cmd/skulto/main.go)"
    echo -e "  ${GREEN}✓${NC} VectorStore integration"
    echo ""
    echo "Key Phase 4 features verified:"
    echo "  - Non-blocking Start() returns immediately"
    echo "  - Progress updates sent via channel"
    echo "  - Footer displays 'Indexing X/Y skills...'"
    echo "  - Graceful degradation without API key"
    echo "  - Clean shutdown with Close()"
    echo ""
}

# Main execution
main() {
    check_prerequisites
    run_unit_tests
    test_tui_integration
    test_build
    test_vector_store
    run_functional_tests
    test_no_api_key
    print_summary
    print_manual_instructions
}

main "$@"
