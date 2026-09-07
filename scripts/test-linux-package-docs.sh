#!/usr/bin/env bash
set -euo pipefail

# Script: test-linux-package-docs.sh
# Purpose: Keep Linux package installation and Makefile release guidance accurate.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

REQUIRED_ARGS=()
REQUIRED_ENV_VARS=()
REQUIRED_PROGRAMS=("grep" "make")

usage() {
  cat << 'EOF'
Usage: scripts/test-linux-package-docs.sh

Verifies the Linux package installation documentation and Makefile release
guidance. Run from any directory in the repository.
EOF
}

check_requirements() {
  local -r provided_arg_count=$1
  local missing=0

  if [ "${#REQUIRED_ARGS[@]}" -gt 0 ] && [ "$provided_arg_count" -lt "${#REQUIRED_ARGS[@]}" ]; then
    printf 'Error: Expected %s arguments (%s) but received %s.\n' \
      "${#REQUIRED_ARGS[@]}" "${REQUIRED_ARGS[*]}" "$provided_arg_count" >&2
    missing=1
  fi

  local env_var
  for env_var in "${REQUIRED_ENV_VARS[@]}"; do
    if [ -z "${!env_var:-}" ]; then
      printf 'Error: Missing required environment variable %s. Please set it before rerunning.\n' "$env_var" >&2
      missing=1
    fi
  done

  local program
  for program in "${REQUIRED_PROGRAMS[@]}"; do
    if ! command -v "$program" > /dev/null 2>&1; then
      printf 'Error: Required program %s is not installed or not on PATH. Please install it first.\n' "$program" >&2
      missing=1
    fi
  done

  if [ "$missing" -ne 0 ]; then
    printf '\n' >&2
    usage >&2
    return 1
  fi
}

assert_contains() {
  local file=$1
  local expected=$2

  if ! grep -F -- "$expected" "$file" > /dev/null; then
    printf 'FAIL: %s must contain %s\n' "$file" "$expected" >&2
    return 1
  fi
}

assert_not_contains() {
  local file=$1
  local unexpected=$2

  if grep -F -- "$unexpected" "$file" > /dev/null; then
    printf 'FAIL: %s must not contain %s\n' "$file" "$unexpected" >&2
    return 1
  fi
}

assert_linux_package_install_docs() {
  local file=$1

  assert_contains "$file" "### Linux Packages"
  assert_contains "$file" ".deb"
  assert_contains "$file" ".rpm"
  assert_contains "$file" "amd64"
  assert_contains "$file" "arm64"
  assert_contains "$file" "curl -LO"
  assert_contains "$file" "sha256sum -c checksums.txt --ignore-missing"
  assert_contains "$file" "sudo apt install ./"
  assert_contains "$file" "sudo dnf install ./"
  assert_contains "$file" "sudo zypper install ./"
  assert_contains "$file" "/usr/bin/skulto"
  assert_contains "$file" "/usr/bin/skulto-mcp"
  assert_contains "$file" "does not require system Git"
  assert_contains "$file" "not a configured package repository"
}

main() {
  check_requirements "$#" || exit 1

  assert_linux_package_install_docs "$PROJECT_ROOT/README.md"
  assert_linux_package_install_docs "$PROJECT_ROOT/docs/getting-started.md"
  assert_contains "$PROJECT_ROOT/docs/development.md" "nFPM"
  assert_contains "$PROJECT_ROOT/docs/development.md" "package inspection"
  assert_contains "$PROJECT_ROOT/docs/development.md" "Linux Debian and RPM packages"
  assert_not_contains "$PROJECT_ROOT/Makefile" "release-all"

  local help_output
  help_output=$(cd "$PROJECT_ROOT" && make help)
  if printf '%s\n' "$help_output" | grep -F -- "release-all" > /dev/null; then
    printf 'FAIL: make help must not advertise release-all\n' >&2
    return 1
  fi

  printf 'PASS: Linux package installation docs and Makefile release guidance\n'
}

main "$@"
