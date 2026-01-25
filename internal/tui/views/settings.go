package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SettingsSection represents the different sections in the settings view
type SettingsSection int

const (
	SectionRepositories SettingsSection = iota
	SectionTools
	SectionCache
	SectionSecurity
)

// ToolStatus represents the installation status of a platform
type ToolStatus struct {
	Platform       installer.Platform
	Configured     bool
	Installed      bool
	Path           string
	Count          int // Managed skills (symlinks to cache)
	UnmanagedCount int // Skills not managed by skulto
	Size           int64
}

// CacheStats holds cache information
type CacheStats struct {
	Directory  string
	TotalSize  int64
	SkillCount int
	Formats    []string
}

// SecurityStats holds security scan statistics
type SecurityStats struct {
	TotalSkills   int64
	CleanSkills   int64
	WarningSkills int64
	CriticalCount int64
	HighCount     int64
	MediumCount   int64
	LowCount      int64
	LastScanAt    *time.Time
}

// RepositoryStatus represents a repository's sync status
type RepositoryStatus struct {
	Source       *models.Source
	SkillCount   int
	Status       string // "synced", "syncing", "error"
	LastSyncTime time.Time
}

// SettingsLoadedMsg is sent when settings data has been loaded
type SettingsLoadedMsg struct {
	Sources       []*models.Source
	Stats         *models.SkillStats
	Tools         map[installer.Platform]*ToolStatus
	CacheStats    *CacheStats
	SecurityStats *SecurityStats
	Err           error
}

// ClearCachedLocationsMsg is sent when user wants to clear cached install locations
type ClearCachedLocationsMsg struct{}

// ScanAllSkillsMsg is sent when user wants to scan all skills for security threats
type ScanAllSkillsMsg struct{}

// ScanAllSkillsCompleteMsg is sent when the scan all operation completes
type ScanAllSkillsCompleteMsg struct {
	ScannedCount int
	WarningCount int
	Err          error
}

// SettingsView displays application settings and configuration
type SettingsView struct {
	db           *db.DB
	cfg          *config.Config
	pathResolver *installer.PathResolver

	// Data
	sources       []*models.Source
	stats         *models.SkillStats
	tools         map[installer.Platform]*ToolStatus
	cacheStats    *CacheStats
	securityStats *SecurityStats

	// UI State
	width        int
	height       int
	scrollOffset int
	maxScroll    int
	section      SettingsSection

	// Loading state
	loading bool
	err     error
}

// NewSettingsView creates a new settings view
func NewSettingsView(database *db.DB, conf *config.Config) *SettingsView {
	return &SettingsView{
		db:           database,
		cfg:          conf,
		pathResolver: installer.NewPathResolver(conf),
		scrollOffset: 0,
		maxScroll:    0,
		section:      SectionRepositories,
		loading:      true,
		err:          nil,
		width:        80,
		height:       24,
		sources:      []*models.Source{},
		stats:        nil,
		tools:        make(map[installer.Platform]*ToolStatus),
		cacheStats:   nil,
	}
}

// SetSize sets the width and height of the view
func (sv *SettingsView) SetSize(width, height int) {
	sv.width = width
	sv.height = height
}

// Init initializes the settings view and loads data
func (sv *SettingsView) Init() tea.Cmd {
	sv.loading = true
	sv.err = nil
	sv.scrollOffset = 0

	return func() tea.Msg {
		// Fetch repositories
		sources, err := sv.db.ListSources()
		if err != nil {
			return SettingsLoadedMsg{Err: fmt.Errorf("failed to load sources: %w", err)}
		}

		// Convert to pointers
		var sourcePtrs []*models.Source
		for i := range sources {
			sourcePtrs = append(sourcePtrs, &sources[i])
		}

		// Fetch stats
		stats, err := sv.db.GetStats()
		if err != nil {
			return SettingsLoadedMsg{Err: fmt.Errorf("failed to load stats: %w", err)}
		}

		// Build tool status map
		tools := make(map[installer.Platform]*ToolStatus)
		for _, platform := range getAllPlatforms() {
			tools[platform] = sv.buildToolStatus(platform)
		}

		// Get cache stats
		cacheStats := sv.getCacheStats()

		return SettingsLoadedMsg{
			Sources:    sourcePtrs,
			Stats:      stats,
			Tools:      tools,
			CacheStats: cacheStats,
		}
	}
}

