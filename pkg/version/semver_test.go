package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsed_ValidSemver(t *testing.T) {
	tests := []struct {
		version    string
		wantMajor  uint64
		wantMinor  uint64
		wantPatch  uint64
		wantPrerel string
		wantMeta   string
	}{
		{"v1.0.0", 1, 0, 0, "", ""},
		{"v1.2.3", 1, 2, 3, "", ""},
		{"v0.1.0", 0, 1, 0, "", ""},
		{"v1.0.0-beta.1", 1, 0, 0, "beta.1", ""},
		{"v1.0.0-rc.2", 1, 0, 0, "rc.2", ""},
		{"v1.0.0-alpha", 1, 0, 0, "alpha", ""},
		{"v1.0.0+build123", 1, 0, 0, "", "build123"},
		{"v1.0.0-beta.1+build456", 1, 0, 0, "beta.1", "build456"},
		{"1.0.0", 1, 0, 0, "", ""}, // without v prefix
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			// Reset cached version
			resetParsedVersion()
			Version = tt.version

			v := Parsed()
			assert.NotNil(t, v, "should parse %s", tt.version)
			assert.Equal(t, tt.wantMajor, v.Major())
			assert.Equal(t, tt.wantMinor, v.Minor())
			assert.Equal(t, tt.wantPatch, v.Patch())
			assert.Equal(t, tt.wantPrerel, v.Prerelease())
			assert.Equal(t, tt.wantMeta, v.Metadata())
		})
	}
}

func TestParsed_InvalidVersion(t *testing.T) {
	tests := []string{
		"dev",
		"unknown",
		"",
		"not-a-version",
		"v1.0.0.0", // too many parts
	}

	for _, version := range tests {
		t.Run(version, func(t *testing.T) {
			resetParsedVersion()
			Version = version

			v := Parsed()
			assert.Nil(t, v, "should not parse %s", version)
		})
	}
}

func TestIsPrerelease(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"v1.0.0", false},
		{"v1.0.0-beta.1", true},
		{"v1.0.0-rc.2", true},
		{"v1.0.0-alpha", true},
		{"v1.0.0+build123", false}, // metadata only, not prerelease
		{"dev", false},             // unparseable
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			resetParsedVersion()
			Version = tt.version

			assert.Equal(t, tt.want, IsPrerelease())
		})
	}
}

func TestIsDevBuild(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"v1.0.0", false},
		{"dev", true},
		{"unknown", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			resetParsedVersion()
			Version = tt.version

			assert.Equal(t, tt.want, IsDevBuild())
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		current string
		other   string
		want    int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"v1.0.1", "v1.0.0", 1},
		{"v1.0.0", "v1.0.1", -1},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.0.0", "v1.0.0-beta.1", 1}, // release > prerelease
		{"v1.0.0-beta.2", "v1.0.0-beta.1", 1},
		{"dev", "v1.0.0", 0}, // unparseable returns 0
		{"v1.0.0", "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.other, func(t *testing.T) {
			resetParsedVersion()
			Version = tt.current

			assert.Equal(t, tt.want, Compare(tt.other))
		})
	}
}

func TestIsNewerThan(t *testing.T) {
	tests := []struct {
		current string
		other   string
		want    bool
	}{
		{"v1.0.1", "v1.0.0", true},
		{"v1.0.0", "v1.0.1", false},
		{"v1.0.0", "v1.0.0", false},
		{"v2.0.0", "v1.0.0-beta.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_newer_than_"+tt.other, func(t *testing.T) {
			resetParsedVersion()
			Version = tt.current

			assert.Equal(t, tt.want, IsNewerThan(tt.other))
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	resetParsedVersion()
	Version = "v1.2.3-beta.1+build456"

	assert.Equal(t, uint64(1), Major())
	assert.Equal(t, uint64(2), Minor())
	assert.Equal(t, uint64(3), Patch())
	assert.Equal(t, "beta.1", SemverPrerelease())
	assert.Equal(t, "build456", SemverMetadata())
}

func TestHelperFunctions_DevBuild(t *testing.T) {
	resetParsedVersion()
	Version = "dev"

	assert.Equal(t, uint64(0), Major())
	assert.Equal(t, uint64(0), Minor())
	assert.Equal(t, uint64(0), Patch())
	assert.Equal(t, "", SemverPrerelease())
	assert.Equal(t, "", SemverMetadata())
}
