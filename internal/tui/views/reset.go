package views

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GetDB returns the current database reference (used by app.go to close it before reset).
func (v *ResetView) GetDB() *db.DB {
	return v.db
}

// ResetCompleteMsg is sent when the async reset operation completes.
// NewDB contains the fresh database connection if successful.
type ResetCompleteMsg struct {
	Success bool
	Err     error
	NewDB   *db.DB // Fresh database connection after reset
}

// ResetView displays the database reset confirmation.
type ResetView struct {
	db        *db.DB
	cfg       *config.Config
	selected  bool   // false = cancel, true = reset
	confirmed bool   // whether user confirmed
	resetting bool   // true while async reset is in progress
	error     string // error message if any
	width     int
	height    int
}

// NewResetView creates a new reset view.
func NewResetView(database *db.DB, conf *config.Config) *ResetView {
	return &ResetView{
		db:       database,
		cfg:      conf,
		selected: false, // Default to "Cancel"
	}
}

// Init initializes the view.
func (v *ResetView) Init() {
	v.selected = false
	v.confirmed = false
	v.resetting = false
	v.error = ""
}

// SetSize sets the width and height of the view.
func (v *ResetView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Update handles key input. Returns (true if should go back, was reset successful, tea.Cmd).
func (v *ResetView) Update(key string) (bool, bool, tea.Cmd) {
	// While resetting is in progress, ignore all keys
	if v.resetting {
		return false, false, nil
	}

	if v.confirmed {
		// Wait for user to acknowledge result
		switch key {
		case "enter", "esc":
			return true, v.error == "", nil
		}
		return false, false, nil
	}

	switch key {
	case "left", "h":
		v.selected = false
	case "right", "l":
		v.selected = true
	case "enter":
		if v.selected {
			// Reset selected - start async reset
			v.resetting = true
			return false, false, v.startResetCmd()
		}
		// Cancel selected - go back immediately
		return true, false, nil
	case "esc":
		return true, false, nil
	}

	return false, false, nil
}

// startResetCmd returns a tea.Cmd that performs the full reset asynchronously.
// This includes deleting files AND recreating the database.
func (v *ResetView) startResetCmd() tea.Cmd {
	// IMPORTANT: Query installations SYNCHRONOUSLY before returning the async cmd.
	// The database will be closed by app.go before the async cmd runs,
	// so we must capture installation data now while the DB is still open.
	var installations []models.SkillInstallation
	if v.db != nil {
		installations, _ = v.db.GetAllInstallations()
	}

	return func() tea.Msg {
		// Step 1: Delete all cached data (using pre-captured installations)
		if err := v.performResetAsync(installations); err != nil {
			return ResetCompleteMsg{Success: false, Err: err, NewDB: nil}
		}

		// Step 2: Recreate the database
		paths := config.GetPaths(v.cfg)
		newDB, err := db.New(db.DefaultConfig(paths.Database))
		if err != nil {
			return ResetCompleteMsg{Success: false, Err: fmt.Errorf("failed to recreate database: %w", err), NewDB: nil}
		}

		// Step 3: Reset onboarding state
		if err := newDB.ResetOnboarding(); err != nil {
			return ResetCompleteMsg{Success: false, Err: fmt.Errorf("failed to reset onboarding: %w", err), NewDB: newDB}
		}

		return ResetCompleteMsg{Success: true, Err: nil, NewDB: newDB}
	}
}

// HandleResetComplete handles the completion of the async reset operation.
func (v *ResetView) HandleResetComplete(msg ResetCompleteMsg) {
	v.resetting = false
	v.confirmed = true
	if msg.Err != nil {
		v.error = msg.Err.Error()
	}
}

// performResetAsync deletes all cached data for a clean slate.
// This includes: symlinks, database, repositories, vectors, embeddings, and logs.
// User skills in ~/.skulto/skills are preserved.
// This runs asynchronously in a goroutine to avoid blocking the TUI.
// installations is passed in because the database is closed before this runs.
func (v *ResetView) performResetAsync(installations []models.SkillInstallation) error {
	paths := config.GetPaths(v.cfg)
	var errors []string

	// Step 1: Remove all skill symlinks using pre-captured installations
	for _, inst := range installations {
		if inst.SymlinkPath != "" {
			if err := os.Remove(inst.SymlinkPath); err != nil && !os.IsNotExist(err) {
				// Log but continue - symlink might already be gone
				errors = append(errors, fmt.Sprintf("Failed to remove symlink %s: %v", inst.SymlinkPath, err))
			}
		}
	}

	// Step 2: Also scan known global locations as backup (in case DB is incomplete)
	// This catches any symlinks not tracked in the database
	v.removeSymlinksFromGlobalLocations(&errors)

	// Items to delete (files)
	filesToDelete := []string{
		paths.Database,   // skulto.db
		paths.Embeddings, // embeddings.db
		filepath.Join(v.cfg.BaseDir, "skulto.log"), // log file
	}

	// Items to delete (directories)
	dirsToDelete := []string{
		paths.Repositories,                      // repositories/
		filepath.Join(v.cfg.BaseDir, "vectors"), // vectors/
	}

	// Delete files
	for _, path := range filesToDelete {
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to delete %s: %v", path, err))
			}
		}
	}

	// Delete directories recursively
	for _, path := range dirsToDelete {
		if _, err := os.Stat(path); err == nil {
			if err := os.RemoveAll(path); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to delete %s: %v", path, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", errors[0])
	}

	return nil
}