// Update handles keyboard input
func (sv *SettingsView) Update(key string) (back bool, cmd tea.Cmd) {
	switch key {
	// Navigation
	case "q", "esc":
		return true, nil

	// Scroll line by line
	case "j", "down":
		sv.scroll(1)
	case "k", "up":
		sv.scroll(-1)

	// Page scrolling
	case "d":
		// Down half page
		pageSize := max(1, (sv.height-4)/2)
		sv.scroll(pageSize)
	case "u":
		// Up half page
		pageSize := max(1, (sv.height-4)/2)
		sv.scroll(-pageSize)

	// Jump to start/end
	case "g":
		sv.scrollOffset = 0
	case "G":
		sv.scrollOffset = sv.maxScroll

	// Switch sections
	case "tab", "l", "right":
		sv.section = SettingsSection((int(sv.section) + 1) % 4)
		sv.scrollOffset = 0
	case "shift+tab", "h", "left":
		sv.section = SettingsSection((int(sv.section) - 1 + 4) % 4)
		sv.scrollOffset = 0

	// Clear cached install locations (only in Cache section)
	case "c":
		if sv.section == SectionCache {
			return false, func() tea.Msg { return ClearCachedLocationsMsg{} }
		}
	}
	return false, nil
}

// View renders the settings view
func (sv *SettingsView) View() string {
	if sv.loading {
		return "Loading settings..."
	}

	if sv.err != nil {
		return fmt.Sprintf("Error loading settings: %v", sv.err)
	}

	header := sv.renderHeader()

	// Render content lines based on current section
	var allLines []string
	switch sv.section {
	case SectionRepositories:
		allLines = sv.renderRepositoriesLines()
	case SectionTools:
		allLines = sv.renderToolsLines()
	case SectionCache:
		allLines = sv.renderCacheLines()
	case SectionSecurity:
		allLines = sv.renderSecurityLines()
	}

	// Calculate visible content with scroll
	contentHeight := sv.height - 4 // header + footer + margins
	content := sv.renderVisibleLines(allLines, contentHeight)

	// Add scroll indicator to the right
	content = sv.addScrollIndicator(content)

	footer := sv.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		"\n",
		content,
		"\n",
		footer,
	)
}

// Helper method to scroll with bounds checking
func (sv *SettingsView) scroll(delta int) {
	sv.scrollOffset = max(0, min(sv.maxScroll, sv.scrollOffset+delta))
}

// renderHeader returns the header line
func (sv *SettingsView) renderHeader() string {
	sectionNames := []string{"REPOSITORIES", "AI TOOLS", "CACHE", "SECURITY"}
	title := fmt.Sprintf("Settings - %s", sectionNames[sv.section])

	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Info).
		Bold(true)

	return headerStyle.Render(title)
}

// renderRepositoriesLines returns lines for the repositories section
func (sv *SettingsView) renderRepositoriesLines() []string {
	if len(sv.sources) == 0 {
		return []string{"(no repositories)"}
	}

	var lines []string
	count := len(sv.sources)
	lines = append(lines, fmt.Sprintf("REPOSITORIES (%d sources)", count))
	lines = append(lines, strings.Repeat("━", 50))

	for _, source := range sv.sources {
		if source == nil {
			continue
		}

		// Determine status icon
		statusIcon := "✓"
		statusText := "synced"

		// Try to determine last sync time from LastScrapedAt or UpdatedAt
		lastSync := source.UpdatedAt
		if source.LastScrapedAt != nil {
			lastSync = *source.LastScrapedAt
		}

		timeDiff := time.Since(lastSync)
		timeStr := formatTimeDiff(timeDiff)

		line := fmt.Sprintf("  %s %-40s [%s %s | %d skills]",
			statusIcon, source.FullName, statusText, timeStr, source.SkillCount)
		lines = append(lines, line)
	}

	return lines
}

// renderToolsLines returns lines for the AI tools section
func (sv *SettingsView) renderToolsLines() []string {
	var lines []string

	// Count configured tools
	configuredCount := 0
	for _, tool := range sv.tools {
		if tool.Configured {
			configuredCount++
		}
	}

	lines = append(lines, fmt.Sprintf("AI TOOLS INSTALLED (%d/6 selected)", configuredCount))
	lines = append(lines, strings.Repeat("━", 50))

	// List all platforms
	platforms := getAllPlatforms()
	for _, platform := range platforms {
		tool := sv.tools[platform]
		if tool == nil {
			continue
		}

		statusIcon := "✗"
		if tool.Configured {
			statusIcon = "✓"
		}

		line := fmt.Sprintf("  %s %s", statusIcon, platformDisplayName(platform))
		lines = append(lines, line)

		if tool.Configured {
			lines = append(lines, fmt.Sprintf("      Directory:  %s", FormatPath(tool.Path)))
			lines = append(lines, fmt.Sprintf("      Managed:    %d (Skulto installed)", tool.Count))
			if tool.UnmanagedCount > 0 {
				lines = append(lines, fmt.Sprintf("      Unmanaged:  %d (local only)", tool.UnmanagedCount))
			}
			lines = append(lines, fmt.Sprintf("      Size:       %s", formatBytes(tool.Size)))
		} else {
			lines = append(lines, "      (not configured)")
		}

		lines = append(lines, "")
	}

	return lines
}

