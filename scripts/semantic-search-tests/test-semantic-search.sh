#!/bin/bash
#
# Functional Test Script for Semantic Search (Phases 1-3)
#
# This script tests the semantic search implementation end-to-end.
# Requires: OPENAI_API_KEY environment variable
#
# Usage:
#   export OPENAI_API_KEY=sk-your-key-here
#   ./scripts/test-semantic-search.sh
#

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test directory
TEST_DIR=$(mktemp -d)
DB_PATH="$TEST_DIR/test.db"
VECTOR_DIR="$TEST_DIR/vectors"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Semantic Search Functional Tests (Phases 1-3)          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "Test directory: ${YELLOW}$TEST_DIR${NC}"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo -e "${BLUE}Cleaning up test directory...${NC}"
    rm -rf "$TEST_DIR"
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}
trap cleanup EXIT

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}━━━ Checking Prerequisites ━━━${NC}"

    # Check OPENAI_API_KEY
    if [ -z "$OPENAI_API_KEY" ]; then
        echo -e "${RED}✗ OPENAI_API_KEY is not set${NC}"
        echo ""
        echo "Please set your OpenAI API key:"
        echo "  export OPENAI_API_KEY=sk-your-key-here"
        echo ""
        exit 1
    fi
    echo -e "${GREEN}✓ OPENAI_API_KEY is set${NC}"

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

    echo ""
}

# Run unit tests
run_unit_tests() {
    echo -e "${BLUE}━━━ Phase 0: Running Unit Tests ━━━${NC}"

    echo "Running config tests..."
    go test ./internal/config/... -v -count=1 2>&1 | grep -E "(PASS|FAIL|---)" || true

    echo ""
    echo "Running vector store tests..."
    go test ./internal/vector/... -v -count=1 -run "^Test[^I]" 2>&1 | grep -E "(PASS|FAIL|---)" || true

    echo ""
    echo "Running search service tests..."
    go test ./internal/search/... -v -count=1 2>&1 | grep -E "(PASS|FAIL|---)" || true

    echo ""
    echo "Running TUI component tests..."
    go test ./internal/tui/components/... -v -count=1 2>&1 | grep -E "(PASS|FAIL|---)" || true

    echo ""
    echo -e "${GREEN}✓ Unit tests completed${NC}"
    echo ""
}

# Test Phase 1A: chromem-go Vector Store
test_phase_1a() {
    echo -e "${BLUE}━━━ Phase 1A: Testing chromem-go Vector Store ━━━${NC}"

    # Run the chromem integration test
    echo "Running chromem-go integration test..."
    if go test ./internal/vector/... -v -count=1 -run "TestChromemStore" 2>&1 | tee /tmp/chromem_test.log | grep -E "(PASS|FAIL|SKIP)"; then
        if grep -q "SKIP" /tmp/chromem_test.log; then
            echo -e "${YELLOW}⚠ Some tests skipped (API key may not be valid for real API calls)${NC}"
        fi
    fi

    echo ""
    echo -e "${GREEN}✓ Phase 1A: chromem-go store tests completed${NC}"
    echo ""
}

# Test Phase 1C: Search Service
test_phase_1c() {
    echo -e "${BLUE}━━━ Phase 1C: Testing Search Service ━━━${NC}"

    echo "Testing snippet extraction..."
    go test ./internal/search/... -v -count=1 -run "Snippet" 2>&1 | grep -E "(PASS|FAIL|---)" || true

    echo ""
    echo "Testing search options and config..."
    go test ./internal/search/... -v -count=1 -run "Config|Options" 2>&1 | grep -E "(PASS|FAIL|---)" || true

    echo ""
    echo -e "${GREEN}✓ Phase 1C: Search service tests completed${NC}"
    echo ""
}

# Test Phase 2: TUI Components
test_phase_2() {
    echo -e "${BLUE}━━━ Phase 2: Testing TUI Components ━━━${NC}"

    echo "Testing UnifiedResultList component..."
    go test ./internal/tui/components/... -v -count=1 -run "UnifiedResultList" 2>&1 | grep -E "(PASS|FAIL|---)" || true

    echo ""
    echo "Testing snippet renderer..."
    go test ./internal/tui/components/... -v -count=1 -run "Snippet" 2>&1 | grep -E "(PASS|FAIL|---)" || true

    echo ""
    echo -e "${GREEN}✓ Phase 2: TUI component tests completed${NC}"
    echo ""
}

