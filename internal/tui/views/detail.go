package views

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/log"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// SkillLoadedMsg is sent when async skill loading completes.
// This message is handled by app.go to update the detail view.
type SkillLoadedMsg struct {
	Skill *models.Skill
	Err   error
}

// SkillInstalledMsg is sent when async skill installation completes.
type SkillInstalledMsg struct {
	Success bool
	Err     error
}

// SkillScanRequestMsg requests a scan of a specific skill.
type SkillScanRequestMsg struct {
	SkillID string
}

// SkillScanCompleteMsg signals that a skill scan has finished.
type SkillScanCompleteMsg struct {
	SkillID string
	Err     error
}

// DetailView displays detailed information about a selected skill.
type DetailView struct {
	db        *db.DB
	cfg       *config.Config
	telemetry telemetry.Client

	skill     *models.Skill
	skillID   string
	loading   bool
	loadError error

	// Installation state
	installing bool
	installErr error

	// Scanning state
	scanning bool

	// Scrolling state
	scrollOffset int
	maxScroll    int

	// Dimensions
	width  int
	height int

	// Content cache
	renderedContent []string

	// Cached markdown renderer (expensive to create)
	glamourRenderer *glamour.TermRenderer
}

// NewDetailView creates a new DetailView.
func NewDetailView(database *db.DB, conf *config.Config) *DetailView {
	// Pre-create the glamour renderer (expensive operation)
	// This is done once at startup rather than on each render
	// Use dark theme for better visibility in terminal
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
		glamour.WithEmoji(),
	)

	return &DetailView{
		db:              database,
		cfg:             conf,
		glamourRenderer: renderer,
	}
}

// Init initializes the DetailView and sets the telemetry client.
func (dv *DetailView) Init(tc telemetry.Client) {
	dv.telemetry = tc
	dv.skill = nil
	dv.skillID = ""
	dv.loading = false
	dv.loadError = nil
	dv.scrollOffset = 0
	dv.maxScroll = 0
	dv.renderedContent = []string{}
}

// SetSkill initiates async loading of a skill by ID.
// Returns a tea.Cmd that will send SkillLoadedMsg when complete.
// The loading indicator will be shown immediately while the skill loads.
func (dv *DetailView) SetSkill(skillID string) tea.Cmd {
	dv.skillID = skillID
	dv.loading = true
	dv.loadError = nil
	dv.scrollOffset = 0
	dv.skill = nil
	dv.renderedContent = nil

	// Return a command that loads the skill asynchronously
	return dv.loadSkillCmd(skillID)
}

// loadSkillCmd returns a command that loads a skill from the database.
// This runs in a goroutine, allowing the UI to remain responsive.
func (dv *DetailView) loadSkillCmd(skillID string) tea.Cmd {
	return func() tea.Msg {
		skill, err := dv.db.GetSkill(skillID)
		return SkillLoadedMsg{Skill: skill, Err: err}
	}
}

// HandleSkillLoaded processes the result of async skill loading.
// This should be called by app.go when SkillLoadedMsg is received.
func (dv *DetailView) HandleSkillLoaded(msg SkillLoadedMsg) {
	dv.loading = false

	if msg.Err != nil {
		dv.loadError = fmt.Errorf("failed to load skill: %w", msg.Err)
		return
	}

	if msg.Skill == nil {
		dv.loadError = fmt.Errorf("skill not found")
		return
	}

	dv.skill = msg.Skill

	// Record that this skill was viewed
	if err := dv.db.RecordSkillView(msg.Skill.ID); err != nil {
		log.Printf("failed to record skill view: %v", err)
	}

	// Track skill preview
	dv.telemetry.TrackSkillPreviewed(dv.skill.Title, dv.skill.Category, 0)

	dv.updateRenderedContent()
}

// updateRenderedContent renders the skill content for display using Glamour markdown renderer.
func (dv *DetailView) updateRenderedContent() {
	if dv.skill == nil {
		dv.renderedContent = []string{}
		dv.maxScroll = 0
		return
	}

	// Render markdown using Glamour - handles all styling including code blocks
	contentLines := dv.renderMarkdownContent(dv.skill.Content)
	dv.renderedContent = contentLines

	// Calculate max scroll
	metadataHeight := dv.estimateMetadataHeight()
	viewportHeight := dv.height - metadataHeight - 2 // 2 for footer/indicators
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	totalLines := len(dv.renderedContent)
	dv.maxScroll = max(0, totalLines-viewportHeight)
}