// removeSymlinksFromAllLocations removes all skill symlinks from all AI tool directories.
// This scans both global (home) and project (CWD) scopes for all platforms.
// This is a backup in case the database doesn't have complete installation records.
func (v *ResetView) removeSymlinksFromGlobalLocations(errors *[]string) {
	// Get base paths for both scopes
	homeDir, homeErr := os.UserHomeDir()
	cwdDir, cwdErr := os.Getwd()

	basePaths := make([]string, 0, 2)
	if homeErr == nil {
		basePaths = append(basePaths, homeDir)
	}
	if cwdErr == nil && cwdDir != homeDir {
		basePaths = append(basePaths, cwdDir)
	}

	// Scan all platforms in both scopes
	for _, platform := range installer.AllPlatforms() {
		info := platform.Info()
		if info.SkillsPath == "" {
			continue
		}

		for _, basePath := range basePaths {
			skillsDir := filepath.Join(basePath, info.SkillsPath)
			if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
				continue
			}

			// Read all entries in the skills directory
			entries, err := os.ReadDir(skillsDir)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				entryPath := filepath.Join(skillsDir, entry.Name())

				// Check if it's a symlink
				fileInfo, err := os.Lstat(entryPath)
				if err != nil {
					continue
				}

				if fileInfo.Mode()&os.ModeSymlink != 0 {
					if err := os.Remove(entryPath); err != nil && !os.IsNotExist(err) {
						*errors = append(*errors, fmt.Sprintf("Failed to remove symlink %s: %v", entryPath, err))
					}
				}
			}
		}
	}
}

// View renders the reset confirmation view.
func (v *ResetView) View() string {
	if v.resetting {
		return v.renderResettingView()
	}
	if v.confirmed {
		return v.renderResult()
	}

	return v.renderConfirmation()
}

// renderResettingView renders the "Resetting..." progress view.
func (v *ResetView) renderResettingView() string {
	// Calculate responsive max width
	maxWidth := v.width
	if maxWidth > 60 {
		maxWidth = 60
	}
	if maxWidth < 40 {
		maxWidth = 40
	}

	progressBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F1C40F")).
		Padding(2, 3).
		MaxWidth(maxWidth).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#F1C40F")).
					Render("Resetting..."),
				"",
				"Please wait while Skulto is being reset.",
				"This may take a moment.",
			),
		)

	dialogWidth := lipgloss.Width(progressBox)
	paddingLeft := (v.width - dialogWidth) / 2
	if paddingLeft < 0 {
		paddingLeft = 0
	}
	paddingTop := (v.height - 10) / 2
	if paddingTop < 1 {
		paddingTop = 1
	}
	return lipgloss.NewStyle().
		Padding(paddingTop, paddingLeft).
		Render(progressBox)
}

