package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImportDialog(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)

	require.NotNil(t, dialog)
	assert.Equal(t, "my-skill", dialog.skillName)
	assert.Equal(t, "/path/to/skill", dialog.skillPath)
	assert.False(t, dialog.hasConflict)
	assert.Equal(t, 0, dialog.selectedIdx)
}

func TestNewImportDialog_WithConflict(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", true)

	require.NotNil(t, dialog)
	assert.True(t, dialog.hasConflict)
}

func TestImportDialog_Render_Normal(t *testing.T) {
	// Test dialog displays skill info and Import/Cancel options
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)
	view := dialog.View()

	assert.Contains(t, view, "my-skill")
	assert.Contains(t, view, "Import")
	assert.Contains(t, view, "Cancel")
}

func TestImportDialog_Render_Conflict(t *testing.T) {
	// Test conflict mode shows Rename/Skip/Replace options
	dialog := NewImportDialog("my-skill", "/path/to/skill", true)
	view := dialog.View()

	assert.Contains(t, view, "Rename")
	assert.Contains(t, view, "Skip")
	assert.Contains(t, view, "Replace")
}

func TestImportDialog_Navigation_Right(t *testing.T) {
	// Test arrow key navigation
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)
	assert.Equal(t, 0, dialog.selectedIdx)

	// Right moves to next option
	msg := tea.KeyMsg{Type: tea.KeyRight}
	dialog.Update(msg)
	assert.Equal(t, 1, dialog.selectedIdx)
}

func TestImportDialog_Navigation_Left(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)
	dialog.selectedIdx = 1

	// Left moves to previous option
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	dialog.Update(msg)
	assert.Equal(t, 0, dialog.selectedIdx)
}

func TestImportDialog_Navigation_Wraps(t *testing.T) {
	// Normal mode has 2 options (Import, Cancel)
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)
	assert.Equal(t, 0, dialog.selectedIdx)

	// Left from first wraps to last
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	dialog.Update(msg)
	assert.Equal(t, 1, dialog.selectedIdx)

	// Right from last wraps to first
	msg = tea.KeyMsg{Type: tea.KeyRight}
	dialog.Update(msg)
	assert.Equal(t, 0, dialog.selectedIdx)
}

func TestImportDialog_Navigation_ConflictMode(t *testing.T) {
	// Conflict mode has 3 options (Rename, Skip, Replace)
	dialog := NewImportDialog("my-skill", "/path/to/skill", true)

	// Navigate through all options
	msg := tea.KeyMsg{Type: tea.KeyRight}
	dialog.Update(msg)
	assert.Equal(t, 1, dialog.selectedIdx)

	dialog.Update(msg)
	assert.Equal(t, 2, dialog.selectedIdx)

	// Wraps to first
	dialog.Update(msg)
	assert.Equal(t, 0, dialog.selectedIdx)
}

func TestImportDialog_VimNavigation(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)

	// 'l' moves right
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	dialog.Update(msg)
	assert.Equal(t, 1, dialog.selectedIdx)

	// 'h' moves left
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	dialog.Update(msg)
	assert.Equal(t, 0, dialog.selectedIdx)
}

func TestImportDialog_EnterConfirms(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)
	assert.False(t, dialog.IsConfirmed())
	assert.False(t, dialog.IsCancelled())

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(msg)

	assert.True(t, dialog.IsConfirmed())
	assert.False(t, dialog.IsCancelled())
}

func TestImportDialog_EscCancels(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)
	assert.False(t, dialog.IsCancelled())

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	dialog.Update(msg)

	assert.True(t, dialog.IsCancelled())
}

func TestImportDialog_GetAction_Normal(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)

	// First option is Import
	assert.Equal(t, ImportDialogImport, dialog.GetAction())

	// Navigate to Cancel
	msg := tea.KeyMsg{Type: tea.KeyRight}
	dialog.Update(msg)
	assert.Equal(t, ImportDialogCancel, dialog.GetAction())
}

