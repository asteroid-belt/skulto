package components

import (
	"fmt"
	"testing"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/skillgen"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNewSkillDialog(t *testing.T) {
	dialog := NewNewSkillDialog()

	require.NotNil(t, dialog)
	assert.Equal(t, StateInput, dialog.state)
	assert.False(t, dialog.cancelled)
	assert.False(t, dialog.confirmed)
}

func TestNewSkillDialog_SetSize(t *testing.T) {
	dialog := NewNewSkillDialog()

	dialog.SetSize(100, 50)

	assert.Equal(t, 100, dialog.width)
	assert.Equal(t, 50, dialog.height)
}

func TestNewSkillDialog_Cancel(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)

	// Press Escape
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	dialog.Update(escMsg)

	assert.True(t, dialog.IsCancelled())
}

func TestNewSkillDialog_TabNavigation(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)

	// Initial focus on prompt
	assert.Equal(t, 0, dialog.focusedField)

	// Tab to tool selector
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	dialog.Update(tabMsg)
	assert.Equal(t, 1, dialog.focusedField)

	// Tab to generate button
	dialog.Update(tabMsg)
	assert.Equal(t, 2, dialog.focusedField)

	// Tab wraps back to prompt
	dialog.Update(tabMsg)
	assert.Equal(t, 0, dialog.focusedField)
}

func TestNewSkillDialog_ToolSelection(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)

	// Set up some mock tools
	dialog.toolOptions = []skillgen.AITool{
		skillgen.AIToolClaude,
		skillgen.AIToolCodex,
	}
	dialog.toolIndex = 0
	dialog.selectedTool = skillgen.AIToolClaude

	// Focus on tool selector
	dialog.focusedField = 1

	// Press right to change tool
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	dialog.Update(rightMsg)

	assert.Equal(t, 1, dialog.toolIndex)
	assert.Equal(t, skillgen.AIToolCodex, dialog.selectedTool)
}

func TestNewSkillDialog_Reset(t *testing.T) {
	dialog := NewNewSkillDialog()

	// Modify state
	dialog.state = StatePreview
	dialog.generatedContent = "test content"
	dialog.cancelled = true

	// Reset
	dialog.Reset()

	assert.Equal(t, StateInput, dialog.state)
	assert.Empty(t, dialog.generatedContent)
	assert.False(t, dialog.cancelled)
	assert.False(t, dialog.confirmed)
}

func TestNewSkillDialog_ViewInput(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateInput

	view := dialog.View()

	assert.Contains(t, view, "Quick Skill Creator")
	assert.Contains(t, view, "Describe")
}

func TestNewSkillDialog_ViewGenerating(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateGenerating
	dialog.selectedTool = skillgen.AIToolClaude

	view := dialog.View()

	assert.Contains(t, view, "Building Skill")
	assert.Contains(t, view, "claude")
	assert.Contains(t, view, "Interactive session running")
}

func TestNewSkillDialog_ViewLaunching(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateLaunching
	dialog.selectedTool = skillgen.AIToolClaude

	view := dialog.View()

	assert.Contains(t, view, "Ready to Launch")
	assert.Contains(t, view, "claude")
	assert.Contains(t, view, "How to Exit")
	assert.Contains(t, view, "Ctrl+C")
	assert.Contains(t, view, "Press Enter to Start")
	assert.Contains(t, view, "Cancel")
}

func TestNewSkillDialog_LaunchingStateNavigation(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateLaunching
	dialog.launching = true

	// Press Esc to go back to input
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	dialog.Update(escMsg)

	assert.Equal(t, StateInput, dialog.state)
	assert.False(t, dialog.launching)
}

func TestNewSkillDialog_ViewPreview(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StatePreview
	dialog.generatedContent = "---\nname: test-skill\n---\n# Test"
	dialog.previewLines = []string{"---", "name: test-skill", "---", "# Test"}

	view := dialog.View()

	assert.Contains(t, view, "Generated")
	assert.Contains(t, view, "Save")
}

func TestNewSkillDialog_ViewError(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateError
	dialog.err = fmt.Errorf("test error")

	view := dialog.View()

	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "test error")
}

func TestNewSkillDialog_PreviewScroll(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StatePreview

	// Create many lines
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = fmt.Sprintf("Line %d", i)
	}
	dialog.previewLines = lines

	// Scroll down
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	dialog.Update(downMsg)

	assert.Equal(t, 1, dialog.previewScroll)

	// Scroll up
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	dialog.Update(upMsg)

	assert.Equal(t, 0, dialog.previewScroll)
}

