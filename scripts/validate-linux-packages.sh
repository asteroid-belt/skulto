#!/usr/bin/env bash
set -euo pipefail

# Script: validate-linux-packages.sh
# Purpose: Validate metadata and installed binary paths in Skulto Linux packages.

REQUIRED_ARGS=(amd64-deb arm64-deb amd64-rpm arm64-rpm)
REQUIRED_ENV_VARS=()
REQUIRED_PROGRAMS=("basename" "dpkg-deb" "grep" "mktemp" "rpm" "tar")

RPM_DATABASE_DIRECTORY=""

usage() {
  cat << 'EOF'
Usage: scripts/validate-linux-packages.sh <amd64.deb> <arm64.deb> <amd64.rpm> <arm64.rpm>

Validates exact artifact names, package metadata, binary paths, and absence of
lifecycle scripts for the four packages produced by scripts/package-linux.sh.

External tools:
  basename, dpkg-deb, grep, rpm, tar
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

assert_contains() {
  local description=$1
  local expected=$2
  local output=$3

  if ! printf '%s\n' "$output" | grep -F -- "$expected" > /dev/null; then
    fail "$description is missing '$expected'." || return 1
  fi
}

assert_no_lifecycle_scripts() {
  local package=$1
  local control_entries
  local lifecycle_entries

  control_entries=$(dpkg-deb --ctrl-tarfile "$package" | tar -tf -)
  lifecycle_entries=$(printf '%s\n' "$control_entries" | grep -E '(^|/)(preinst|postinst|prerm|postrm)$' || true)
  if [ -n "$lifecycle_entries" ]; then
    fail "Debian package '$package' declares lifecycle scripts." || return 1
  fi
}

rpm_query() {
  rpm --dbpath "$RPM_DATABASE_DIRECTORY" "$@"
}

validate_deb() {
  local package=$1
  local expected_architecture=$2
  local base
  local version
  local contents

  [ -f "$package" ] || fail "Expected Debian package '$package' does not exist." || return 1
  base=$(basename "$package")
  case "$base" in
    skulto_*_linux_"$expected_architecture".deb) ;;
    *) fail "Debian package '$base' does not match the expected $expected_architecture filename." || return 1 ;;
  esac
  version=${base#skulto_}
  version=${version%_linux_"$expected_architecture".deb}

  [ "$(dpkg-deb --field "$package" Package)" = "skulto" ] || fail "Debian package '$base' has an unexpected name." || return 1
  [ "$(dpkg-deb --field "$package" Version)" = "$version-1" ] || fail "Debian package '$base' has an unexpected version." || return 1
  [ "$(dpkg-deb --field "$package" Architecture)" = "$expected_architecture" ] || fail "Debian package '$base' has an unexpected architecture." || return 1
  contents=$(dpkg-deb --contents "$package")
  assert_contains "Debian package '$base'" "./usr/bin/skulto" "$contents"
  assert_contains "Debian package '$base'" "./usr/bin/skulto-mcp" "$contents"
  assert_no_lifecycle_scripts "$package"
}

validate_rpm() {
  local package=$1
  local expected_architecture=$2
  local base
  local version
  local contents
  local lifecycle_scripts

  [ -f "$package" ] || fail "Expected RPM package '$package' does not exist." || return 1
  base=$(basename "$package")
  case "$base" in
    skulto-*-1."$expected_architecture".rpm) ;;
    *) fail "RPM package '$base' does not match the expected $expected_architecture filename." || return 1 ;;
  esac
  version=${base#skulto-}
  version=${version%-1."$expected_architecture".rpm}

  contents=$(rpm_query -qpl "$package")
  [ "$(rpm_query -qp --qf '%{NAME}' "$package")" = "skulto" ] || fail "RPM package '$base' has an unexpected name." || return 1
  [ "$(rpm_query -qp --qf '%{VERSION}' "$package")" = "$version" ] || fail "RPM package '$base' has an unexpected version." || return 1
  [ "$(rpm_query -qp --qf '%{RELEASE}' "$package")" = "1" ] || fail "RPM package '$base' has an unexpected release." || return 1
  [ "$(rpm_query -qp --qf '%{LICENSE}' "$package")" = "MIT" ] || fail "RPM package '$base' has an unexpected license." || return 1
  [ "$(rpm_query -qp --qf '%{ARCH}' "$package")" = "$expected_architecture" ] || fail "RPM package '$base' has an unexpected architecture." || return 1
  assert_contains "RPM package '$base'" "/usr/bin/skulto" "$contents"
  assert_contains "RPM package '$base'" "/usr/bin/skulto-mcp" "$contents"
  lifecycle_scripts=$(rpm_query -qp --scripts "$package")
  if [ -n "$lifecycle_scripts" ]; then
    fail "RPM package '$base' declares lifecycle scripts." || return 1
  fi
}

main() {
  check_requirements "$#" || exit 1

  if [ "$#" -ne "${#REQUIRED_ARGS[@]}" ]; then
    printf 'Error: Expected exactly %s arguments but received %s.\n' "${#REQUIRED_ARGS[@]}" "$#" >&2
    usage >&2
    exit 1
  fi

  RPM_DATABASE_DIRECTORY=$(mktemp -d "${TMPDIR:-/tmp}/skulto-rpmdb.XXXXXX")
  RPM_DATABASE_DIRECTORY=$(cd "$RPM_DATABASE_DIRECTORY" && pwd)
  trap 'rm -rf "$RPM_DATABASE_DIRECTORY"' EXIT HUP INT TERM

  validate_deb "$1" amd64
  validate_deb "$2" arm64
  validate_rpm "$3" x86_64
  validate_rpm "$4" aarch64
  printf 'Validated Linux packages.\n'
}

main "$@"