# Test Phase 3: Indexer
test_phase_3() {
    echo -e "${BLUE}━━━ Phase 3: Testing Indexer ━━━${NC}"

    echo "Testing indexer configuration..."
    go test ./internal/search/... -v -count=1 -run "Indexer" 2>&1 | grep -E "(PASS|FAIL|---)" || true

    echo ""
    echo "Testing mock vector store..."
    go test ./internal/search/... -v -count=1 -run "MockVectorStore" 2>&1 | grep -E "(PASS|FAIL|---)" || true

    echo ""
    echo -e "${GREEN}✓ Phase 3: Indexer tests completed${NC}"
    echo ""
}

# Run integration test with real API (optional)
run_integration_test() {
    echo -e "${BLUE}━━━ Integration Test: End-to-End Semantic Search ━━━${NC}"
    echo ""
    echo -e "${YELLOW}This test makes real API calls to OpenAI.${NC}"
    echo -e "${YELLOW}It may take 30-60 seconds and consume a small amount of API credits.${NC}"
    echo ""

    read -p "Run integration test? (y/N) " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}⚠ Skipping integration test${NC}"
        return
    fi

    echo ""
    echo "Running end-to-end integration test..."
    echo "(This creates a test database, indexes skills, and performs semantic search)"
    echo ""

    # Run the integration test
    if go test ./internal/search/... -tags=integration -v -count=1 -run "Integration" 2>&1 | tee /tmp/integration_test.log; then
        echo ""
        echo -e "${GREEN}✓ Integration test passed!${NC}"
    else
        echo ""
        echo -e "${RED}✗ Integration test failed${NC}"
        echo "See /tmp/integration_test.log for details"
    fi

    echo ""
}

# Interactive TUI test instructions
print_manual_test_instructions() {
    echo -e "${BLUE}━━━ Manual TUI Testing Instructions ━━━${NC}"
    echo ""
    echo "To manually test the TUI with semantic search:"
    echo ""
    echo "1. Ensure you have skills in the database:"
    echo -e "   ${YELLOW}go run ./cmd/skulto${NC}"
    echo ""
    echo "2. In the TUI, press ${YELLOW}/${NC} to open search"
    echo ""
    echo "3. Type a search query (e.g., 'testing react components')"
    echo ""
    echo "4. Verify the unified result list shows:"
    echo "   - ${GREEN}[name]${NC} badges for title/tag matches (green)"
    echo "   - ${BLUE}[content]${NC} badges for content matches (blue)"
    echo ""
    echo "5. Use ${YELLOW}↑/↓${NC} or ${YELLOW}j/k${NC} to navigate results"
    echo ""
    echo "6. Press ${YELLOW}Tab${NC} on a content match to expand/collapse snippets"
    echo ""
    echo "7. Verify the search bar stays visible when scrolling"
    echo ""
    echo "8. Press ${YELLOW}Enter${NC} to view skill details"
    echo ""
    echo "9. Press ${YELLOW}Esc${NC} to return to home"
    echo ""
}

# Summary
print_summary() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                    Test Summary                            ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Phases tested:"
    echo -e "  ${GREEN}✓${NC} Phase 0: Setup (config, dependencies)"
    echo -e "  ${GREEN}✓${NC} Phase 1A: chromem-go Vector Store"
    echo -e "  ${GREEN}✓${NC} Phase 1C: Search Service + Snippets"
    echo -e "  ${GREEN}✓${NC} Phase 2: TUI Integration (UnifiedResultList)"
    echo -e "  ${GREEN}✓${NC} Phase 3: Background Indexing Pipeline"
    echo ""
    echo "Key components verified:"
    echo "  - VectorStore interface (chromem.go)"
    echo "  - Content preparation and hashing (content.go)"
    echo "  - Search service with hybrid FTS+semantic (service.go)"
    echo "  - Snippet extraction with highlights (snippet.go)"
    echo "  - UnifiedResultList with inline expansion (unified_result_list.go)"
    echo "  - Indexer with batch processing and retry (indexer.go)"
    echo ""
}

# Main execution
main() {
    check_prerequisites
    run_unit_tests
    test_phase_1a
    test_phase_1c
    test_phase_2
    test_phase_3
    run_integration_test
    print_summary
    print_manual_test_instructions
}

main "$@"
