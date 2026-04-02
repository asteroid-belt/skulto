#!/usr/bin/env bash
# Skulto Release Certification — Pass 1: Unit Tests, Lint, Cross-Compile
# Exit on first failure.
set -euo pipefail

PASS=0
FAIL=0
REPO_ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
SKULTO="$REPO_ROOT/build/skulto"

pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1"; echo "        $2"; FAIL=$((FAIL + 1)); exit 1; }

echo "PASS 1: Unit Tests, Lint, Cross-Compile"
echo "========================================"
echo ""

# Build
echo "[1/7] make build-all"
if make build-all > /dev/null 2>&1; then
  pass "make build-all"
else
  fail "make build-all" "Build failed"
fi

# Tests
echo "[2/7] make test"
test_output=$(make test 2>&1)
if echo "$test_output" | grep -q "^FAIL"; then
  fail "make test" "$(echo "$test_output" | grep "^FAIL")"
else
  pass "make test"
fi

# Lint
echo "[3/7] make lint"
lint_output=$(make lint 2>&1)
if echo "$lint_output" | grep -q "0 issues"; then
  pass "make lint"
else
  fail "make lint" "$lint_output"
fi

# Format
echo "[4/7] make format"
fmt_output=$(make format 2>&1)
# Check if any files were reformatted (go fmt prints changed filenames)
changed=$(echo "$fmt_output" | grep -E "^internal/|^cmd/" || true)
if [ -n "$changed" ]; then
  fail "make format" "Files need formatting: $changed"
else
  pass "make format"
fi

# Cross-compile
echo "[5/7] cross-compile linux/amd64"
if GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /dev/null ./cmd/skulto 2>&1; then
  pass "linux/amd64"
else
  fail "linux/amd64" "Cross-compile failed"
fi

echo "[6/7] cross-compile linux/arm64"
if GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /dev/null ./cmd/skulto 2>&1; then
  pass "linux/arm64"
else
  fail "linux/arm64" "Cross-compile failed"
fi

echo "[7/7] cross-compile darwin (amd64 + arm64)"
if GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o /dev/null ./cmd/skulto 2>&1 && \
   GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o /dev/null ./cmd/skulto 2>&1; then
  pass "darwin/amd64 + darwin/arm64"
else
  fail "darwin cross-compile" "Cross-compile failed"
fi

echo ""
echo "Pass 1 complete: $PASS passed, $FAIL failed"
