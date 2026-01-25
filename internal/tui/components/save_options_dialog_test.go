package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSaveOptionsDialog(t *testing.T) {
	dialog := NewSaveOptionsDialog()

	if dialog == nil {
		t.Fatal("NewSaveOptionsDialog should not return nil")
	}

	// Should have 3 options
	if len(dialog.options) != 3 {
		t.Errorf("expected 3 options, got %d", len(dialog.options))
	}

	// Default selection should be "Both" (index 0)
	if dialog.selectedIndex != 0 {
		t.Errorf("default selection should be 0, got %d", dialog.selectedIndex)
	}

	// Should not be cancelled or confirmed initially
	if dialog.IsCancelled() {
		t.Error("dialog should not be cancelled initially")
	}
	if dialog.IsConfirmed() {
		t.Error("dialog should not be confirmed initially")
	}

	// Default selection should be SaveToBoth
	if dialog.GetSelection() != SaveToBoth {
		t.Errorf("default selection should be SaveToBoth, got %v", dialog.GetSelection())
	}
}

func TestSaveOptionsDialog_KeyboardNavigation(t *testing.T) {
	tests := []struct {
		name              string
		keys              []tea.KeyMsg
		expectedIndex     int
		expectedConfirmed bool
		expectedCancelled bool
	}{
		{
			name:          "down arrow moves selection down",
			keys:          []tea.KeyMsg{{Type: tea.KeyDown}},
			expectedIndex: 1,
		},
		{
			name:          "up arrow moves selection up from first wraps to last",
			keys:          []tea.KeyMsg{{Type: tea.KeyUp}},
			expectedIndex: 2,
		},
		{
			name: "down arrow wraps around",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
			},
			expectedIndex: 0,
		},
		{
			name:              "enter confirms selection",
			keys:              []tea.KeyMsg{{Type: tea.KeyEnter}},
			expectedIndex:     0,
			expectedConfirmed: true,
		},
		{
			name:              "escape cancels dialog",
			keys:              []tea.KeyMsg{{Type: tea.KeyEsc}},
			expectedIndex:     0,
			expectedCancelled: true,
		},
		{
			name:          "tab moves selection down",
			keys:          []tea.KeyMsg{{Type: tea.KeyTab}},
			expectedIndex: 1,
		},
		{
			name:          "shift+tab moves selection up",
			keys:          []tea.KeyMsg{{Type: tea.KeyShiftTab}},
			expectedIndex: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dialog := NewSaveOptionsDialog()

			for _, key := range tc.keys {
				dialog.Update(key)
			}

			if dialog.selectedIndex != tc.expectedIndex {
				t.Errorf("expected selectedIndex %d, got %d", tc.expectedIndex, dialog.selectedIndex)
			}

			if dialog.IsConfirmed() != tc.expectedConfirmed {
				t.Errorf("expected confirmed=%v, got %v", tc.expectedConfirmed, dialog.IsConfirmed())
			}

			if dialog.IsCancelled() != tc.expectedCancelled {
				t.Errorf("expected cancelled=%v, got %v", tc.expectedCancelled, dialog.IsCancelled())
			}
		})
	}
}

func TestSaveOptionsDialog_VimNavigation(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		expectedIndex int
	}{
		{"j moves down", "j", 1},
		{"k moves up (wraps)", "k", 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dialog := NewSaveOptionsDialog()
			dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})

			if dialog.selectedIndex != tc.expectedIndex {
				t.Errorf("expected selectedIndex %d, got %d", tc.expectedIndex, dialog.selectedIndex)
			}
		})
	}
}

func TestSaveOptionsDialog_NumberShortcuts(t *testing.T) {
	tests := []struct {
		name              string
		key               string
		expectedSelection SaveDestination
		expectedConfirmed bool
	}{
		{"1 selects Both and confirms", "1", SaveToBoth, true},
		{"2 selects Database and confirms", "2", SaveToDatabase, true},
		{"3 selects Files and confirms", "3", SaveToFiles, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dialog := NewSaveOptionsDialog()
			dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})

			if dialog.GetSelection() != tc.expectedSelection {
				t.Errorf("expected selection %v, got %v", tc.expectedSelection, dialog.GetSelection())
			}

			if dialog.IsConfirmed() != tc.expectedConfirmed {
				t.Errorf("expected confirmed=%v, got %v", tc.expectedConfirmed, dialog.IsConfirmed())
			}
		})
	}
}

