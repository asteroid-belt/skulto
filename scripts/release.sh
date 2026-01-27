#!/usr/bin/env bash
set -euo pipefail

# Script: release.sh
# Purpose: Build Skulto binaries for a specified platform
# Requirements: git, go

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Configuration
VALID_OS_TYPES="linux darwin"
VALID_ARCH_TYPES="amd64 arm64"
VALID_BINARIES="skulto skulto-mcp"
RELEASE_DIR="${PROJECT_ROOT}/release"
CGO_ENABLED=0

# Requirements
REQUIRED_ARGS=(BINARY)
REQUIRED_ENV_VARS=(VERSION SKULTO_POSTHOG_API_KEY GOOS GOARCH)
REQUIRED_PROGRAMS=("git" "go")

# Parsed arguments
BINARY=""

usage() {
  cat << 'EOF'
Usage: VERSION=<version> SKULTO_POSTHOG_API_KEY=<key> GOOS=<os> GOARCH=<arch> release.sh <binary>

Arguments:
  binary                   Binary to build: skulto or skulto-mcp [required]

Environment variables:
  VERSION                  Version string for the build [required]
  SKULTO_POSTHOG_API_KEY   PostHog API key for telemetry [required]
  GOOS                     Target OS (linux, darwin) [required]
  GOARCH                   Target architecture (amd64, arm64) [required]

External tools:
  git, go

Examples:
  VERSION=1.0.0 SKULTO_POSTHOG_API_KEY=key GOOS=linux GOARCH=amd64 ./scripts/release.sh skulto
  VERSION=1.0.0 SKULTO_POSTHOG_API_KEY=key GOOS=darwin GOARCH=arm64 ./scripts/release.sh skulto-mcp
EOF
}

check_requirements() {
  local -r provided_arg_count=$1
  local missing=0

  if [ ${#REQUIRED_ARGS[@]} -gt 0 ] && [ "$provided_arg_count" -lt ${#REQUIRED_ARGS[@]} ]; then
    printf 'Error: Expected %s argument(s) (%s) but received %s.\n' \
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

get_version_info() {
  COMMIT=$(git rev-parse --short HEAD 2> /dev/null || echo "unknown")
  BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  POSTHOG_API_KEY="${SKULTO_POSTHOG_API_KEY}"
}

get_ldflags() {
  printf '%s' "-s -w \
    -X github.com/asteroid-belt/skulto/pkg/version.Version=${VERSION} \
    -X github.com/asteroid-belt/skulto/pkg/version.Commit=${COMMIT} \
    -X github.com/asteroid-belt/skulto/pkg/version.BuildDate=${BUILD_DATE} \
    -X github.com/asteroid-belt/skulto/internal/telemetry.PostHogAPIKey=${POSTHOG_API_KEY} \
    -X github.com/asteroid-belt/skulto/internal/telemetry.Version=${VERSION}"
}

get_cmd_path() {
  local binary_name="$1"
  case "$binary_name" in
    skulto)     printf '%s' "./cmd/skulto" ;;
    skulto-mcp) printf '%s' "./cmd/skulto-mcp" ;;
    *)          printf 'Error: Unknown binary %s\n' "$binary_name" >&2; return 1 ;;
  esac
}

build_artifact() {
  local binary_name="$1"
  local cmd_path
  local ldflags
  local output_dir

  cmd_path=$(get_cmd_path "$binary_name")
  ldflags=$(get_ldflags)
  output_dir="${RELEASE_DIR}/${GOOS}/${GOARCH}"

  printf '📦 Building %s for %s-%s...\n' "$binary_name" "$GOOS" "$GOARCH"
  printf '   Version: %s\n' "$VERSION"
  printf '   Commit:  %s\n\n' "$COMMIT"

  mkdir -p "$output_dir"

  printf '  Building %s...\n' "$binary_name"

  CGO_ENABLED="$CGO_ENABLED" \
    go build -v -ldflags "$ldflags" -o "${output_dir}/${binary_name}" "$cmd_path"

  chmod +x "${output_dir}/${binary_name}"

  printf '\n✅ %s built for %s-%s\n' "$binary_name" "$GOOS" "$GOARCH"
}

validate_artifact() {
  local binary_name="$1"
  local artifact_path="${RELEASE_DIR}/${GOOS}/${GOARCH}/${binary_name}"
  local host_os
  local host_arch

  # Detect host OS
  case "$(uname -s)" in
    Linux)  host_os="linux" ;;
    Darwin) host_os="darwin" ;;
    *)      host_os="unknown" ;;
  esac

  # Detect host architecture
  case "$(uname -m)" in
    x86_64)  host_arch="amd64" ;;
    aarch64) host_arch="arm64" ;;
    arm64)   host_arch="arm64" ;;
    *)       host_arch="unknown" ;;
  esac

  printf '\n🔍 Validating %s...\n' "$binary_name"

  if [ "$GOOS" != "$host_os" ] || [ "$GOARCH" != "$host_arch" ]; then
    printf '   ⚠️  Skipping validation: cross-compiled binary (target: %s-%s, host: %s-%s)\n' \
      "$GOOS" "$GOARCH" "$host_os" "$host_arch"
    return 0
  fi

  local version_output
  if version_output=$("$artifact_path" --version 2>&1); then
    printf '   Version output: %s\n' "$version_output"
    printf '   ✅ %s validated successfully\n' "$binary_name"
  else
    printf '   ❌ Failed to run %s --version\n' "$binary_name" >&2
    printf '   Output: %s\n' "$version_output" >&2
    return 1
  fi
}

show_release_structure() {
  local output_dir="${RELEASE_DIR}/${GOOS}/${GOARCH}"
  printf '\n📁 Release structure:\n'
  if command -v tree > /dev/null 2>&1; then
    tree "$output_dir"
  else
    find "$output_dir" -type f
  fi
  printf '\n📍 Release location: %s/\n' "$output_dir"
}

parse_args() {
  if [ $# -lt 1 ]; then
    printf 'Error: Missing required argument: binary\n' >&2
    usage >&2
    return 1
  fi
  BINARY="$1"
}

validate_binary() {
  if ! echo "$VALID_BINARIES" | grep -qw "$BINARY"; then
    printf 'Error: Invalid binary "%s". Valid options: %s\n' "$BINARY" "$VALID_BINARIES" >&2
    printf '\n' >&2
    usage >&2
    return 1
  fi
}

build_single() {
  local binary_name="$1"
  build_artifact "$binary_name"
  validate_artifact "$binary_name"
}

validate_platform() {
  local valid=0

  if ! echo "$VALID_OS_TYPES" | grep -qw "$GOOS"; then
    printf 'Error: Invalid GOOS "%s". Valid options: %s\n' "$GOOS" "$VALID_OS_TYPES" >&2
    valid=1
  fi

  if ! echo "$VALID_ARCH_TYPES" | grep -qw "$GOARCH"; then
    printf 'Error: Invalid GOARCH "%s". Valid options: %s\n' "$GOARCH" "$VALID_ARCH_TYPES" >&2
    valid=1
  fi

  if [ "$valid" -ne 0 ]; then
    printf '\n' >&2
    usage >&2
    return 1
  fi
}

main() {
  parse_args "$@" || exit 1
  check_requirements "$#" || exit 1
  validate_platform || exit 1
  validate_binary || exit 1

  cd "$PROJECT_ROOT"
  get_version_info

  build_single "$BINARY"
  show_release_structure

  printf '🎉 Release complete!\n'
}

main "$@"
