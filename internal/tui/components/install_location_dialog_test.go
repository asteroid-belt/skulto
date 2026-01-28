package components

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/detect"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergePreferencesWithDetected_Ordering(t *testing.T) {
	allPlatforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCursor,
		installer.PlatformCline,
		installer.PlatformRooCode,
		installer.PlatformGoose,
	}

	saved := []string{"claude", "cline"}
	detected := []detect.DetectionResult{
		{Platform: installer.PlatformCursor, Detected: true},
		{Platform: installer.PlatformGoose, Detected: true},
	}

	result := mergePreferencesWithDetected(allPlatforms, saved, detected)
	require.Len(t, result, 5)

	// Saved first
	assert.Equal(t, installer.PlatformClaude, result[0])
	assert.Equal(t, installer.PlatformCline, result[1])
	// Detected (not in saved) next
	assert.Equal(t, installer.PlatformCursor, result[2])
	assert.Equal(t, installer.PlatformGoose, result[3])
	// Others last
	assert.Equal(t, installer.PlatformRooCode, result[4])
}

func TestMergePreferencesWithDetected_EmptySaved(t *testing.T) {
	allPlatforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCursor,
	}

	detected := []detect.DetectionResult{
		{Platform: installer.PlatformClaude, Detected: true},
	}

	result := mergePreferencesWithDetected(allPlatforms, nil, detected)
	require.Len(t, result, 2)

	// Detected first since no saved
	assert.Equal(t, installer.PlatformClaude, result[0])
	assert.Equal(t, installer.PlatformCursor, result[1])
}

func TestMergePreferencesWithDetected_NoDetection(t *testing.T) {
	allPlatforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCursor,
	}

	saved := []string{"cursor"}
	var detected []detect.DetectionResult

	result := mergePreferencesWithDetected(allPlatforms, saved, detected)
	require.Len(t, result, 2)

	// Saved first
	assert.Equal(t, installer.PlatformCursor, result[0])
	assert.Equal(t, installer.PlatformClaude, result[1])
}

func TestNewInstallLocationDialogWithPrefs_PreSelection(t *testing.T) {
	platforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCline,
		installer.PlatformRooCode,
	}

	saved := []string{"claude", "cline"}
	detected := []detect.DetectionResult{
		{Platform: installer.PlatformRooCode, Detected: true},
	}

	dialog := NewInstallLocationDialogWithPrefs(platforms, saved, detected)
	require.NotNil(t, dialog)

	// Should have options for all platforms (global + project each)
	locations := dialog.GetSelectedLocations()
	assert.NotEmpty(t, locations, "should have pre-selected locations")
}

func TestNewInstallLocationDialogWithPrefs_SavedAndDetectedPreSelected(t *testing.T) {
	platforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCline,
	}

	saved := []string{"claude"}
	detected := []detect.DetectionResult{
		{Platform: installer.PlatformCline, Detected: true},
	}

	dialog := NewInstallLocationDialogWithPrefs(platforms, saved, detected)
	require.NotNil(t, dialog)

	// Both saved and detected platforms should have global selected
	selected := dialog.GetSelectedLocations()
	var selectedPlatforms []string
	for _, loc := range selected {
		selectedPlatforms = append(selectedPlatforms, string(loc.Platform))
	}
	assert.Contains(t, selectedPlatforms, "claude", "saved platform should be pre-selected")
	assert.Contains(t, selectedPlatforms, "cline", "detected platform should be pre-selected")
}
