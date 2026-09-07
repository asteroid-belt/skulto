#!/usr/bin/env bash
set -euo pipefail

# Script: package-linux.sh
# Purpose: Create a Debian or RPM package from verified Linux binaries.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

REQUIRED_ARGS=(format version architecture source-directory output-directory)
REQUIRED_ENV_VARS=()
REQUIRED_PROGRAMS=("nfpm")

usage() {
  cat << 'EOF'
Usage: scripts/package-linux.sh <deb|rpm> <version> <amd64|arm64> <source-directory> <output-directory>

Required arguments:
  format             Package format: deb or rpm
  version            Normalized release version without a leading v
  architecture       Go architecture: amd64 or arm64
  source-directory   Directory containing executable skulto and skulto-mcp
  output-directory   Directory where the package will be written

External tools:
  nfpm

Examples:
  scripts/package-linux.sh deb 1.2.3 amd64 release/linux/amd64 dist
  scripts/package-linux.sh rpm 1.2.3 arm64 release/linux/arm64 dist
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

fail() {
  printf 'Error: %s\n' "$1" >&2
  return 1
}

package_filename() {
  local format=$1
  local version=$2
  local architecture=$3

  case "$format:$architecture" in
    deb:amd64) printf 'skulto_%s_linux_amd64.deb' "$version" ;;
    deb:arm64) printf 'skulto_%s_linux_arm64.deb' "$version" ;;
    rpm:amd64) printf 'skulto-%s-1.x86_64.rpm' "$version" ;;
    rpm:arm64) printf 'skulto-%s-1.aarch64.rpm' "$version" ;;
    *) return 1 ;;
  esac
}

validate_arguments() {
  local format=$1
  local version=$2
  local architecture=$3
  local source_directory=$4

  case "$format" in
    deb | rpm) ;;
    *) fail "Unsupported package format '$format'. Use deb or rpm." || return 1 ;;
  esac

  if [ -z "$version" ] || [ "${version#v}" != "$version" ]; then
    fail "Version '$version' must be normalized and must not begin with v." || return 1
  fi

  case "$architecture" in
    amd64 | arm64) ;;
    *) fail "Unsupported architecture '$architecture'. Use amd64 or arm64." || return 1 ;;
  esac

  if [ ! -d "$source_directory" ]; then
    fail "Source directory '$source_directory' does not exist." || return 1
  fi

  local binary
  for binary in skulto skulto-mcp; do
    if [ ! -f "$source_directory/$binary" ] || [ ! -x "$source_directory/$binary" ]; then
      fail "Expected executable '$source_directory/$binary' is missing." || return 1
    fi
  done
}

main() {
  check_requirements "$#" || exit 1

  if [ "$#" -ne "${#REQUIRED_ARGS[@]}" ]; then
    printf 'Error: Expected exactly %s arguments but received %s.\n' "${#REQUIRED_ARGS[@]}" "$#" >&2
    usage >&2
    exit 1
  fi

  local format=$1
  local version=$2
  local architecture=$3
  local source_directory=$4
  local output_directory=$5
  local filename
  local target

  validate_arguments "$format" "$version" "$architecture" "$source_directory" || exit 1
  filename=$(package_filename "$format" "$version" "$architecture") || {
    fail "Unable to determine package filename."
    exit 1
  }
  mkdir -p "$output_directory"
  target="$output_directory/$filename"

  SKULTO_PACKAGE_ARCH="$architecture" \
    SKULTO_PACKAGE_VERSION="$version" \
    SKULTO_PACKAGE_SOURCE_DIR="$source_directory" \
    nfpm package --config "$PROJECT_ROOT/packaging/nfpm.yaml" --packager "$format" --target "$target"

  printf 'Created package: %s\n' "$target"
}

main "$@"