// renderMarkdownContent renders markdown content using the cached Glamour renderer.
func (dv *DetailView) renderMarkdownContent(content string) []string {
	if content == "" {
		return []string{}
	}

	// Strip frontmatter if present (YAML between --- delimiters at start)
	contentWithoutFrontmatter := dv.stripFrontmatter(content)

	// Use cached renderer if available, otherwise fallback to plain text
	if dv.glamourRenderer == nil {
		return strings.Split(contentWithoutFrontmatter, "\n")
	}

	// Render the markdown using cached renderer
	rendered, err := dv.glamourRenderer.Render(contentWithoutFrontmatter)
	if err != nil {
		// Fallback to plain text if rendering fails
		return strings.Split(contentWithoutFrontmatter, "\n")
	}

	// Split into lines for scrolling
	lines := strings.Split(rendered, "\n")

	// Remove trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// stripFrontmatter removes YAML frontmatter from markdown content.
// Frontmatter is expected to be at the start of the content, delimited by --- on separate lines.
func (dv *DetailView) stripFrontmatter(content string) string {
	lines := strings.Split(content, "\n")

	// Check if content starts with frontmatter delimiter
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return content
	}

	// Find the closing delimiter
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			// Return content after the closing delimiter, skipping the empty line after it
			remaining := strings.Join(lines[i+1:], "\n")
			return strings.TrimPrefix(remaining, "\n")
		}
	}

	// If no closing delimiter found, return original content
	return content
}

// estimateMetadataHeight estimates how many lines the metadata section takes.
func (dv *DetailView) estimateMetadataHeight() int {
	// Metadata section:
	// Title: 2 lines
	// Author/Category/Difficulty: 1 line
	// Stats: 1 line
	// Source/FilePath: 1 line
	// Version/License: 1 line
	// License URL: 1 line (if available)
	// Tags: variable, assume 1-2 lines
	// Timestamps: 1 line
	// Installed: 1 line
	// Divider: 1 line
	// Subtotal: 11 lines

	// Frontmatter section:
	// Blank line before: 1 line
	// "---": 1 line
	// Fields (name, description, version, author, license, tags, source): ~6 lines
	// "---": 1 line
	// Blank line after: 1 line
	// Subtotal: ~9 lines

	return 12
}

// Update handles keyboard input and returns (shouldGoBack, cmd).
func (dv *DetailView) Update(key string) (back bool, cmd tea.Cmd) {
	if dv.skill == nil && dv.loadError == nil {
		return false, nil
	}

	switch key {
	case "up", "k":
		dv.scrollOffset = max(0, dv.scrollOffset-1)
	case "down", "j":
		dv.scrollOffset = min(dv.maxScroll, dv.scrollOffset+1)
	case "t":
		// Go to top
		dv.scrollOffset = 0
	case "b":
		// Go to bottom
		dv.scrollOffset = dv.maxScroll
	case "i":
		// Toggle install - set installing state and return command to perform installation
		// Works for both local and remote skills
		if dv.skill != nil && !dv.installing {
			dv.skill.IsInstalled = !dv.skill.IsInstalled
			dv.installing = true
			dv.installErr = nil
			return false, nil // Command will be handled by app.go
		}
	case "c":
		// Copy to clipboard
		if dv.skill != nil {
			_ = clipboard.WriteAll(dv.skill.Content)
			dv.telemetry.TrackSkillCopied(dv.skill.Title)
		}
	case "S":
		// Trigger skill scan
		if dv.skill != nil {
			return false, dv.scanSkillCmd()
		}
	case "esc":
		return true, nil
	}

	return false, nil
}

// scanSkillCmd returns a command to scan the current skill.
func (dv *DetailView) scanSkillCmd() tea.Cmd {
	return func() tea.Msg {
		return SkillScanRequestMsg{SkillID: dv.skillID}
	}
}

// SetInstallingState sets the installing state and returns a command to perform the installation.
func (dv *DetailView) SetInstallingState(isInstalling bool) {
	dv.installing = isInstalling
	if !isInstalling {
		dv.installErr = nil
	}
}

// SetInstallError sets the install error state.
func (dv *DetailView) SetInstallError(err error) {
	dv.installErr = err
	dv.installing = false
}

// IsInstalling returns whether an installation is in progress.
func (dv *DetailView) IsInstalling() bool {
	return dv.installing
}

// SetScanning sets the scanning state.
func (dv *DetailView) SetScanning(scanning bool) {
	dv.scanning = scanning
}

// IsScanning returns whether a scan is in progress.
func (dv *DetailView) IsScanning() bool {
	return dv.scanning
}