func TestNewSkillDialog_ConfirmFromPreview(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StatePreview
	dialog.generatedContent = "test content"

	// Press Ctrl+S to confirm
	ctrlSMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
	dialog.Update(ctrlSMsg)

	assert.True(t, dialog.IsConfirmed())
}

func TestNewSkillDialog_EmptyPromptError(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateInput
	dialog.focusedField = 2 // Generate button

	// Mock tools available
	dialog.toolOptions = []skillgen.AITool{skillgen.AIToolClaude}

	// Press enter with empty prompt
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(enterMsg)

	assert.Equal(t, StateError, dialog.state)
	assert.NotNil(t, dialog.err)
}

func TestNewSkillDialog_NoToolsError(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateInput
	dialog.focusedField = 2
	dialog.toolOptions = []skillgen.AITool{} // No tools available

	// Set some prompt text
	dialog.promptInput.SetValue("test skill")

	// Press enter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(enterMsg)

	assert.Equal(t, StateError, dialog.state)
	assert.Contains(t, dialog.err.Error(), "no AI CLI tools")
}

func TestNewSkillDialog_CenteredView(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)

	view := dialog.CenteredView(100, 50)

	// View should be non-empty and larger than dialog alone
	assert.NotEmpty(t, view)
}

func TestNewSkillDialog_GetGeneratedContent(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.generatedContent = "test content"

	assert.Equal(t, "test content", dialog.GetGeneratedContent())
}

func TestNewSkillDialog_ShiftTabNavigation(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)

	// Start at prompt (0)
	assert.Equal(t, 0, dialog.focusedField)

	// Shift+Tab should go to generate button (2)
	shiftTabMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	dialog.Update(shiftTabMsg)
	assert.Equal(t, 2, dialog.focusedField)

	// Shift+Tab again should go to tool selector (1)
	dialog.Update(shiftTabMsg)
	assert.Equal(t, 1, dialog.focusedField)

	// Shift+Tab again should go to prompt (0)
	dialog.Update(shiftTabMsg)
	assert.Equal(t, 0, dialog.focusedField)
}

func TestNewSkillDialog_LeftArrowToolSelection(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)

	// Set up tools
	dialog.toolOptions = []skillgen.AITool{
		skillgen.AIToolClaude,
		skillgen.AIToolCodex,
		skillgen.AIToolOpenCode,
	}
	dialog.toolIndex = 1 // Start at codex
	dialog.selectedTool = skillgen.AIToolCodex
	dialog.focusedField = 1

	// Press left to go to claude
	leftMsg := tea.KeyMsg{Type: tea.KeyLeft}
	dialog.Update(leftMsg)

	assert.Equal(t, 0, dialog.toolIndex)
	assert.Equal(t, skillgen.AIToolClaude, dialog.selectedTool)
}

func TestNewSkillDialog_VimScrollKeys(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StatePreview

	// Create many lines
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = fmt.Sprintf("Line %d", i)
	}
	dialog.previewLines = lines

	// j should scroll down
	jMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	dialog.Update(jMsg)
	assert.Equal(t, 1, dialog.previewScroll)

	// k should scroll up
	kMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	dialog.Update(kMsg)
	assert.Equal(t, 0, dialog.previewScroll)
}

func TestNewSkillDialog_ErrorStateRecovery(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateError
	dialog.err = fmt.Errorf("test error")

	// Press Enter to recover
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(enterMsg)

	assert.Equal(t, StateInput, dialog.state)
	assert.Nil(t, dialog.err)
}

func TestNewSkillDialog_EscapeFromError(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateError
	dialog.err = fmt.Errorf("test error")

	// Press Escape to recover
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	dialog.Update(escMsg)

	assert.Equal(t, StateInput, dialog.state)
	assert.Nil(t, dialog.err)
}

func TestNewSkillDialog_BackFromPreview(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StatePreview
	dialog.generatedContent = "test"
	dialog.previewScroll = 5

	// Press Escape to go back to input
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	dialog.Update(escMsg)

	assert.Equal(t, StateInput, dialog.state)
	assert.Empty(t, dialog.generatedContent)
	assert.Equal(t, 0, dialog.previewScroll)
}

func TestNewSkillDialog_EnterConfirmsFromPreview(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StatePreview
	dialog.generatedContent = "test content"

	// Press Enter to confirm
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(enterMsg)

	assert.True(t, dialog.IsConfirmed())
}

func TestNewSkillDialog_StreamMsgHandling(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateGenerating

	// Simulate stream message
	streamMsg := NewSkillStreamMsg{
		Content: "test content",
		Done:    true,
	}
	dialog.Update(streamMsg)

	assert.Equal(t, StatePreview, dialog.state)
	assert.Equal(t, "test content", dialog.generatedContent)
}

