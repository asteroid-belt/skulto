// Package version provides build version information and semver utilities.
package version

import (
	"github.com/Masterminds/semver/v3"
)

var (
	parsedVersion  *semver.Version
	parseAttempted bool
)

// resetParsedVersion clears the cached parsed version for testing.
func resetParsedVersion() {
	parsedVersion = nil
	parseAttempted = false
}

// Parsed returns the parsed semantic version, or nil if unparseable.
// This is computed lazily on first call and cached.
func Parsed() *semver.Version {
	if parsedVersion != nil || parseAttempted {
		return parsedVersion
	}
	parseAttempted = true

	v, err := semver.NewVersion(Version)
	if err != nil {
		return nil
	}
	parsedVersion = v
	return parsedVersion
}

// IsPrerelease returns true if the current version is a pre-release.
// Returns false for unparseable versions (like "dev").
func IsPrerelease() bool {
	v := Parsed()
	if v == nil {
		return false
	}
	return v.Prerelease() != ""
}

// IsDevBuild returns true if this is a development build (no valid semver).
func IsDevBuild() bool {
	return Parsed() == nil
}

// Compare compares the current version to another version string.
// Returns: -1 if current < other, 0 if equal, 1 if current > other.
// Returns 0 if either version is unparseable.
func Compare(other string) int {
	current := Parsed()
	if current == nil {
		return 0
	}

	otherV, err := semver.NewVersion(other)
	if err != nil {
		return 0
	}

	return current.Compare(otherV)
}

// IsNewerThan returns true if the current version is newer than other.
// Returns false if either version is unparseable.
func IsNewerThan(other string) bool {
	return Compare(other) > 0
}

// Major returns the major version number, or 0 if unparseable.
func Major() uint64 {
	v := Parsed()
	if v == nil {
		return 0
	}
	return v.Major()
}

// Minor returns the minor version number, or 0 if unparseable.
func Minor() uint64 {
	v := Parsed()
	if v == nil {
		return 0
	}
	return v.Minor()
}

// Patch returns the patch version number, or 0 if unparseable.
func Patch() uint64 {
	v := Parsed()
	if v == nil {
		return 0
	}
	return v.Patch()
}

// SemverPrerelease returns the prerelease string (e.g., "beta.1"), or empty string.
// Named SemverPrerelease to avoid collision with any existing Prerelease function.
func SemverPrerelease() string {
	v := Parsed()
	if v == nil {
		return ""
	}
	return v.Prerelease()
}

// SemverMetadata returns the build metadata string, or empty string.
// Named SemverMetadata to avoid collision with any existing Metadata function.
func SemverMetadata() string {
	v := Parsed()
	if v == nil {
		return ""
	}
	return v.Metadata()
}