// View renders the detailed skill view.
func (dv *DetailView) View() string {
	if dv.loading {
		return dv.renderLoading()
	}

	if dv.loadError != nil {
		return dv.renderError()
	}

	if dv.skill == nil {
		return dv.renderError()
	}

	// Build warning banner - FIXED at top, not scrollable
	// Only show for skills with actual threats
	var warningBanner string
	if dv.skill.ThreatLevel != models.ThreatLevelNone && dv.skill.ThreatLevel != "" {
		warningBanner = dv.renderWarningBanner()
	}

	// Render metadata and content sections (scrollable)
	var metadata string
	if dv.skill.IsLocal {
		metadata = dv.renderLocalDetails()
	} else {
		metadata = dv.renderRemoteDetails()
	}
	content := dv.renderContent()
	scrollIndicator := dv.renderScrollIndicator()

	// Build scrollable area
	scrollableContent := strings.Join([]string{metadata, content}, "\n")

	// Build final output: fixed banner at top, then scrollable content, then footer
	var parts []string
	if warningBanner != "" {
		parts = append(parts, warningBanner)
	}
	parts = append(parts, scrollableContent, "", scrollIndicator)

	return strings.Join(parts, "\n")
}

// renderWarningBanner renders the security warning banner.
func (dv *DetailView) renderWarningBanner() string {
	if dv.skill == nil || dv.skill.ThreatLevel == models.ThreatLevelNone {
		return ""
	}

	// Determine color based on threat level
	var bgColor, fgColor string
	switch dv.skill.ThreatLevel {
	case models.ThreatLevelCritical:
		bgColor = "#8B0000" // Dark red
		fgColor = "#FFFFFF"
	case models.ThreatLevelHigh:
		bgColor = "#FF4500" // Orange red
		fgColor = "#FFFFFF"
	case models.ThreatLevelMedium:
		bgColor = "#FF8C00" // Dark orange
		fgColor = "#000000"
	case models.ThreatLevelLow:
		bgColor = "#FFD700" // Gold
		fgColor = "#000000"
	default:
		// DEBUG: Show cyan banner for skills with no threat level
		bgColor = "#00CED1" // Dark cyan - debug color
		fgColor = "#000000"
	}

	bannerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(bgColor)).
		Foreground(lipgloss.Color(fgColor)).
		Bold(true).
		Padding(0, 1).
		Width(dv.width)

	// Build warning text
	warningText := fmt.Sprintf("WARNING: May contain risky patterns [%s]", dv.skill.ThreatLevel)
	if dv.skill.ThreatSummary != "" {
		// Show first part of summary (highest pattern)
		summary := dv.skill.ThreatSummary
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		warningText += " - " + summary
	}

	return bannerStyle.Render(warningText)
}

// SetSize updates the dimensions of the view.
func (dv *DetailView) SetSize(w, h int) {
	dv.width = w
	dv.height = h
	if dv.skill != nil {
		dv.updateRenderedContent()
	}
}

// Skill returns the currently displayed skill.
func (dv *DetailView) Skill() *models.Skill {
	return dv.skill
}

// renderTitle renders the skill title with consistent styling.
func (dv *DetailView) renderTitle() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DC143C")).
		Bold(true).
		Render(dv.skill.Title)
}

// renderDescription renders the skill description with fallback to summary.
func (dv *DetailView) renderDescription() string {
	desc := dv.skill.Description
	if desc == "" {
		desc = dv.skill.Summary
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E5E5")).
		Padding(0, 0, 1, 0).
		Width(dv.width).
		Render(desc)
}

// renderInstallIndicator renders the install/uninstall status indicator.
func (dv *DetailView) renderInstallIndicator() string {
	installText := "☐ Install to AI tools (i)"
	if dv.skill.IsInstalled {
		installText = "☑ Installed (i to uninstall)"
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F1C40F")).
		Bold(true).
		Render(installText)
}

// renderDivider renders a horizontal divider line.
func (dv *DetailView) renderDivider() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B6B6B")).
		Render(strings.Repeat("─", dv.width))
}

// renderMetadataRow renders a row of metadata text in gray.
func (dv *DetailView) renderMetadataRow(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B6B6B")).
		Render(text)
}

// renderLocalDetails renders a simplified metadata section for local skills.
func (dv *DetailView) renderLocalDetails() string {
	if dv.skill == nil {
		return ""
	}

	// Local skill badge
	badge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true).
		Render("Local Skill")

	// Folder path (parent directory of skill file)
	folderPath := filepath.Dir(dv.skill.FilePath)
	folderStr := dv.renderMetadataRow(fmt.Sprintf("📁 %s", folderPath))

	result := []string{
		badge,
		folderStr,
		dv.renderDivider(),
		dv.renderTitle(),
		dv.renderDescription(),
	}

	if tagsStr := dv.renderTags(); tagsStr != "" {
		result = append(result, tagsStr)
	}

	result = append(result,
		"",
		dv.renderInstallIndicator(),
		dv.renderDivider(),
	)

	return strings.Join(result, "\n")
}