func TestNewSkillDialog_StreamMsgError(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateGenerating

	// Simulate stream error
	streamMsg := NewSkillStreamMsg{
		Err:  fmt.Errorf("test error"),
		Done: true,
	}
	dialog.Update(streamMsg)

	assert.Equal(t, StateError, dialog.state)
	assert.NotNil(t, dialog.err)
}

func TestNewSkillDialog_TickMsgUpdatesAnimation(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateGenerating
	initialTick := dialog.animTick

	// Send tick message
	tickMsg := NewSkillTickMsg{}
	dialog.Update(tickMsg)

	assert.Equal(t, initialTick+1, dialog.animTick)
}

func TestNewSkillDialog_HKeyNavigatesToolLeft(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.toolOptions = []skillgen.AITool{
		skillgen.AIToolClaude,
		skillgen.AIToolCodex,
	}
	dialog.toolIndex = 1
	dialog.selectedTool = skillgen.AIToolCodex
	dialog.focusedField = 1

	// h should navigate left
	hMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	dialog.Update(hMsg)

	assert.Equal(t, 0, dialog.toolIndex)
	assert.Equal(t, skillgen.AIToolClaude, dialog.selectedTool)
}

func TestNewSkillDialog_LKeyNavigatesToolRight(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.toolOptions = []skillgen.AITool{
		skillgen.AIToolClaude,
		skillgen.AIToolCodex,
	}
	dialog.toolIndex = 0
	dialog.selectedTool = skillgen.AIToolClaude
	dialog.focusedField = 1

	// l should navigate right
	lMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	dialog.Update(lMsg)

	assert.Equal(t, 1, dialog.toolIndex)
	assert.Equal(t, skillgen.AIToolCodex, dialog.selectedTool)
}

func TestNewSkillDialog_ClaudeFinishedWithError(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.launching = true
	// Set skillsBefore to current state so no "new" skills are detected
	dialog.skillsBefore, _ = skillgen.ScanSkills()

	// Simulate Claude exiting with error and no skills found
	finishedMsg := NewSkillClaudeFinishedMsg{
		Err: fmt.Errorf("user cancelled"),
	}
	dialog.Update(finishedMsg)

	assert.Equal(t, StateError, dialog.state)
	assert.False(t, dialog.launching)
	assert.Contains(t, dialog.err.Error(), "session ended")
}

func TestNewSkillDialog_ClaudeFinishedNoSkills(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.launching = true
	// Set skillsBefore to current state so no "new" skills are detected
	dialog.skillsBefore, _ = skillgen.ScanSkills()

	// Simulate Claude exiting successfully but no new skills detected
	finishedMsg := NewSkillClaudeFinishedMsg{
		Err: nil,
	}
	dialog.Update(finishedMsg)

	// Should go to result state (not error) showing "no skills detected"
	assert.Equal(t, StateResult, dialog.state)
	assert.False(t, dialog.launching)
	assert.Empty(t, dialog.newSkills)
}

func TestNewSkillDialog_ResultStateHandling(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateResult

	// Press 'r' to try again
	rMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	dialog.Update(rMsg)

	assert.Equal(t, StateInput, dialog.state)
}

func TestNewSkillDialog_ResultStateClose(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 40)
	dialog.state = StateResult

	// Press Enter to close
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(enterMsg)

	assert.True(t, dialog.IsConfirmed())
}

func TestNewSkillDialog_NeedsSave(t *testing.T) {
	tests := []struct {
		name      string
		state     NewSkillDialogState
		confirmed bool
		wantSave  bool
	}{
		{"preview_confirmed", StatePreview, true, true},
		{"preview_not_confirmed", StatePreview, false, false},
		{"result_confirmed", StateResult, true, false},
		{"result_not_confirmed", StateResult, false, false},
		{"input_confirmed", StateInput, true, false},
		{"generating_confirmed", StateGenerating, true, false},
		{"error_confirmed", StateError, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialog := NewNewSkillDialog()
			dialog.state = tt.state
			dialog.confirmed = tt.confirmed

			assert.Equal(t, tt.wantSave, dialog.NeedsSave())
		})
	}
}

// Install state tests

func TestNewSkillDialogInstallState(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 24)

	// Set up platforms
	dialog.SetPlatforms([]installer.Platform{installer.PlatformClaude})

	// Simulate successful skill creation
	dialog.newSkills = []skillgen.SkillInfo{
		{Slug: "test-skill", Path: "/tmp/test-skill/skill.md"},
	}
	dialog.state = StateResult

	// Press enter to go to install state
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(enterMsg)

	assert.Equal(t, StateInstall, dialog.state)
	assert.NotNil(t, dialog.installSkillInfo)
	assert.Equal(t, "test-skill", dialog.installSkillInfo.Slug)
}

