#!/usr/bin/env bash
set -euo pipefail

# Script: test-linux-packages.sh
# Purpose: Exercise the Linux Debian and RPM package contract.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

REQUIRED_ARGS=()
REQUIRED_ENV_VARS=()
REQUIRED_PROGRAMS=("chmod" "grep" "mktemp" "sed")

RPM_DATABASE_DIRECTORY=""

usage() {
  cat << 'EOF'
Usage: scripts/test-linux-packages.sh

Creates fixture Skulto binaries and verifies Debian/RPM package names,
metadata, contents, and input-validation failures.

External tools:
  chmod, grep, mktemp, nfpm, dpkg-deb, rpm
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

require_program() {
  local program=$1
  if ! command -v "$program" > /dev/null 2>&1; then
    printf 'Error: Required program %s is not installed or not on PATH. Please install it first.\n' "$program" >&2
    return 1
  fi
}

assert_file() {
  local path=$1
  if [ ! -f "$path" ]; then
    printf 'FAIL: expected file %s\n' "$path" >&2
    return 1
  fi
}

assert_output_contains() {
  local description=$1
  local expected=$2
  local output=$3

  if ! printf '%s\n' "$output" | grep -F -- "$expected" > /dev/null; then
    printf 'FAIL: %s did not contain %s\n' "$description" "$expected" >&2
    return 1
  fi
}

assert_equal() {
  local description=$1
  local expected=$2
  local actual=$3

  if [ "$actual" != "$expected" ]; then
    printf 'FAIL: %s expected %s but received %s\n' "$description" "$expected" "$actual" >&2
    return 1
  fi
}

assert_precedes() {
  local before=$1
  local workflow=$2
  local after=$3
  local before_line
  local after_line

  before_line=$(printf '%s\n' "$workflow" | grep -nF -- "$before" | sed -n '1s/:.*//p')
  after_line=$(printf '%s\n' "$workflow" | grep -nF -- "$after" | sed -n '1s/:.*//p')

  if [ -z "$before_line" ] || [ -z "$after_line" ] || [ "$before_line" -ge "$after_line" ]; then
    printf 'FAIL: expected workflow command %s to run before %s\n' "$before" "$after" >&2
    return 1
  fi
}

assert_workflow_contains() {
  local workflow_name=$1
  local expected=$2
  local workflow=$3

  assert_output_contains "$workflow_name workflow" "$expected" "$workflow"
}

assert_workflow_contract() {
  local release_workflow
  local ci_workflow
  local package_command

  release_workflow=$(sed -n '/^  release:/,$p' "$PROJECT_ROOT/.github/workflows/release.yml")
  ci_workflow=$(sed -n '1,260p' "$PROJECT_ROOT/.github/workflows/ci.yml")

  assert_workflow_contains "release" "uses: actions/checkout@v4" "$release_workflow"
  assert_workflow_contains "release" "go-version: '1.25.3'" "$release_workflow"
  assert_workflow_contains "release" "rpm" "$release_workflow"
  assert_workflow_contains "release" "NFPM_VERSION=2.47.0" "$release_workflow"
  assert_workflow_contains "release" "goreleaser/nfpm" "$release_workflow"
  assert_workflow_contains "release" "GOTOOLCHAIN=go1.26.8" "$release_workflow"
  assert_workflow_contains "release" "source=\"\$artifact_dir/\$binary\"" "$release_workflow"
  assert_workflow_contains "release" "FATAL: Missing expected artifact" "$release_workflow"

  for package_command in \
    'scripts/package-linux.sh deb "$PACKAGE_VERSION" amd64' \
    'scripts/package-linux.sh deb "$PACKAGE_VERSION" arm64' \
    'scripts/package-linux.sh rpm "$PACKAGE_VERSION" amd64' \
    'scripts/package-linux.sh rpm "$PACKAGE_VERSION" arm64'; do
    assert_workflow_contains "release" "$package_command" "$release_workflow"
    assert_precedes "$package_command" "$release_workflow" 'gh release create "$VERSION"'
  done

  assert_workflow_contains "release" "scripts/validate-linux-packages.sh" "$release_workflow"
  assert_precedes "scripts/validate-linux-packages.sh" "$release_workflow" 'gh release create "$VERSION"'
  assert_workflow_contains "release" "release_assets=(" "$release_workflow"
  assert_workflow_contains "release" "dist/skulto-\${VERSION}-linux-amd64.tar.gz" "$release_workflow"
  assert_workflow_contains "release" "dist/skulto-\${VERSION}-linux-arm64.tar.gz" "$release_workflow"
  assert_workflow_contains "release" "dist/skulto-\${VERSION}-darwin-amd64.tar.gz" "$release_workflow"
  assert_workflow_contains "release" "dist/skulto-\${VERSION}-darwin-arm64.tar.gz" "$release_workflow"
  assert_workflow_contains "release" "dist/skulto_\${PACKAGE_VERSION}_linux_amd64.deb" "$release_workflow"
  assert_workflow_contains "release" "dist/skulto_\${PACKAGE_VERSION}_linux_arm64.deb" "$release_workflow"
  assert_workflow_contains "release" "dist/skulto-\${PACKAGE_VERSION}-1.x86_64.rpm" "$release_workflow"
  assert_workflow_contains "release" "dist/skulto-\${PACKAGE_VERSION}-1.aarch64.rpm" "$release_workflow"
  assert_workflow_contains "release" "sha256sum \"\${release_assets[@]}\"" "$release_workflow"
  assert_workflow_contains "release" "EXPECTED_ASSET_COUNT=9" "$release_workflow"
  assert_workflow_contains "release" "uploaded_asset_count" "$release_workflow"
  assert_workflow_contains "release" "--draft" "$release_workflow"
  assert_workflow_contains "release" "gh release upload \"\$VERSION\"" "$release_workflow"
  assert_workflow_contains "release" "gh release view \"\$VERSION\"" "$release_workflow"
  assert_workflow_contains "release" "gh release edit \"\$VERSION\"" "$release_workflow"
  assert_precedes "gh release create \"\$VERSION\"" "$release_workflow" 'gh release upload "$VERSION"'
  assert_precedes "gh release upload \"\$VERSION\"" "$release_workflow" 'gh release edit "$VERSION"'

  assert_workflow_contains "CI" "linux-packages:" "$ci_workflow"
  assert_workflow_contains "CI" "go-version: '1.25'" "$ci_workflow"
  assert_workflow_contains "CI" "rpm" "$ci_workflow"
  assert_workflow_contains "CI" "NFPM_VERSION=2.47.0" "$ci_workflow"
  assert_workflow_contains "CI" "GOTOOLCHAIN=go1.26.8" "$ci_workflow"
  assert_workflow_contains "CI" "GOOS=linux GOARCH=amd64 VERSION=v0.0.0-ci make release" "$ci_workflow"
  assert_workflow_contains "CI" "GOOS=linux GOARCH=arm64 VERSION=v0.0.0-ci make release" "$ci_workflow"
  assert_workflow_contains "CI" "SKULTO_POSTHOG_API_KEY: test" "$ci_workflow"
  assert_workflow_contains "CI" "bash scripts/test-linux-packages.sh" "$ci_workflow"
  printf 'PASS: release and CI workflow package contract\n'
}

assert_command_fails() {
  local description=$1
  shift

  if "$@" > /dev/null 2>&1; then
    printf 'FAIL: %s unexpectedly succeeded\n' "$description" >&2
    return 1
  fi

  printf 'PASS: %s\n' "$description"
}

write_fixture() {
  local path=$1
  cat > "$path" << 'EOF'
#!/bin/sh
exit 0
EOF
  chmod +x "$path"
}

assert_deb_package() {
  local package=$1
  local info
  local contents

  assert_file "$package"
  info=$(dpkg-deb --info "$package")
  contents=$(dpkg-deb --contents "$package")
  assert_output_contains "Debian package name" "Package: skulto" "$info"
  assert_output_contains "Debian package version" "Version: 1.2.3-1" "$info"
  assert_output_contains "Debian package skulto binary" "./usr/bin/skulto" "$contents"
  assert_output_contains "Debian package MCP binary" "./usr/bin/skulto-mcp" "$contents"
  printf 'PASS: %s\n' "$(basename "$package")"
}

assert_rpm_package() {
  local package=$1
  local info
  local contents

  assert_file "$package"
  info=$(rpm --dbpath "$RPM_DATABASE_DIRECTORY" -qpi "$package")
  contents=$(rpm --dbpath "$RPM_DATABASE_DIRECTORY" -qpl "$package")
  assert_equal "RPM package name" "skulto" "$(rpm --dbpath "$RPM_DATABASE_DIRECTORY" -qp --qf '%{NAME}' "$package")"
  assert_equal "RPM package version" "1.2.3" "$(rpm --dbpath "$RPM_DATABASE_DIRECTORY" -qp --qf '%{VERSION}' "$package")"
  assert_equal "RPM package release" "1" "$(rpm --dbpath "$RPM_DATABASE_DIRECTORY" -qp --qf '%{RELEASE}' "$package")"
  assert_equal "RPM package license" "MIT" "$(rpm --dbpath "$RPM_DATABASE_DIRECTORY" -qp --qf '%{LICENSE}' "$package")"
  assert_output_contains "RPM package skulto binary" "/usr/bin/skulto" "$contents"
  assert_output_contains "RPM package MCP binary" "/usr/bin/skulto-mcp" "$contents"
  printf 'PASS: %s\n' "$(basename "$package")"
}

main() {
  check_requirements "$#" || exit 1

  local temporary_directory
  local source_directory
  local output_directory
  local incomplete_source_directory
  local version=1.2.3

  temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/skulto-linux-packages.XXXXXX")
  temporary_directory=$(cd "$temporary_directory" && pwd)
  trap 'rm -rf "${temporary_directory:-}"' EXIT HUP INT TERM
  source_directory="$temporary_directory/source"
  output_directory="$temporary_directory/dist"
  incomplete_source_directory="$temporary_directory/incomplete-source"
  RPM_DATABASE_DIRECTORY="$temporary_directory/rpmdb"
  mkdir -p "$source_directory" "$output_directory" "$incomplete_source_directory"
  mkdir -p "$RPM_DATABASE_DIRECTORY"
  write_fixture "$source_directory/skulto"
  write_fixture "$source_directory/skulto-mcp"
  write_fixture "$incomplete_source_directory/skulto"

  local configuration
  configuration=$(sed -n '1,200p' "$PROJECT_ROOT/packaging/nfpm.yaml")
  assert_output_contains "nFPM package name" "name: skulto" "$configuration"
  assert_output_contains "nFPM package license declaration" "license: MIT" "$configuration"
  assert_workflow_contract

  "$PROJECT_ROOT/scripts/package-linux.sh" deb "$version" amd64 "$source_directory" "$output_directory"
  "$PROJECT_ROOT/scripts/package-linux.sh" deb "$version" arm64 "$source_directory" "$output_directory"
  "$PROJECT_ROOT/scripts/package-linux.sh" rpm "$version" amd64 "$source_directory" "$output_directory"
  "$PROJECT_ROOT/scripts/package-linux.sh" rpm "$version" arm64 "$source_directory" "$output_directory"

  require_program dpkg-deb
  require_program rpm
  assert_deb_package "$output_directory/skulto_${version}_linux_amd64.deb"
  assert_deb_package "$output_directory/skulto_${version}_linux_arm64.deb"
  assert_rpm_package "$output_directory/skulto-${version}-1.x86_64.rpm"
  assert_rpm_package "$output_directory/skulto-${version}-1.aarch64.rpm"

  "$PROJECT_ROOT/scripts/validate-linux-packages.sh" \
    "$output_directory/skulto_${version}_linux_amd64.deb" \
    "$output_directory/skulto_${version}_linux_arm64.deb" \
    "$output_directory/skulto-${version}-1.x86_64.rpm" \
    "$output_directory/skulto-${version}-1.aarch64.rpm"

  assert_command_fails "missing skulto-mcp fixture" \
    "$PROJECT_ROOT/scripts/package-linux.sh" deb "$version" amd64 "$incomplete_source_directory" "$output_directory"
  assert_command_fails "unsupported package format" \
    "$PROJECT_ROOT/scripts/package-linux.sh" archive "$version" amd64 "$source_directory" "$output_directory"
  assert_command_fails "missing package validation input" \
    "$PROJECT_ROOT/scripts/validate-linux-packages.sh" \
    "$output_directory/skulto_${version}_linux_amd64.deb" \
    "$output_directory/skulto_${version}_linux_arm64.deb" \
    "$output_directory/skulto-${version}-1.x86_64.rpm" \
    "$output_directory/missing.rpm"
}

main "$@"