// renderConfirmation renders the confirmation prompt.
func (v *ResetView) renderConfirmation() string {
	// Button styles
	cancelStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(lipgloss.Color("240"))

	resetStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(lipgloss.Color("240"))

	if v.selected {
		resetStyle = resetStyle.
			Background(lipgloss.Color("196")).
			Foreground(lipgloss.Color("255")).
			Bold(true)
	} else {
		cancelStyle = cancelStyle.
			Background(lipgloss.Color("220")).
			Foreground(lipgloss.Color("0")).
			Bold(true)
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Left,
		"[ ",
		cancelStyle.Render("Cancel"),
		" ] [ ",
		resetStyle.Render("Reset"),
		" ]",
	)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Render("⚠️  Reset Database")

	message := "This will permanently delete:\n• Database and cached skills\n• Cloned repositories\n• Vector embeddings\n\nYour local skills will be preserved.\nYou will need to go through onboarding again."

	// Calculate responsive max width (80% of available width, but not more than 60)
	maxWidth := v.width
	if maxWidth > 60 {
		maxWidth = 60
	}
	if maxWidth < 40 {
		maxWidth = 40
	}

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(2, 3).
		MaxWidth(maxWidth).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				title,
				"",
				message,
				"",
				buttons,
			),
		)

	// Center the dialog with proper spacing
	dialogWidth := lipgloss.Width(dialog)
	paddingLeft := (v.width - dialogWidth) / 2
	if paddingLeft < 0 {
		paddingLeft = 0
	}

	paddingTop := (v.height - 12) / 2
	if paddingTop < 1 {
		paddingTop = 1
	}

	return lipgloss.NewStyle().
		Padding(paddingTop, paddingLeft).
		Render(dialog)
}

// renderResult renders the result of the reset operation.
func (v *ResetView) renderResult() string {
	// Calculate responsive max width (80% of available width, but not more than 60)
	maxWidth := v.width
	if maxWidth > 60 {
		maxWidth = 60
	}
	if maxWidth < 40 {
		maxWidth = 40
	}

	if v.error != "" {
		resultBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(2, 3).
			MaxWidth(maxWidth).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Center,
					lipgloss.NewStyle().
						Bold(true).
						Foreground(lipgloss.Color("196")).
						Render("Reset Failed"),
					"",
					v.error,
					"",
					"Press enter to continue",
				),
			)

		dialogWidth := lipgloss.Width(resultBox)
		paddingLeft := (v.width - dialogWidth) / 2
		if paddingLeft < 0 {
			paddingLeft = 0
		}
		paddingTop := (v.height - 12) / 2
		if paddingTop < 1 {
			paddingTop = 1
		}
		return lipgloss.NewStyle().
			Padding(paddingTop, paddingLeft).
			Render(resultBox)
	}

	resultBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("46")).
		Padding(2, 3).
		MaxWidth(maxWidth).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("46")).
					Render("✅ Reset"),
				"",
				"Skulto has been reset successfully.",
				"",
				"Press enter to continue",
			),
		)

	dialogWidth := lipgloss.Width(resultBox)
	paddingLeft := (v.width - dialogWidth) / 2
	if paddingLeft < 0 {
		paddingLeft = 0
	}
	paddingTop := (v.height - 12) / 2
	if paddingTop < 1 {
		paddingTop = 1
	}
	return lipgloss.NewStyle().
		Padding(paddingTop, paddingLeft).
		Render(resultBox)
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (v *ResetView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Reset",
		Commands: []Command{
			{Key: "←→, h/l", Description: "Select Cancel or Reset button"},
			{Key: "Enter", Description: "Confirm selection"},
			{Key: "Esc", Description: "Cancel reset and go back"},
		},
	}
}