// renderCacheLines returns lines for the cache information section
func (sv *SettingsView) renderCacheLines() []string {
	var lines []string
	lines = append(lines, "CACHE INFORMATION")
	lines = append(lines, strings.Repeat("━", 50))

	if sv.cacheStats == nil {
		lines = append(lines, "(cache information unavailable)")
		return lines
	}

	lines = append(lines, fmt.Sprintf("Cache Directory:    %s", FormatPath(sv.cacheStats.Directory)))
	lines = append(lines, fmt.Sprintf("Total Size:         %s", formatBytes(sv.cacheStats.TotalSize)))
	lines = append(lines, fmt.Sprintf("Skills Cached:      %d", sv.cacheStats.SkillCount))
	lines = append(lines, fmt.Sprintf("Platform Formats:   %d (%s)", len(sv.cacheStats.Formats), strings.Join(sv.cacheStats.Formats, ", ")))

	return lines
}

// renderSecurityLines returns lines for the security scan section
func (sv *SettingsView) renderSecurityLines() []string {
	var lines []string
	lines = append(lines, "SECURITY")
	lines = append(lines, strings.Repeat("━", 50))
	lines = append(lines, "")
	lines = append(lines, "Security stats coming soon.")
	lines = append(lines, "")
	lines = append(lines, "Use 'skulto scan --all' from CLI to scan skills.")
	return lines
}

// renderVisibleLines returns only the visible portion of content
func (sv *SettingsView) renderVisibleLines(allLines []string, height int) string {
	if len(allLines) == 0 {
		return "(no content)"
	}

	// Calculate visible range
	endIdx := min(len(allLines), sv.scrollOffset+height)
	visibleLines := allLines[sv.scrollOffset:endIdx]

	// Update max scroll
	sv.maxScroll = max(0, len(allLines)-height)

	return strings.Join(visibleLines, "\n")
}

// addScrollIndicator adds a visual scroll indicator to the right side
func (sv *SettingsView) addScrollIndicator(content string) string {
	if sv.maxScroll == 0 {
		return content // No scrolling needed
	}

	// Calculate scroll position indicator
	scrollPercent := float64(sv.scrollOffset) / float64(sv.maxScroll)

	indicator := "█" // Full (at start or end)
	if scrollPercent > 0.1 && scrollPercent < 0.9 {
		indicator = "▓" // Half
	}
	if scrollPercent > 0.2 && scrollPercent < 0.8 {
		indicator = "░" // Light
	}

	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = lines[i] + " " + indicator
	}

	return strings.Join(lines, "\n")
}

// renderFooter returns the footer line
func (sv *SettingsView) renderFooter() string {
	// Current position in section
	currentLine := sv.scrollOffset + 1
	totalLines := sv.maxScroll + (sv.height - 4)
	if sv.maxScroll == 0 {
		totalLines = 1
	}
	positionStr := fmt.Sprintf("Line %d/%d", currentLine, totalLines)

	// Section indicator
	sectionNames := []string{"Repositories", "Tools", "Cache", "Security"}
	sectionStr := fmt.Sprintf("[%d/4] %s", int(sv.section)+1, sectionNames[sv.section])

	// Help text (varies by section)
	helpStr := "j/k:scroll  d/u:page  g/G:jump  tab:switch  q:quit"
	if sv.section == SectionCache {
		helpStr = "j/k:scroll  c:clear cached locations  tab:switch  q:quit"
	}

	// Telemetry status lines
	var telemetryLines string
	if telemetry.IsEnabled() {
		trackingID := sv.db.GetOrCreateTrackingID()
		telemetryStyle := lipgloss.NewStyle().Foreground(theme.Current.Success)
		telemetryLines = telemetryStyle.Render("Telemetry: ON (set SKULTO_TELEMETRY_TRACKING_ENABLED=false to disable)") + "\n" +
			telemetryStyle.Render(fmt.Sprintf("Anon ID: %s", trackingID))
	} else {
		telemetryStyle := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
		telemetryLines = telemetryStyle.Render("Telemetry: OFF")
	}

	// Navigation line
	navStyle := lipgloss.NewStyle().Foreground(theme.Current.TextMuted).
		PaddingTop(1)
	navLine := navStyle.Render(fmt.Sprintf("%s | %s | %s", positionStr, sectionStr, helpStr))

	return telemetryLines + "\n" + navLine
}