// renderRemoteDetails renders the skill metadata section for remote skills.
func (dv *DetailView) renderRemoteDetails() string {
	if dv.skill == nil {
		return ""
	}

	// First row: Source Owner, Category, and Stats
	author := "Unknown"
	if dv.skill.Source != nil && dv.skill.Source.Owner != "" {
		author = dv.skill.Source.Owner
	}
	category := dv.skill.Category
	if category == "" {
		category = "General"
	}
	row1 := dv.renderMetadataRow(fmt.Sprintf("👤 %s  •  📁 %s  •  ⭐ %d  •  🍴 %d",
		author, category, dv.skill.Stars, dv.skill.Forks))

	// Source info row
	sourceInfo := "Unknown source"
	if dv.skill.Source != nil {
		sourceInfo = fmt.Sprintf("%s/%s", dv.skill.Source.Owner, dv.skill.Source.Repo)
	}
	filePath := dv.skill.FilePath
	if filePath == "" {
		filePath = "Unknown"
	} else {
		filePath = FormatPath(filePath)
	}
	row3 := dv.renderMetadataRow(fmt.Sprintf("📦 Source: %s | %s", sourceInfo, filePath))

	// Commit info row
	commitSHA := "Unknown"
	commitURL := ""
	licenseType := ""
	licenseURL := ""

	if dv.skill.Source != nil {
		if dv.skill.Source.LastCommitSHA != "" {
			commitSHA = dv.skill.Source.LastCommitSHA
			if len(commitSHA) > 7 {
				commitSHA = commitSHA[:7]
			}
			commitURL = fmt.Sprintf("https://github.com/%s/%s/commit/%s",
				dv.skill.Source.Owner, dv.skill.Source.Repo, dv.skill.Source.LastCommitSHA)
		}
		if dv.skill.Source.LicenseType != "" {
			licenseType = dv.skill.Source.LicenseType
		}
		licenseURL = dv.skill.Source.LicenseURL
	} else if dv.skill.SourceID != nil {
		commitSHA = *dv.skill.SourceID + " (source not preloaded)"
	}
	row5 := dv.renderMetadataRow(fmt.Sprintf("📌 Commit: %s | %s", commitSHA, commitURL))

	// Timestamps row
	indexedAt := "Unknown"
	if !dv.skill.IndexedAt.IsZero() {
		indexedAt = formatTime(dv.skill.IndexedAt)
	}
	lastSyncAt := "Never"
	if dv.skill.LastSyncAt != nil && !dv.skill.LastSyncAt.IsZero() {
		lastSyncAt = formatTime(*dv.skill.LastSyncAt)
	}
	row6 := dv.renderMetadataRow(fmt.Sprintf("🔍 Indexed: %s  •  🔄 Last Sync: %s", indexedAt, lastSyncAt))

	// Build result
	result := []string{
		row1,
		row3,
	}

	// Only add license row if license info is available
	if licenseType != "" || licenseURL != "" {
		result = append(result, dv.renderMetadataRow(fmt.Sprintf("⚖️  License: %s | %s", licenseType, licenseURL)))
	}

	result = append(result,
		row5,
		dv.renderDivider(),
		dv.renderTitle(),
		dv.renderDescription(),
	)

	if tagsStr := dv.renderTags(); tagsStr != "" {
		result = append(result, tagsStr)
	}

	result = append(result,
		row6,
		"",
		dv.renderInstallIndicator(),
	)

	// Add security warning banner only for skills with threats
	if dv.skill.ThreatLevel != models.ThreatLevelNone && dv.skill.ThreatLevel != "" {
		securityBanner := dv.renderWarningBanner()
		if securityBanner != "" {
			result = append(result, securityBanner)
		}
	}

	result = append(result, dv.renderDivider())

	return strings.Join(result, "\n")
}

// renderTags renders the skill tags with automatic wrapping based on screen width.
func (dv *DetailView) renderTags() string {
	if dv.skill == nil || len(dv.skill.Tags) == 0 {
		return ""
	}

	var tagStrs []string
	for _, tag := range dv.skill.Tags {
		color := getTagColor(tag.Category)
		tagStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(color).
			Padding(0, 1)
		tagStrs = append(tagStrs, tagStyle.Render(tag.Name))
	}

	result := strings.Join(tagStrs, " ")
	return lipgloss.NewStyle().
		PaddingBottom(1).
		Width(dv.width).
		Render("Tags: " + result)
}

