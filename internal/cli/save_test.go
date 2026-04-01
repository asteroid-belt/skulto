package cli

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/manifest"
	"github.com/stretchr/testify/assert"
)

func TestFilterIgnored_FiltersCorrectly(t *testing.T) {
	existing := &manifest.ManifestFile{
		Ignored: []string{"a", "b"},
	}
	input := []installer.UnmanagedEntry{
		{Name: "a"}, {Name: "c"}, {Name: "d"},
	}
	got := filterIgnored(input, existing)
	assert.Len(t, got, 2)
	assert.Equal(t, "c", got[0].Name)
	assert.Equal(t, "d", got[1].Name)
}

func TestFilterIgnored_NilManifest(t *testing.T) {
	input := []installer.UnmanagedEntry{{Name: "a"}, {Name: "b"}}
	got := filterIgnored(input, nil)
	assert.Len(t, got, 2)
}

func TestFilterIgnored_EmptyIgnored(t *testing.T) {
	existing := &manifest.ManifestFile{Ignored: []string{}}
	input := []installer.UnmanagedEntry{{Name: "a"}}
	got := filterIgnored(input, existing)
	assert.Len(t, got, 1)
}