func TestSaveOptionsDialog_HandleKey(t *testing.T) {
	tests := []struct {
		name              string
		keys              []string
		expectedIndex     int
		expectedConfirmed bool
		expectedCancelled bool
	}{
		{"up moves up", []string{"up"}, 2, false, false},
		{"down moves down", []string{"down"}, 1, false, false},
		{"enter confirms", []string{"enter"}, 0, true, false},
		{"esc cancels", []string{"esc"}, 0, false, true},
		{"j moves down", []string{"j"}, 1, false, false},
		{"k moves up", []string{"k"}, 2, false, false},
		{"1 selects first and confirms", []string{"1"}, 0, true, false},
		{"2 selects second and confirms", []string{"2"}, 1, true, false},
		{"3 selects third and confirms", []string{"3"}, 2, true, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dialog := NewSaveOptionsDialog()

			for _, key := range tc.keys {
				dialog.HandleKey(key)
			}

			if dialog.selectedIndex != tc.expectedIndex {
				t.Errorf("expected selectedIndex %d, got %d", tc.expectedIndex, dialog.selectedIndex)
			}

			if dialog.IsConfirmed() != tc.expectedConfirmed {
				t.Errorf("expected confirmed=%v, got %v", tc.expectedConfirmed, dialog.IsConfirmed())
			}

			if dialog.IsCancelled() != tc.expectedCancelled {
				t.Errorf("expected cancelled=%v, got %v", tc.expectedCancelled, dialog.IsCancelled())
			}
		})
	}
}

func TestSaveOptionsDialog_Reset(t *testing.T) {
	dialog := NewSaveOptionsDialog()

	// Navigate and confirm
	dialog.HandleKey("down")
	dialog.HandleKey("enter")

	if !dialog.IsConfirmed() {
		t.Error("dialog should be confirmed before reset")
	}

	// Reset
	dialog.Reset()

	if dialog.selectedIndex != 0 {
		t.Errorf("selectedIndex should be 0 after reset, got %d", dialog.selectedIndex)
	}
	if dialog.IsConfirmed() {
		t.Error("dialog should not be confirmed after reset")
	}
	if dialog.IsCancelled() {
		t.Error("dialog should not be cancelled after reset")
	}
}

func TestSaveOptionsDialog_GetSelection(t *testing.T) {
	dialog := NewSaveOptionsDialog()

	// Default is SaveToBoth (index 0)
	if dialog.GetSelection() != SaveToBoth {
		t.Errorf("expected SaveToBoth, got %v", dialog.GetSelection())
	}

	// Move to SaveToDatabase (index 1)
	dialog.HandleKey("down")
	if dialog.GetSelection() != SaveToDatabase {
		t.Errorf("expected SaveToDatabase, got %v", dialog.GetSelection())
	}

	// Move to SaveToFiles (index 2)
	dialog.HandleKey("down")
	if dialog.GetSelection() != SaveToFiles {
		t.Errorf("expected SaveToFiles, got %v", dialog.GetSelection())
	}
}

func TestSaveOptionsDialog_View(t *testing.T) {
	dialog := NewSaveOptionsDialog()
	dialog.SetWidth(60)

	view := dialog.View()

	// Check that key elements are present
	expectedElements := []string{
		"Save Skill",                   // Title
		"Where would you like to save", // Subtitle
		"Both",                         // Option 1
		"Database Only",                // Option 2
		"Local Files Only",             // Option 3
		"recommended",                  // Badge for Both
		"confirm",                      // Help text
		"cancel",                       // Help text
	}

	for _, elem := range expectedElements {
		if !strings.Contains(view, elem) {
			t.Errorf("view should contain %q", elem)
		}
	}
}

func TestSaveOptionsDialog_CenteredView(t *testing.T) {
	dialog := NewSaveOptionsDialog()
	dialog.SetWidth(60)

	view := dialog.CenteredView(100, 50)

	// Should contain the dialog content
	if !strings.Contains(view, "Save Skill") {
		t.Error("centered view should contain dialog content")
	}
}

func TestSaveOptionsDialog_OverlayView(t *testing.T) {
	dialog := NewSaveOptionsDialog()
	dialog.SetWidth(60)

	background := strings.Repeat("Background content\n", 20)
	view := dialog.OverlayView(background, 100, 30)

	// Should contain the dialog content
	if !strings.Contains(view, "Save Skill") {
		t.Error("overlay view should contain dialog content")
	}
}

func TestSaveDestination_Values(t *testing.T) {
	// Verify the enum values (iota order: Database=0, Files=1, Both=2)
	if SaveToDatabase != 0 {
		t.Errorf("SaveToDatabase should be 0, got %d", SaveToDatabase)
	}
	if SaveToFiles != 1 {
		t.Errorf("SaveToFiles should be 1, got %d", SaveToFiles)
	}
	if SaveToBoth != 2 {
		t.Errorf("SaveToBoth should be 2, got %d", SaveToBoth)
	}

	// Verify that options array has correct mapping
	dialog := NewSaveOptionsDialog()
	// options[0] = SaveToBoth (recommended, first in UI)
	// options[1] = SaveToDatabase
	// options[2] = SaveToFiles
	if dialog.options[0].Destination != SaveToBoth {
		t.Errorf("first option should be SaveToBoth")
	}
	if dialog.options[1].Destination != SaveToDatabase {
		t.Errorf("second option should be SaveToDatabase")
	}
	if dialog.options[2].Destination != SaveToFiles {
		t.Errorf("third option should be SaveToFiles")
	}
}