// renderContent renders the scrollable skill content.
func (dv *DetailView) renderContent() string {
	if dv.skill == nil || len(dv.renderedContent) == 0 {
		return ""
	}

	metadataHeight := dv.estimateMetadataHeight()
	viewportHeight := dv.height - metadataHeight - 3 // 3 for spacing and scroll indicator
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	// Get visible lines
	startIdx := dv.scrollOffset
	endIdx := min(len(dv.renderedContent), startIdx+viewportHeight)
	visibleLines := dv.renderedContent[startIdx:endIdx]

	// Pad visible lines to fill entire viewport height
	for len(visibleLines) < viewportHeight {
		visibleLines = append(visibleLines, "")
	}

	// Join lines - Glamour has already styled the content with ANSI colors
	content := strings.Join(visibleLines, "\n")

	// Apply minimal styling to preserve Glamour's formatting
	styledContent := lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1).
		Render(content)

	return styledContent
}

// renderScrollIndicator renders the scroll indicator with action hints.
func (dv *DetailView) renderScrollIndicator() string {
	if dv.skill == nil {
		return ""
	}

	// If scanning, show scanning status prominently
	if dv.scanning {
		scanStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)
		return scanStyle.Render("🔒 Scanning skill for security threats...")
	}

	metadataHeight := dv.estimateMetadataHeight()
	viewportHeight := dv.height - metadataHeight - 3
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	totalLines := len(dv.renderedContent)
	endLine := min(totalLines, dv.scrollOffset+viewportHeight)

	var indicators []string

	if dv.scrollOffset > 0 {
		indicators = append(indicators, "↑ (scroll up)")
	}

	if dv.scrollOffset < dv.maxScroll {
		indicators = append(indicators, "↓ (scroll down)")
	}

	lineInfo := fmt.Sprintf("Line %d-%d of %d", dv.scrollOffset+1, endLine, totalLines)
	indicators = append(indicators, lineInfo)

	// Show install and scan options for all skills
	if dv.skill != nil {
		indicators = append(indicators, "i (install)")
		indicators = append(indicators, "S (scan)")
	}
	indicators = append(indicators, "? (help)  •  esc (back)")

	indicator := strings.Join(indicators, "  •  ")
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B6B6B")).
		Italic(true).
		Render(indicator)
}

// renderLoading renders a loading indicator.
func (dv *DetailView) renderLoading() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F1C40F")).
		Render("Loading skill details...")
}

// renderError renders an error message.
func (dv *DetailView) renderError() string {
	errMsg := "Failed to load skill"
	if dv.loadError != nil {
		errMsg = dv.loadError.Error()
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#DC143C")).
		Padding(1).
		Foreground(lipgloss.Color("#DC143C"))

	content := fmt.Sprintf("%s\n\nPress ESC to go back", errMsg)
	return box.Render(content)
}

// Helper functions

func formatTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%d minute%s ago", minutes, pluralize(minutes))
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hour%s ago", hours, pluralize(hours))
	}
	if duration < 30*24*time.Hour {
		days := int(duration.Hours()) / 24
		return fmt.Sprintf("%d day%s ago", days, pluralize(days))
	}

	return t.Format("2006-01-02")
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// HandleMouse handles mouse events for scrolling.
func (dv *DetailView) HandleMouse(msg tea.MouseMsg) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		dv.scrollOffset = max(0, dv.scrollOffset-3)
	case tea.MouseButtonWheelDown:
		dv.scrollOffset = min(dv.maxScroll, dv.scrollOffset+3)
	}
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (dv *DetailView) GetKeyboardCommands() ViewCommands {
	commands := []Command{
		{Key: "↑↓, k/j", Description: "Scroll content up/down"},
		{Key: "t", Description: "Jump to top of content"},
		{Key: "b", Description: "Jump to bottom of content"},
	}

	// Show install and scan commands for all skills (local and remote)
	if dv.skill != nil {
		commands = append(commands, Command{Key: "i", Description: "Toggle install/uninstall skill"})
		commands = append(commands, Command{Key: "S", Description: "Scan skill for threats"})
	}

	commands = append(commands,
		Command{Key: "c", Description: "Copy skill content to clipboard"},
		Command{Key: "Esc", Description: "Go back to previous view"},
	)

	return ViewCommands{
		ViewName: "Skill Details",
		Commands: commands,
	}
}