// HandleSettingsLoaded handles the SettingsLoadedMsg
func (sv *SettingsView) HandleSettingsLoaded(msg SettingsLoadedMsg) {
	if msg.Err != nil {
		sv.err = msg.Err
	} else {
		sv.sources = msg.Sources
		sv.stats = msg.Stats
		sv.tools = msg.Tools
		sv.cacheStats = msg.CacheStats
		sv.securityStats = msg.SecurityStats
	}
	sv.loading = false
}

// Helper functions

// buildToolStatus builds tool status for a platform
func (sv *SettingsView) buildToolStatus(platform installer.Platform) *ToolStatus {
	status := &ToolStatus{
		Platform:       platform,
		Configured:     false,
		Installed:      false,
		Path:           "",
		Count:          0,
		UnmanagedCount: 0,
		Size:           0,
	}

	// Use PathResolver to get the correct base directory
	// Convert installer.Platform to installer.Platform (both are string types)
	basePath, err := sv.pathResolver.GetBasePath(installer.Platform(platform))
	if err != nil {
		return status
	}

	status.Path = basePath

	// Check if directory exists
	if _, err := os.Stat(status.Path); err == nil {
		status.Configured = true

		// Calculate directory size
		status.Size = calculateDirSize(status.Path)

		// Count managed (symlinks to cache) vs unmanaged skills
		if entries, err := os.ReadDir(status.Path); err == nil {
			for _, entry := range entries {
				if entry.IsDir() || (entry.Type()&os.ModeSymlink) != 0 {
					fullPath := filepath.Join(status.Path, entry.Name())
					// Check if it's a symlink pointing to cache
					if isSymlinkToCache(fullPath, sv.cfg.BaseDir) {
						status.Count++
					} else {
						status.UnmanagedCount++
					}
				}
			}
		}
	}

	return status
}

// isSymlinkToCache checks if a path is a symlink pointing to the cache directory
func isSymlinkToCache(path string, cacheDir string) bool {
	// Check if it's a symlink
	linkTarget, err := os.Readlink(path)
	if err != nil {
		return false // Not a symlink or can't read it
	}

	// Resolve both paths to absolute for comparison
	absTarget, err := filepath.Abs(linkTarget)
	if err != nil {
		return false
	}

	absCacheDir, err := filepath.Abs(cacheDir)
	if err != nil {
		return false
	}

	// Check if the symlink target points to something inside the cache directory
	return strings.HasPrefix(absTarget, absCacheDir)
}

// getCacheStats gets cache statistics
func (sv *SettingsView) getCacheStats() *CacheStats {
	cacheDir := sv.cfg.BaseDir

	stats := &CacheStats{
		Directory:  cacheDir,
		TotalSize:  0,
		SkillCount: 0,
		Formats:    []string{"claude", "cursor", "copilot", "codex", "opencode", "windsurf"},
	}

	// Count skills in cache
	skillsDir := filepath.Join(cacheDir, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		stats.SkillCount = len(entries)
	}

	// Calculate cache size
	stats.TotalSize = calculateDirSize(cacheDir)

	return stats
}

// Utility functions

// getAllPlatforms returns all supported platforms
func getAllPlatforms() []installer.Platform {
	return []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCursor,
		installer.PlatformCopilot,
		installer.PlatformCodex,
		installer.PlatformOpenCode,
		installer.PlatformWindsurf,
	}
}

// platformDisplayName returns the display name for a platform
func platformDisplayName(platform installer.Platform) string {
	switch platform {
	case installer.PlatformClaude:
		return "Claude Code"
	case installer.PlatformCursor:
		return "Cursor"
	case installer.PlatformCopilot:
		return "GitHub Copilot"
	case installer.PlatformCodex:
		return "Codex"
	case installer.PlatformOpenCode:
		return "OpenCode"
	case installer.PlatformWindsurf:
		return "Windsurf"
	default:
		return "Unknown"
	}
}

// formatTimeDiff formats a time duration as a relative string
func formatTimeDiff(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return fmt.Sprintf("%dw ago", int(d.Hours()/24/7))
}

// formatBytes formats a byte size as a human-readable string
func formatBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(bytes)
	unitIdx := 0

	for size >= 1024 && unitIdx < len(units)-1 {
		size /= 1024
		unitIdx++
	}

	return fmt.Sprintf("~%.1f %s", size, units[unitIdx])
}

// calculateDirSize calculates the total size of a directory
func calculateDirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