func TestNewSkillDialogInstallLater(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 24)
	dialog.SetPlatforms([]installer.Platform{installer.PlatformClaude})

	dialog.newSkills = []skillgen.SkillInfo{
		{Slug: "test-skill", Path: "/tmp/test-skill/skill.md"},
	}
	dialog.state = StateResult

	// Press L for install later
	lMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	dialog.Update(lMsg)

	assert.True(t, dialog.confirmed)
	assert.False(t, dialog.installSuccess)
}

func TestNewSkillDialogInstallConfirm(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 24)
	dialog.SetPlatforms([]installer.Platform{installer.PlatformClaude})

	dialog.newSkills = []skillgen.SkillInfo{
		{Slug: "test-skill", Path: "/tmp/test-skill/skill.md"},
	}
	dialog.state = StateResult

	// Go to install state
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(enterMsg)
	assert.Equal(t, StateInstall, dialog.state)

	// Confirm installation (Enter with default selection)
	dialog.Update(enterMsg)

	assert.True(t, dialog.confirmed)
	assert.True(t, dialog.installSuccess)
	assert.NotEmpty(t, dialog.GetInstallLocations())
}

func TestNewSkillDialogInstallEscGoesBack(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 24)
	dialog.SetPlatforms([]installer.Platform{installer.PlatformClaude})

	dialog.newSkills = []skillgen.SkillInfo{
		{Slug: "test-skill", Path: "/tmp/test-skill/skill.md"},
	}
	dialog.state = StateInstall
	dialog.installSkillInfo = &dialog.newSkills[0]

	// Press Esc to go back
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	dialog.Update(escMsg)

	assert.Equal(t, StateResult, dialog.state)
	assert.False(t, dialog.confirmed)
}

func TestNewSkillDialogWantsInstall(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 24)

	// Initially should not want install (no platforms, no install success)
	assert.False(t, dialog.WantsInstall())

	// After setting install success but no install dialog
	dialog.installSuccess = true
	assert.False(t, dialog.WantsInstall())

	// Now set platforms (this creates install dialog with default selections)
	dialog.SetPlatforms([]installer.Platform{installer.PlatformClaude})

	// Now with install success and installDialog, should want install
	assert.True(t, dialog.WantsInstall())

	// Reset install success
	dialog.installSuccess = false
	assert.False(t, dialog.WantsInstall())

	// Full flow: create skill, go to install, confirm
	dialog.newSkills = []skillgen.SkillInfo{
		{Slug: "test-skill", Path: "/tmp/test-skill/skill.md"},
	}
	dialog.state = StateResult

	// Go to install state and confirm
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	dialog.Update(enterMsg) // Go to install
	dialog.Update(enterMsg) // Confirm

	assert.True(t, dialog.WantsInstall())
	assert.NotNil(t, dialog.GetInstallSkillInfo())
}

func TestNewSkillDialogSetPlatforms(t *testing.T) {
	dialog := NewNewSkillDialog()

	// Initially no install dialog
	assert.Nil(t, dialog.installDialog)

	// Set platforms creates install dialog
	dialog.SetPlatforms([]installer.Platform{installer.PlatformClaude, installer.PlatformCursor})
	assert.NotNil(t, dialog.installDialog)

	// Empty platforms doesn't create dialog
	dialog2 := NewNewSkillDialog()
	dialog2.SetPlatforms([]installer.Platform{})
	assert.Nil(t, dialog2.installDialog)
}

func TestNewSkillDialogResetClearsInstallState(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 24)
	dialog.SetPlatforms([]installer.Platform{installer.PlatformClaude})

	// Set install state
	dialog.installSkillInfo = &skillgen.SkillInfo{Slug: "test"}
	dialog.installSuccess = true

	// Reset
	dialog.Reset()

	assert.Nil(t, dialog.installSkillInfo)
	assert.False(t, dialog.installSuccess)
}

func TestNewSkillDialogViewInstall(t *testing.T) {
	dialog := NewNewSkillDialog()
	dialog.SetSize(80, 24)
	dialog.SetPlatforms([]installer.Platform{installer.PlatformClaude})

	dialog.state = StateInstall
	dialog.installSkillInfo = &skillgen.SkillInfo{Slug: "my-skill"}

	view := dialog.View()

	assert.Contains(t, view, "Install")
	assert.Contains(t, view, "my-skill")
	assert.Contains(t, view, "Install Later")
}