func TestImportDialog_GetAction_Conflict(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", true)

	// First option is Rename
	assert.Equal(t, ImportDialogRename, dialog.GetAction())

	// Navigate to Skip
	msg := tea.KeyMsg{Type: tea.KeyRight}
	dialog.Update(msg)
	assert.Equal(t, ImportDialogSkip, dialog.GetAction())

	// Navigate to Replace
	dialog.Update(msg)
	assert.Equal(t, ImportDialogReplace, dialog.GetAction())
}

func TestImportDialog_RenameMode(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", true)
	assert.False(t, dialog.IsRenaming())

	// Confirm on Rename option
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(msg)

	assert.True(t, dialog.IsRenaming())
	assert.False(t, dialog.IsConfirmed())
}

func TestImportDialog_RenameInput(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", true)

	// Enter rename mode
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(msg)
	assert.True(t, dialog.IsRenaming())

	// Type a new name
	for _, r := range "new-name" {
		runeMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		dialog.Update(runeMsg)
	}

	assert.Equal(t, "new-name", dialog.GetNewName())
}

func TestImportDialog_RenameConfirm(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", true)

	// Enter rename mode
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(msg)

	// Type a new name
	for _, r := range "new-name" {
		runeMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		dialog.Update(runeMsg)
	}

	// Confirm the rename
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(msg)

	assert.True(t, dialog.IsConfirmed())
	assert.Equal(t, "new-name", dialog.GetNewName())
}

func TestImportDialog_RenameEscGoesBack(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", true)

	// Enter rename mode
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(msg)
	assert.True(t, dialog.IsRenaming())

	// Press Esc to go back
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	dialog.Update(msg)

	assert.False(t, dialog.IsRenaming())
	assert.False(t, dialog.IsCancelled())
}

func TestImportDialog_Reset(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)

	// Modify state
	dialog.selectedIdx = 1
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(msg)

	// Reset
	dialog.Reset()

	assert.Equal(t, 0, dialog.selectedIdx)
	assert.False(t, dialog.IsConfirmed())
	assert.False(t, dialog.IsCancelled())
	assert.False(t, dialog.IsRenaming())
	assert.Empty(t, dialog.GetNewName())
}

func TestImportDialog_SetWidth(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)
	dialog.SetWidth(80)

	assert.Equal(t, 80, dialog.width)
}

func TestImportDialog_CenteredView(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)

	view := dialog.CenteredView(100, 50)

	// View should be non-empty
	assert.NotEmpty(t, view)
}

func TestImportDialog_SkillPath_Shown(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)
	view := dialog.View()

	assert.Contains(t, view, "/path/to/skill")
}

func TestImportDialog_ConflictWarning(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", true)
	view := dialog.View()

	// Should show conflict warning
	assert.Contains(t, view, "already exists")
}

func TestImportDialog_TabNavigation(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)

	// Tab moves to next option
	msg := tea.KeyMsg{Type: tea.KeyTab}
	dialog.Update(msg)
	assert.Equal(t, 1, dialog.selectedIdx)

	// Shift+Tab moves to previous
	msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	dialog.Update(msg)
	assert.Equal(t, 0, dialog.selectedIdx)
}

func TestImportDialog_HandleKey(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", false)

	// Test string-based key handling for compatibility
	dialog.HandleKey("right")
	assert.Equal(t, 1, dialog.selectedIdx)

	dialog.HandleKey("left")
	assert.Equal(t, 0, dialog.selectedIdx)

	dialog.HandleKey("enter")
	assert.True(t, dialog.IsConfirmed())
}

func TestImportDialog_RenameBackspace(t *testing.T) {
	dialog := NewImportDialog("my-skill", "/path/to/skill", true)

	// Enter rename mode
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(msg)

	// Type something
	for _, r := range "abc" {
		runeMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		dialog.Update(runeMsg)
	}
	assert.Equal(t, "abc", dialog.GetNewName())

	// Backspace removes last character
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	dialog.Update(msg)
	assert.Equal(t, "ab", dialog.GetNewName())
}
