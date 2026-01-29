package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/constants"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/detect"
	"github.com/asteroid-belt/skulto/internal/favorites"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/asteroid-belt/skulto/internal/search"
	"github.com/asteroid-belt/skulto/internal/security"
	"github.com/asteroid-belt/skulto/internal/skillgen"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/internal/tui/components"
	"github.com/asteroid-belt/skulto/internal/tui/views"
	"github.com/asteroid-belt/skulto/internal/vector"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewType identifies the current view.
type ViewType int

const (
	ViewHome ViewType = iota
	ViewSearch
	ViewReset
	ViewSkillDetail
	ViewTag
	ViewOnboardingIntro
	ViewOnboardingSetup
	ViewOnboardingTools
	ViewOnboardingSkills // Primary skills selection
	ViewAddSource
	ViewHelp
	ViewSettings
	ViewManage
)

// Model is the main Bubble Tea model for the TUI.
type Model struct {
	db        *db.DB
	cfg       *config.Config
	keymap    Keymap
	searchSvc *search.Service
	telemetry telemetry.Client
	favorites *favorites.Store

	// Background indexer for semantic search
	bgIndexer       *search.BackgroundIndexer
	indexProgressCh chan search.IndexProgress
	pullProgressCh  chan pullProgressMsg
	scanProgressCh  chan scanProgressMsg
	installer       *installer.Installer
	installService  *installer.InstallService

	// Views
	currentView          ViewType
	previousView         ViewType
	helpReturnView       ViewType // Where to return when closing help (preserves previousView chain)
	homeView             *views.HomeView
	searchView           *views.SearchView
	resetView            *views.ResetView
	detailView           *views.DetailView
	tagView              *views.TagView
	onboardingIntroView  *views.OnboardingIntroView
	onboardingSetupView  *views.OnboardingSetupView
	onboardingToolsView  *views.OnboardingToolsView
	onboardingSkillsView *views.OnboardingSkillsView
	addSourceView        *views.AddSourceView
	helpView             *views.HelpView
	settingsView         *views.SettingsView

	// State
	width    int
	height   int
	ready    bool
	quitting bool
	err      error

	// Session tracking
	sessionStart      time.Time
	viewsVisited      int
	searchesPerformed int
	skillsInstalled   int
	skillsUninstalled int
	reposAdded        int
	reposRemoved      int

	// Animation ticker
	ticker   *time.Ticker
	tickChan <-chan time.Time
	animTick int

	// Quick skill dialog
	newSkillDialog        *components.NewSkillDialog
	showingNewSkillDialog bool

	// Install location dialog
	locationDialog       *components.InstallLocationDialog
	cachedLocations      []installer.InstallLocation
	showLocationDialog   bool
	pendingInstallSkill  *models.Skill
	pendingInstallSource *models.Source
	pendingInstallSkills []models.Skill // For batch onboarding install

	// Quit confirmation dialog
	quitConfirmDialog  *components.ConfirmDialog
	showingQuitConfirm bool

	// Manage view
	manageView           *views.ManageView
	manageDialog         *components.ManageSkillDialog
	showManageDialog     bool
	confirmChangesDialog *components.ConfirmChangesDialog
	showConfirmChanges   bool
	pendingManageChanges struct {
		toInstall   []installer.InstallLocation
		toUninstall []installer.InstallLocation
	}
}

// Message types for Bubble Tea
type (
	skillsLoadedMsg struct{}
	tickMsg         struct{}
	pullStartedMsg  struct{}
	pullCompleteMsg struct {
		skillsFound int
		skillsNew   int
		localSynced int // Skills synced from ~/.skulto/skills
		cwdSynced   int // Skills synced from ./.skulto/skills
		err         error
	}
	pullProgressMsg struct {
		completed int
		total     int
		repoName  string
	}
	// scanProgressMsg carries security scan progress updates
	scanProgressMsg struct {
		scanned  int
		total    int
		repoName string
	}

	// indexProgressMsg carries background indexing progress updates
	indexProgressMsg struct {
		running   bool
		total     int
		completed int
		failed    int
		message   string
	}

	// localSkillsSyncMsg is sent when local skills sync completes
	localSkillsSyncMsg struct {
		indexed int
		err     error
	}

	// cwdSkillsSyncMsg is sent when CWD skills sync completes
	cwdSkillsSyncMsg struct {
		indexed int
		err     error
	}
)

// NewModel creates a new TUI model.
func NewModel(database *db.DB, conf *config.Config) *Model {
	keymap := DefaultKeymap()
	stats, _ := database.GetStats()

	skillCount := int64(0)
	tagCount := int64(0)
	if stats != nil {
		skillCount = int64(stats.TotalSkills)
		tagCount = int64(stats.TotalTags)
	}

	// Determine starting view based on onboarding state from database
	startingView := ViewHome
	state, _ := database.GetUserState()
	if state != nil && !state.IsOnboardingCompleted() {
		startingView = ViewOnboardingIntro
	}

	homeView := views.NewHomeView(database, conf)
	// Set initial stats for home view
	homeView.SetStats(skillCount, tagCount)

	// Create installer for skill installation management
	inst := installer.New(database, conf)

	// Create install service for unified installation operations
	instService := installer.NewInstallService(database, conf, nil)

	// Create search service (nil VectorStore means FTS-only mode)
	// VectorStore can be injected later via NewModelWithSearchService if needed
	searchSvc := search.New(database, nil, search.DefaultConfig())

	// Initialize favorites store
	paths := config.GetPaths(conf)
	favStore := favorites.NewStore(paths.Favorites)
	_ = favStore.Load() // Ignore error, will use empty store

	return &Model{
		db:                   database,
		cfg:                  conf,
		keymap:               keymap,
		searchSvc:            searchSvc,
		favorites:            favStore,
		currentView:          startingView,
		previousView:         startingView,
		homeView:             homeView,
		searchView:           views.NewSearchView(database, conf, searchSvc),
		resetView:            views.NewResetView(database, conf),
		detailView:           views.NewDetailView(database, conf, favStore),
		tagView:              views.NewTagView(database, conf),
		onboardingIntroView:  views.NewOnboardingIntroView(conf),
		onboardingSetupView:  views.NewOnboardingSetupView(conf),
		onboardingToolsView:  views.NewOnboardingToolsView(conf),
		onboardingSkillsView: views.NewOnboardingSkillsView(conf, database),
		addSourceView:        views.NewAddSourceView(database, conf),
		helpView:             views.NewHelpView(database, conf),
		manageView:           views.NewManageView(database, conf, instService, nil),
		ticker:               time.NewTicker(500 * time.Millisecond),
		animTick:             0,
		installer:            inst,
		installService:       instService,
		newSkillDialog:       components.NewNewSkillDialog(),
		quitConfirmDialog:    newQuitDialog(),
	}
}

// NewModelWithIndexer creates a new TUI model with optional background indexer.
func NewModelWithIndexer(database *db.DB, conf *config.Config, indexer *search.BackgroundIndexer, tc telemetry.Client) *Model {
	keymap := DefaultKeymap()
	stats, _ := database.GetStats()

	skillCount := int64(0)
	tagCount := int64(0)
	if stats != nil {
		skillCount = int64(stats.TotalSkills)
		tagCount = int64(stats.TotalTags)
	}

	// Determine starting view based on onboarding state from database
	startingView := ViewHome
	state, _ := database.GetUserState()
	if state != nil && !state.IsOnboardingCompleted() {
		startingView = ViewOnboardingIntro
	}

	homeView := views.NewHomeView(database, conf)
	homeView.SetStats(skillCount, tagCount)

	// Create search service with vector store from indexer if available
	var vectorStore vector.VectorStore
	if indexer != nil {
		vectorStore = indexer.VectorStore()
	}
	searchSvc := search.New(database, vectorStore, search.DefaultConfig())

	// Create installer for skill installation management
	inst := installer.New(database, conf)

	// Create install service for unified installation operations
	instService := installer.NewInstallService(database, conf, tc)

	// Initialize favorites store
	paths := config.GetPaths(conf)
	favStore := favorites.NewStore(paths.Favorites)
	_ = favStore.Load() // Ignore error, will use empty store

	searchView := views.NewSearchView(database, conf, searchSvc)
	detailView := views.NewDetailView(database, conf, favStore)

	// Track app started
	sourceCount := 0
	sources, _ := database.ListSources()
	if sources != nil {
		sourceCount = len(sources)
	}
	tc.TrackAppStarted("tui", sourceCount > 0, sourceCount)

	return &Model{
		db:                   database,
		cfg:                  conf,
		keymap:               keymap,
		telemetry:            tc,
		installer:            inst,
		installService:       instService,
		searchSvc:            searchSvc,
		favorites:            favStore,
		bgIndexer:            indexer,
		indexProgressCh:      make(chan search.IndexProgress, 10),
		pullProgressCh:       make(chan pullProgressMsg, 20),
		scanProgressCh:       make(chan scanProgressMsg, 50),
		currentView:          startingView,
		previousView:         startingView,
		homeView:             homeView,
		searchView:           searchView,
		resetView:            views.NewResetView(database, conf),
		detailView:           detailView,
		tagView:              views.NewTagView(database, conf),
		onboardingIntroView:  views.NewOnboardingIntroView(conf),
		onboardingSetupView:  views.NewOnboardingSetupView(conf),
		onboardingToolsView:  views.NewOnboardingToolsView(conf),
		onboardingSkillsView: views.NewOnboardingSkillsView(conf, database),
		addSourceView:        views.NewAddSourceView(database, conf),
		helpView:             views.NewHelpView(database, conf),
		settingsView:         views.NewSettingsView(database, conf),
		manageView:           views.NewManageView(database, conf, instService, tc),
		sessionStart:         time.Now(),
		ticker:               time.NewTicker(500 * time.Millisecond),
		animTick:             0,
		newSkillDialog:       components.NewNewSkillDialog(),
		quitConfirmDialog:    newQuitDialog(),
	}
}

// newQuitDialog creates a quit confirmation dialog with feedback URL in footer.
func newQuitDialog() *components.ConfirmDialog {
	dialog := components.NewConfirmDialog("Quit Skulto?", "Are you sure you want to exit?")
	dialog.SetFooter("Feedback? " + constants.FeedbackURL)
	return dialog
}

// viewName returns a string name for a ViewType.
func (v ViewType) String() string {
	switch v {
	case ViewHome:
		return "home"
	case ViewSearch:
		return "search"
	case ViewReset:
		return "reset"
	case ViewSkillDetail:
		return "detail"
	case ViewTag:
		return "tag"
	case ViewOnboardingIntro:
		return "onboarding_intro"
	case ViewOnboardingSetup:
		return "onboarding_setup"
	case ViewOnboardingTools:
		return "onboarding_tools"
	case ViewAddSource:
		return "add_source"
	case ViewHelp:
		return "help"
	case ViewSettings:
		return "settings"
	case ViewManage:
		return "manage"
	default:
		return "unknown"
	}
}

// trackViewNavigation tracks view changes for telemetry.
func (m *Model) trackViewNavigation(toView ViewType) {
	m.telemetry.TrackViewNavigated(toView.String(), m.currentView.String())
	m.viewsVisited++
}

// Init initializes the model and starts the animation ticker.
func (m *Model) Init() tea.Cmd {
	m.homeView.Init(m.telemetry)
	m.searchView.Init(m.telemetry)
	m.detailView.Init(m.telemetry)
	m.onboardingIntroView.Init()
	m.onboardingSetupView.Init()
	m.onboardingToolsView.Init()
	m.addSourceView.Init()
	m.helpView.Init(m.telemetry)

	m.tickChan = m.ticker.C

	// Run repository cleanup in background (non-blocking)
	go m.cleanupOldRepositories()

	// Sync install state SYNCHRONOUSLY before loading data
	// This ensures is_installed flags match actual symlinks on disk
	m.syncInstallState()

	// Start background indexer if available
	cmds := []tea.Cmd{
		m.tickCmd(),
		m.loadDataCmd(),
		m.syncLocalSkillsCmd(), // Sync ~/.skulto/skills on startup
		m.syncCwdSkillsCmd(),   // Sync ./.skulto/skills (cwd) on startup
	}

	// If onboarding is completed, sync primary skills repo in background
	state, _ := m.db.GetUserState()
	if state != nil && state.IsOnboardingCompleted() {
		cmds = append(cmds, m.syncPrimarySkillsCmd())
	}

	if m.bgIndexer != nil {
		// Start indexing in background
		ctx := context.Background()
		_ = m.bgIndexer.Start(ctx, m.indexProgressCh)
		// Add command to watch for progress updates
		cmds = append(cmds, m.watchIndexProgressCmd())
	}

	return tea.Batch(cmds...)
}

// watchIndexProgressCmd returns a command that watches for index progress updates.
func (m *Model) watchIndexProgressCmd() tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-m.indexProgressCh
		if !ok {
			return nil
		}
		return indexProgressMsg{
			running:   progress.Running,
			total:     progress.Total,
			completed: progress.Completed,
			failed:    progress.Failed,
			message:   progress.Message,
		}
	}
}

// watchPullProgressCmd returns a command that watches for pull progress updates.
func (m *Model) watchPullProgressCmd() tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-m.pullProgressCh
		if !ok {
			return nil
		}
		return progress
	}
}

// watchScanProgressCmd returns a command that watches for security scan progress updates.
func (m *Model) watchScanProgressCmd() tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-m.scanProgressCh
		if !ok {
			return nil
		}
		return progress
	}
}

// cleanupOldRepositories removes old cloned repositories in the background.
func (m *Model) cleanupOldRepositories() {
	// Only cleanup if using git clone mode
	if !m.cfg.GitHub.UseGitClone {
		return
	}

	cfg := scraper.ScraperConfig{
		Token:        m.cfg.GitHub.Token,
		DataDir:      m.cfg.BaseDir,
		RepoCacheTTL: m.cfg.GitHub.RepoCacheTTL,
		UseGitClone:  true,
	}
	s := scraper.NewScraperWithConfig(cfg, m.db)
	_ = s.CleanupOldRepositories() // Ignore errors - cleanup is best-effort
}

// syncInstallState reconciles the database install state with actual symlinks on disk.
// This runs in background on app launch to ensure is_installed flags match reality.
func (m *Model) syncInstallState() {
	if m.installer == nil {
		return
	}
	ctx := context.Background()
	_ = m.installer.SyncInstallState(ctx) // Ignore errors - sync is best-effort
}

// getCurrentViewCommands returns the keyboard commands for the current view.
func (m *Model) getCurrentViewCommands() views.ViewCommands {
	switch m.currentView {
	case ViewHome:
		return m.homeView.GetKeyboardCommands()
	case ViewSearch:
		return m.searchView.GetKeyboardCommands()
	case ViewSkillDetail:
		return m.detailView.GetKeyboardCommands()
	case ViewTag:
		return m.tagView.GetKeyboardCommands()
	case ViewAddSource:
		return m.addSourceView.GetKeyboardCommands()
	case ViewReset:
		return m.resetView.GetKeyboardCommands()
	case ViewOnboardingIntro:
		return m.onboardingIntroView.GetKeyboardCommands()
	case ViewOnboardingSetup:
		return m.onboardingSetupView.GetKeyboardCommands()
	case ViewOnboardingTools:
		return m.onboardingToolsView.GetKeyboardCommands()
	case ViewOnboardingSkills:
		return m.onboardingSkillsView.GetKeyboardCommands()
	case ViewManage:
		return m.manageView.GetKeyboardCommands()
	default:
		return views.ViewCommands{ViewName: "UNKNOWN VIEW", Commands: []views.Command{}}
	}
}

// Update handles all messages and user input.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle quick skill dialog if showing
		if m.showingNewSkillDialog {
			cmd := m.newSkillDialog.Update(msg)

			if m.newSkillDialog.IsCancelled() {
				m.showingNewSkillDialog = false
				return m, nil
			}

			if m.newSkillDialog.IsConfirmed() {
				m.showingNewSkillDialog = false

				// Check if user wants to install the skill
				if m.newSkillDialog.WantsInstall() {
					skillInfo := m.newSkillDialog.GetInstallSkillInfo()
					locations := m.newSkillDialog.GetInstallLocations()
					if skillInfo != nil && len(locations) > 0 {
						return m, m.installLocalSkillCmd(*skillInfo, locations)
					}
				}

				// Only call saveNewSkillCmd for legacy preview flow
				// In StateResult (Claude flow), skills are already saved
				if m.newSkillDialog.NeedsSave() {
					return m, m.saveNewSkillCmd()
				}
				// Claude flow - just close and refresh home view
				m.homeView.Init(m.telemetry)
				return m, nil
			}

			return m, cmd
		}

		// Handle new skill dialog trigger (n) - only in Home and Detail views
		// Skip if location dialog is showing (n selects none in that dialog)
		if key == "n" && !m.showLocationDialog && (m.currentView == ViewHome || m.currentView == ViewSkillDetail) {
			if !m.showingNewSkillDialog {
				m.newSkillDialog.Reset()
				// Set platforms for potential installation after skill creation
				userState, _ := m.db.GetUserState()
				if userState != nil {
					platforms := parsePlatformsFromState(userState)
					m.newSkillDialog.SetPlatforms(platforms)
				}
				m.newSkillDialog.SetSize(m.width, m.height)
				m.showingNewSkillDialog = true
			}
			return m, nil
		}

		// Handle location dialog if showing
		if m.showLocationDialog && m.locationDialog != nil {
			m.locationDialog.Update(msg)

			if m.locationDialog.IsConfirmed() {
				selectedLocations := m.locationDialog.GetSelectedLocations()
				// Only cache if user explicitly checked "Remember these locations"
				if m.locationDialog.ShouldRememberLocations() {
					m.cachedLocations = selectedLocations
				}
				m.showLocationDialog = false

				// Persist newly selected platforms with scope to agent_preferences
				agentScopes := make(map[string]string)
				for _, loc := range selectedLocations {
					agentScopes[string(loc.Platform)] = string(loc.Scope)
				}
				_ = m.db.EnableAgentsWithScopes(agentScopes)

				// Check if this is from onboarding flow (batch install)
				if len(m.pendingInstallSkills) > 0 {
					return m, m.installBatchSkillsCmd(m.pendingInstallSkills, selectedLocations)
				}

				// Single skill install (from detail view)
				skill := m.pendingInstallSkill
				source := m.pendingInstallSource
				m.pendingInstallSkill = nil
				m.pendingInstallSource = nil
				// Use appropriate installer based on skill type
				if skill.IsLocal {
					return m, m.installLocalSkillFromDetailCmd(skill, selectedLocations)
				}
				return m, m.installToLocationsCmd(skill, source, selectedLocations)
			}

			if m.locationDialog.IsCancelled() {
				m.showLocationDialog = false

				// Check if this was from onboarding flow
				if len(m.pendingInstallSkills) > 0 {
					m.pendingInstallSkills = nil
					// Return to home and complete onboarding anyway
					m.currentView = ViewHome
					return m, m.completeOnboarding()
				}

				// Single skill cancellation (from detail view)
				m.pendingInstallSkill = nil
				m.pendingInstallSource = nil
				// Reset the installing state in detail view
				m.detailView.SetInstallingState(false)
				// Revert the optimistic UI update
				if skill := m.detailView.Skill(); skill != nil {
					skill.IsInstalled = !skill.IsInstalled
				}
				return m, nil
			}

			return m, nil
		}

		// Handle quit confirmation dialog if showing
		if m.showingQuitConfirm {
			switch key {
			case "left", "right", "h", "l", "tab":
				m.quitConfirmDialog.Toggle()
			case "enter":
				if m.quitConfirmDialog.IsYesSelected() {
					m.trackSessionExit()
					m.quitting = true
					return m, tea.Quit
				}
				// "No" selected - close the dialog
				m.showingQuitConfirm = false
				m.quitConfirmDialog.SelectNo() // Reset to default
			case "esc", "n":
				// Cancel - close the dialog
				m.showingQuitConfirm = false
				m.quitConfirmDialog.SelectNo() // Reset to default
			case "y":
				// Quick confirm with "y" key
				m.trackSessionExit()
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		// Global quit - ctrl+c always quits immediately, "q" shows confirmation
		if key == "ctrl+c" {
			m.trackSessionExit()
			m.quitting = true
			return m, tea.Quit
		}
		if key == "q" {
			m.showingQuitConfirm = true
			return m, nil
		}

		// Handle reset key
		if key == "ctrl+r" {
			m.currentView = ViewReset
			m.resetView.Init()
			return m, nil
		}

		// Handle help key
		if key == "?" {
			m.helpReturnView = m.currentView // Use separate field to preserve previousView chain
			viewCommands := m.getCurrentViewCommands()
			m.currentView = ViewHelp
			m.helpView.SetViewCommands(viewCommands)
			m.helpView.Init(m.telemetry)
			return m, nil
		}

		// Handle pull key (only on home view)
		if key == "p" && m.currentView == ViewHome && !m.homeView.IsPulling() {
			m.homeView.SetPulling(true)
			return m, tea.Batch(
				m.syncCmd(m.cfg.GitHub.Token),
				m.watchPullProgressCmd(),
				m.watchScanProgressCmd(),
				m.syncPrimarySkillsCmd(), // Also sync primary repo on pull
			)
		}

		// Handle view-specific keys
		switch m.currentView {
		case ViewHome:
			action := m.homeView.Update(key)

			switch action {
			case views.HomeActionSearch:
				m.trackViewNavigation(ViewSearch)
				m.currentView = ViewSearch
				m.searchView.Init(m.telemetry)
			case views.HomeActionAddSource:
				m.trackViewNavigation(ViewAddSource)
				m.currentView = ViewAddSource
				m.addSourceView.Init()
			case views.HomeActionSettings:
				m.previousView = ViewHome
				m.trackViewNavigation(ViewSettings)
				m.currentView = ViewSettings
				return m, m.settingsView.Init()
			case views.HomeActionSelectTag:
				if tag := m.homeView.GetSelectedTag(); tag != nil {
					m.previousView = ViewHome
					m.currentView = ViewTag
					return m, m.tagView.SetTag(tag)
				}
			case views.HomeActionSelectSkill:
				if skill := m.homeView.GetSelectedSkill(); skill != nil {
					m.previousView = ViewHome
					m.currentView = ViewSkillDetail
					return m, m.detailView.SetSkill(skill.ID)
				}
			}

			// Handle 'm' key to open Manage view
			if key == "m" {
				m.trackViewNavigation(ViewManage)
				m.previousView = ViewHome
				m.currentView = ViewManage
				return m, m.manageView.Init()
			}

		case ViewSearch:
			switch key {
			case "esc":
				m.trackViewNavigation(ViewHome)
				m.currentView = ViewHome
			case "enter":
				// Check if tag is selected (in tag mode)
				if tag := m.searchView.GetSelectedTag(); tag != nil {
					m.previousView = ViewSearch
					m.currentView = ViewTag
					return m, m.tagView.SetTag(tag)
				}
				// Otherwise select skill and show detail view
				if skill := m.searchView.GetSelectedSkill(); skill != nil {
					m.previousView = ViewSearch
					m.trackViewNavigation(ViewSkillDetail)
					m.currentView = ViewSkillDetail
					return m, m.detailView.SetSkill(skill.ID)
				}
			default:
				// Pass the key to search view and handle the returned command
				back, tagSelected, cmd := m.searchView.Update(key)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				if tagSelected {
					if tag := m.searchView.GetSelectedTag(); tag != nil {
						m.previousView = ViewSearch
						m.currentView = ViewTag
						return m, m.tagView.SetTag(tag)
					}
				}
				if back {
					m.currentView = ViewHome
				}
			}

		case ViewSkillDetail:
			back, detailCmd := m.detailView.Update(key)
			skill := m.detailView.Skill()
			if skill != nil && m.detailView.IsInstalling() {
				// User pressed 'i' - perform async install/uninstall
				if skill.IsInstalled {
					// Installing - always show location dialog for detail view installs
					m.showLocationDialog = true
					m.pendingInstallSkill = skill
					m.pendingInstallSource = skill.Source // nil for local skills
					// Reset installing state - dialog is showing, not actually installing yet
					m.detailView.SetInstallingState(false)

					// Create dialog with saved preferences + detection
					savedScopes, _ := m.db.GetEnabledAgentScopes()
					detectionResults := detect.DetectAll()
					allPlatforms := installer.AllPlatforms()

					// Fall back to legacy if no saved prefs and no detection
					if len(savedScopes) == 0 && len(detectionResults) == 0 {
						// Try legacy UserState
						userState, _ := m.db.GetUserState()
						platforms := parsePlatformsFromState(userState)
						if len(platforms) == 0 {
							m.showLocationDialog = false
							m.detailView.SetInstallingState(true)
							if skill.IsLocal {
								return m, func() tea.Msg {
									return views.SkillInstalledMsg{
										Success: false,
										Err:     fmt.Errorf("no AI tool platforms configured - please run skulto setup"),
									}
								}
							}
							return m, m.installCmd(skill)
						}
						m.locationDialog = components.NewInstallLocationDialog(platforms)
					} else {
						// Get last install locations for above-the-fold grouping
						var lastInstall []components.LastInstallChoice
						if lastLocs, err := m.db.GetLastInstallLocations(); err == nil {
							for _, loc := range lastLocs {
								lastInstall = append(lastInstall, components.LastInstallChoice{
									Platform: loc.Platform,
									Scope:    loc.Scope,
								})
							}
						}
						m.locationDialog = components.NewInstallLocationDialogWithPrefs(
							allPlatforms, savedScopes, detectionResults, lastInstall,
						)
					}
					m.locationDialog.SetWidth(m.width)
					return m, nil
				}
				// Uninstalling - use existing uninstall
				return m, m.installCmd(skill)
			}
			if back {
				m.currentView = m.previousView
				m.homeView.Init(m.telemetry)
				return m, nil
			}
			// Return any command from detail view (e.g., scan request)
			if detailCmd != nil {
				return m, detailCmd
			}

		case ViewTag:
			back, openDetail := m.tagView.Update(key)
			if openDetail {
				// Navigate to detail view
				if skill := m.tagView.GetSelectedSkill(); skill != nil {
					m.previousView = ViewTag
					m.currentView = ViewSkillDetail
					return m, m.detailView.SetSkill(skill.ID)
				}
			}
			if back {
				m.currentView = ViewHome
				// Refresh home view if returning to it
				m.homeView.Init(m.telemetry)
			}

		case ViewReset:
			// Pass the key to reset view
			back, _, cmd := m.resetView.Update(key)
			if cmd != nil {
				// Close the old database before starting async reset
				if m.db != nil {
					_ = m.db.Close()
					m.db = nil
				}
				cmds = append(cmds, cmd)
			}
			if back {
				// User cancelled, go back to home
				m.currentView = ViewHome
			}

		case ViewAddSource:
			back, wasSuccessful := m.addSourceView.Update(key)
			if back {
				if wasSuccessful {
					// Get the repository URL
					repoURL := m.addSourceView.GetRepositoryURL()

					// Set pulling state and navigate back to home
					m.currentView = ViewHome
					m.homeView.SetPulling(true)

					// Trigger the add source command
					return m, m.addSourceCmd(repoURL)
				} else {
					// User cancelled - just go back to home
					m.currentView = ViewHome
				}
			}

		case ViewHelp:
			back, _ := m.helpView.Update(key)
			if back {
				m.currentView = m.helpReturnView
				return m, nil
			}

		case ViewSettings:
			back, cmd := m.settingsView.Update(key)
			if back {
				m.currentView = m.previousView
			}
			if cmd != nil {
				return m, cmd
			}

		case ViewManage:
			// Handle manage dialog if showing
			if m.showManageDialog && m.manageDialog != nil {
				m.manageDialog.HandleKey(key)

				if m.manageDialog.IsCancelled() {
					m.showManageDialog = false
					return m, nil
				}

				if m.manageDialog.IsConfirmed() {
					toInstall, toUninstall := m.manageDialog.GetChanges()

					// If there are removals, check if we need confirmation
					if len(toUninstall) > 0 {
						skipConfirm, _ := m.db.GetSkipUninstallConfirm()
						if !skipConfirm {
							// Show confirmation dialog
							skill := m.manageDialog.GetSkill()
							m.confirmChangesDialog = components.NewConfirmChangesDialog(
								skill.Slug,
								toInstall,
								toUninstall,
							)
							m.confirmChangesDialog.SetWidth(m.width)
							m.pendingManageChanges.toInstall = toInstall
							m.pendingManageChanges.toUninstall = toUninstall
							m.showManageDialog = false
							m.showConfirmChanges = true
							return m, nil
						}
					}

					// Execute changes directly (no removals or skip confirm)
					m.showManageDialog = false
					skill := m.manageDialog.GetSkill()
					return m, m.executeManageChangesCmd(skill.Slug, toInstall, toUninstall)
				}

				return m, nil
			}

			// Handle confirm changes dialog if showing
			if m.showConfirmChanges && m.confirmChangesDialog != nil {
				m.confirmChangesDialog.HandleKey(key)

				if m.confirmChangesDialog.IsCancelled() {
					m.showConfirmChanges = false
					// Re-show manage dialog
					m.showManageDialog = true
					return m, nil
				}

				if m.confirmChangesDialog.IsConfirmed() {
					// Save "do not show again" preference if checked
					if m.confirmChangesDialog.DoNotShowAgain() {
						_ = m.db.SetSkipUninstallConfirm(true)
					}

					m.showConfirmChanges = false
					skill := m.manageDialog.GetSkill()
					return m, m.executeManageChangesCmd(
						skill.Slug,
						m.pendingManageChanges.toInstall,
						m.pendingManageChanges.toUninstall,
					)
				}

				return m, nil
			}

			// Handle ManageView updates
			action, cmd := m.manageView.Update(key)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			switch action {
			case views.ManageActionBack:
				m.currentView = ViewHome
				m.homeView.Init(m.telemetry)
				return m, nil

			case views.ManageActionSelectSkill:
				if skill := m.manageView.GetSelectedSkill(); skill != nil {
					// Use all known platforms so the dialog shows everything
					platforms := installer.AllPlatforms()

					m.manageDialog = components.NewManageSkillDialog(*skill, platforms)
					m.manageDialog.SetWidth(m.width)
					m.showManageDialog = true
				}
				return m, nil
			}

		case ViewOnboardingIntro:
			back, skipped := m.onboardingIntroView.Update(key)
			if back {
				if skipped {
					m.telemetry.TrackOnboardingSkipped("intro")
					m.trackViewNavigation(ViewHome)
					m.currentView = ViewHome
					return m, m.completeOnboardingWithSkip(true, 1)
				} else {
					// Continue to setup phase
					m.trackViewNavigation(ViewOnboardingSetup)
					m.currentView = ViewOnboardingSetup
					m.onboardingSetupView.Init()
				}
			}

		case ViewOnboardingSetup:
			back, skipped := m.onboardingSetupView.Update(key)
			if back {
				if skipped {
					m.telemetry.TrackOnboardingSkipped("setup")
					m.trackViewNavigation(ViewHome)
					m.currentView = ViewHome
					return m, m.completeOnboardingWithSkip(true, 2)
				} else {
					// Continue to tools phase
					m.trackViewNavigation(ViewOnboardingTools)
					m.currentView = ViewOnboardingTools
					m.onboardingToolsView.Init()
				}
			}

		case ViewOnboardingTools:
			done, _ := m.onboardingToolsView.Update(key)
			if done {
				// Save selected AI tools before changing view
				selectedPlatforms := m.onboardingToolsView.GetSelectedPlatforms()
				if len(selectedPlatforms) > 0 {
					var platformCodes []string
					for _, platform := range selectedPlatforms {
						platformCodes = append(platformCodes, string(platform))
					}
					// Save to UserState.AITools for backward compat
					toolsString := strings.Join(platformCodes, ",")
					if err := m.db.UpdateAITools(toolsString); err != nil {
						m.setError(fmt.Errorf("failed to save AI tools: %w", err), "database")
						return m, nil
					}
					// Save to agent_preferences table
					if err := m.db.SetAgentsEnabled(platformCodes); err != nil {
						m.setError(fmt.Errorf("failed to save agent preferences: %w", err), "database")
						return m, nil
					}
				}

				// Transition to skills selection instead of completing
				m.currentView = ViewOnboardingSkills
				m.onboardingSkillsView.Init()
				m.onboardingSkillsView.SetSize(m.width, m.height)
				return m, m.startPrimarySkillsFetchCmd()
			}

		case ViewOnboardingSkills:
			done, skipped, cmd := m.onboardingSkillsView.Update(key)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			if done {
				if skipped {
					m.telemetry.TrackOnboardingSkipped("skills")
					m.currentView = ViewHome
					return m, m.completeOnboarding()
				} else {
					selectedSkills := m.onboardingSkillsView.GetSelectedSkills()
					replaceSkills := m.onboardingSkillsView.GetReplaceSkills()
					allSkills := append(selectedSkills, replaceSkills...)

					if len(allSkills) > 0 {
						m.pendingInstallSkills = allSkills
						m.locationDialog = components.NewInstallLocationDialogForOnboarding(
							m.onboardingToolsView.GetSelectedPlatforms(),
						)
						m.showLocationDialog = true
					} else {
						m.currentView = ViewHome
						return m, m.completeOnboarding()
					}
				}
			}
		}

	case views.ResetCompleteMsg:
		if msg.Success && msg.NewDB != nil {
			// Reset completed successfully - reinitialize with new database
			m.db = msg.NewDB
			m.finishResetWithNewDB()
			// Go directly to onboarding
			m.currentView = ViewOnboardingIntro
			m.onboardingIntroView.Init()
		} else {
			// Reset failed - show error and go back to home
			m.resetView.HandleResetComplete(msg)
			if msg.Err != nil {
				m.setError(msg.Err, "reset")
			}
		}

	case views.SkillLoadedMsg:
		m.detailView.HandleSkillLoaded(msg)

	case views.SkillInstalledMsg:
		if msg.Success {
			// Installation/uninstallation completed successfully
			m.detailView.SetInstallingState(false)
		} else {
			// Installation/uninstallation failed
			if m.detailView.Skill() != nil {
				// Revert the optimistic UI update
				m.detailView.Skill().IsInstalled = !m.detailView.Skill().IsInstalled
			}
			m.detailView.SetInstallError(msg.Err)
		}

	case views.SkillsLoadedMsg:
		m.tagView.HandleSkillsLoaded(msg)

	case views.SkillScanRequestMsg:
		// Set scanning state and scan the requested skill
		m.detailView.SetScanning(true)
		return m, m.scanSkillCmd(msg.SkillID)

	case views.SkillScanCompleteMsg:
		// Clear scanning state and refresh the detail view
		m.detailView.SetScanning(false)
		if m.currentView == ViewSkillDetail {
			return m, m.detailView.SetSkill(msg.SkillID)
		}

	case views.PrimarySkillsFetchedMsg:
		m.onboardingSkillsView.HandleSkillsFetched(msg.Skills, msg.Err)

	case views.SettingsLoadedMsg:
		m.settingsView.HandleSettingsLoaded(msg)

	case views.ManageSkillsLoadedMsg:
		m.manageView.HandleManageSkillsLoaded(msg)

	case manageChangesCompleteMsg:
		if msg.err != nil {
			m.setError(msg.err, "manage_changes")
		}
		// Refresh the manage view
		return m, m.manageView.RefreshSkills()

	case views.ClearCachedLocationsMsg:
		// Clear cached install locations so user sees the dialog again
		m.cachedLocations = nil

	case views.NewSkillSavedMsg:
		if msg.Err != nil {
			m.setError(msg.Err, "skill_save")
		}
		// Could add success notification here in future

	case views.NewSkillInstallCompleteMsg:
		if msg.Success {
			// Installation succeeded - refresh home view
			m.homeView.Init(m.telemetry)
		}
		// Installation failed - skill is still in Skulto, don't block user
		// Error is logged but not shown to user since skill creation was successful

	case components.NewSkillStreamMsg, components.NewSkillTickMsg:
		if m.showingNewSkillDialog {
			cmd := m.newSkillDialog.Update(msg)
			cmds = append(cmds, cmd)
		}

	case components.NewSkillClaudeFinishedMsg:
		if m.showingNewSkillDialog {
			cmd := m.newSkillDialog.Update(msg)
			cmds = append(cmds, cmd)

			// Always sync local skills when returning from Claude
			// This ensures we catch any skills saved to ~/.skulto/skills
			cmds = append(cmds, m.syncLocalSkillsCmd())
		}

	case tea.MouseMsg:
		// Handle mouse wheel scrolling in views that support it
		switch m.currentView {
		case ViewSearch:
			m.searchView.HandleMouse(msg)
		case ViewHome:
			m.homeView.HandleMouse(msg)
		case ViewSkillDetail:
			m.detailView.HandleMouse(msg)
		case ViewTag:
			m.tagView.HandleMouse(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		// Content height for views that don't render their own header/footer
		contentHeight := m.height - 4
		if contentHeight < 5 {
			contentHeight = 5
		}
		m.homeView.SetSize(m.width, contentHeight)
		m.searchView.SetSize(m.width, contentHeight)
		m.resetView.SetSize(m.width, contentHeight)
		m.detailView.SetSize(m.width, contentHeight)
		m.tagView.SetSize(m.width, contentHeight)
		// Onboarding views use full height
		m.onboardingIntroView.SetSize(m.width, m.height)
		m.onboardingSetupView.SetSize(m.width, m.height)
		m.onboardingToolsView.SetSize(m.width, m.height)
		// AddSourceView uses content height
		m.addSourceView.SetSize(m.width, contentHeight)
		// HelpView uses content height
		m.helpView.SetSize(m.width, contentHeight)
		// SettingsView uses content height
		m.settingsView.SetSize(m.width, contentHeight)
		// ManageView uses content height
		m.manageView.SetSize(m.width, contentHeight)
		// NewSkillDialog uses full size
		m.newSkillDialog.SetSize(m.width, m.height)

	case tickMsg:
		m.animTick++
		m.homeView.UpdateAnimation()
		cmds = append(cmds, m.tickCmd())

	case pullStartedMsg:
		m.homeView.SetPulling(true)

	case pullProgressMsg:
		// Update home view with pull progress
		m.homeView.SetPulling(true)
		m.homeView.SetPullProgress(msg.completed, msg.total, msg.repoName)
		// Continue watching for more progress updates
		cmds = append(cmds, m.watchPullProgressCmd())

	case scanProgressMsg:
		// Update home view with security scan progress
		m.homeView.SetScanProgress(msg.scanned, msg.total, msg.repoName)
		// Continue watching for more scan progress updates
		cmds = append(cmds, m.watchScanProgressCmd())

	case indexProgressMsg:
		// Update home view with indexing progress
		if msg.running {
			progressText := fmt.Sprintf("Indexing %d/%d skills...", msg.completed, msg.total)
			m.homeView.SetIndexing(true, progressText)
			// Continue watching for more progress updates
			cmds = append(cmds, m.watchIndexProgressCmd())
		} else {
			// Indexing complete or stopped
			m.homeView.SetIndexing(false, "")
		}

	case pullCompleteMsg:
		m.homeView.SetPulling(false)
		m.homeView.ClearScanProgress() // Clear security scan progress
		if msg.err != nil {
			m.setError(fmt.Errorf("sync failed: %w", msg.err), "sync")
		} else {
			// Reload home view with latest repository data
			m.homeView.Init(m.telemetry)

			// Update footer statistics
			stats, _ := m.db.GetStats()
			if stats != nil {
				m.homeView.SetStats(stats.TotalSkills, stats.TotalTags)
			}

			// Only refresh view if user is still on home view
			if m.currentView == ViewHome {
				m.currentView = ViewHome
			}
		}

	case batchInstallCompleteMsg:
		// Clear pending skills from onboarding flow
		m.pendingInstallSkills = nil

		// Sync install state to ensure DB matches filesystem
		m.syncInstallState()

		// Reload home view with latest data
		m.homeView.Init(m.telemetry)
		stats, _ := m.db.GetStats()
		if stats != nil {
			m.homeView.SetStats(stats.TotalSkills, stats.TotalTags)
		}

		// Complete onboarding and go to home
		m.currentView = ViewHome
		return m, m.completeOnboarding()

	case localSkillsSyncMsg:
		if msg.indexed > 0 {
			// Refresh home view stats after indexing new local skills
			m.homeView.Init(m.telemetry)
			stats, _ := m.db.GetStats()
			if stats != nil {
				m.homeView.SetStats(stats.TotalSkills, stats.TotalTags)
			}
		}
		// Sync install state after local skills are indexed
		go m.syncInstallState()

	case cwdSkillsSyncMsg:
		if msg.indexed > 0 {
			// Refresh home view stats after indexing CWD skills
			m.homeView.Init(m.telemetry)
			stats, _ := m.db.GetStats()
			if stats != nil {
				m.homeView.SetStats(stats.TotalSkills, stats.TotalTags)
			}
		}
		// Sync install state after CWD skills are indexed
		go m.syncInstallState()

	case error:
		m.setError(msg, "unknown")
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// View returns the current view as a string.
func (m *Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.quitting {
		return ""
	}

	// Render content based on current view
	var contentView string
	switch m.currentView {
	case ViewHome:
		contentView = m.homeView.View()
	case ViewSearch:
		contentView = m.searchView.View()
	case ViewSkillDetail:
		contentView = m.detailView.View()
	case ViewTag:
		contentView = m.tagView.View()
	case ViewReset:
		contentView = m.resetView.View()
	case ViewAddSource:
		contentView = m.addSourceView.View()
	case ViewHelp:
		contentView = m.helpView.View()
	case ViewSettings:
		contentView = m.settingsView.View()
	case ViewManage:
		contentView = m.manageView.View()
	case ViewOnboardingIntro:
		contentView = m.onboardingIntroView.View()
	case ViewOnboardingSetup:
		contentView = m.onboardingSetupView.View()
	case ViewOnboardingTools:
		contentView = m.onboardingToolsView.View()
	case ViewOnboardingSkills:
		contentView = m.onboardingSkillsView.View()
	default:
		contentView = "Unknown view"
	}

	// Overlay quick skill dialog if showing
	if m.showingNewSkillDialog {
		return m.newSkillDialog.CenteredView(m.width, m.height)
	}

	// Overlay location dialog if showing
	if m.showLocationDialog && m.locationDialog != nil {
		return m.locationDialog.CenteredView(m.width, m.height)
	}

	// Overlay quit confirmation dialog if showing
	if m.showingQuitConfirm {
		return m.quitConfirmDialog.CenteredView(m.width, m.height)
	}

	// Overlay manage dialog if showing
	if m.showManageDialog && m.manageDialog != nil {
		return m.manageDialog.CenteredView(m.width, m.height)
	}

	// Overlay confirm changes dialog if showing
	if m.showConfirmChanges && m.confirmChangesDialog != nil {
		return m.confirmChangesDialog.CenteredView(m.width, m.height)
	}

	return contentView
}

// tickCmd returns a command that sends a tick message.
func (m *Model) tickCmd() tea.Cmd {
	return func() tea.Msg {
		<-m.tickChan
		return tickMsg{}
	}
}

// finishResetWithNewDB reinitializes views after reset with the new database.
// The database is already created by the async reset operation.
func (m *Model) finishResetWithNewDB() {
	// Reinitialize installer with new database (critical for post-reset installs)
	m.installer = installer.New(m.db, m.cfg)

	// Reinitialize install service with new database
	m.installService = installer.NewInstallService(m.db, m.cfg, m.telemetry)

	// Reinitialize search service with new database
	m.searchSvc = search.New(m.db, nil, search.DefaultConfig())

	// Reinitialize ALL views with the new database connection
	m.homeView = views.NewHomeView(m.db, m.cfg)
	m.searchView = views.NewSearchView(m.db, m.cfg, m.searchSvc)
	m.detailView = views.NewDetailView(m.db, m.cfg, m.favorites)
	m.tagView = views.NewTagView(m.db, m.cfg)
	m.resetView = views.NewResetView(m.db, m.cfg)
	m.addSourceView = views.NewAddSourceView(m.db, m.cfg)
	m.helpView = views.NewHelpView(m.db, m.cfg)
	m.settingsView = views.NewSettingsView(m.db, m.cfg)
	m.manageView = views.NewManageView(m.db, m.cfg, m.installService, m.telemetry)

	// Set header and footer for home view (empty DB, so 0 counts)
	m.homeView.SetStats(0, 1) // 1 tag = "mine" tag
	m.homeView.Init(m.telemetry)
	m.searchView.Init(m.telemetry)
	m.detailView.Init(m.telemetry)
	m.helpView.Init(m.telemetry)

	// Set size on ALL views after recreating them
	contentHeight := m.height - 4
	if contentHeight < 5 {
		contentHeight = 5
	}
	m.homeView.SetSize(m.width, contentHeight)
	m.searchView.SetSize(m.width, contentHeight)
	m.resetView.SetSize(m.width, contentHeight)
	m.detailView.SetSize(m.width, contentHeight)
	m.tagView.SetSize(m.width, contentHeight)
	m.addSourceView.SetSize(m.width, contentHeight)
	m.helpView.SetSize(m.width, contentHeight)
	m.settingsView.SetSize(m.width, contentHeight)
	m.manageView.SetSize(m.width, contentHeight)
}

// loadDataCmd returns a command that loads initial data.
func (m *Model) loadDataCmd() tea.Cmd {
	return func() tea.Msg {
		return skillsLoadedMsg{}
	}
}

// syncCmd returns a command that syncs with seed repositories, local skills, and CWD skills.
func (m *Model) syncCmd(githubToken string) tea.Cmd {
	return func() tea.Msg {
		// Create scraper with configuration
		cfg := scraper.ScraperConfig{
			Token:        githubToken,
			DataDir:      m.cfg.BaseDir,
			RepoCacheTTL: m.cfg.GitHub.RepoCacheTTL,
			UseGitClone:  m.cfg.GitHub.UseGitClone,
		}
		s := scraper.NewScraperWithConfig(cfg, m.db)

		// Set timeout for syncing
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// Set up progress callback that sends to channel
		opts := scraper.ScrapeSeedsOptions{
			Force:          false,
			MaxConcurrency: 5,
			OnProgress: func(completed, total int, repoName string) {
				// Non-blocking send to progress channel
				select {
				case m.pullProgressCh <- pullProgressMsg{
					completed: completed,
					total:     total,
					repoName:  repoName,
				}:
				default:
					// Channel full, skip this update
				}
			},
		}

		// Scrape all seeds in parallel with progress reporting
		result, err := s.ScrapeSeedsWithOptions(ctx, opts)
		if err != nil {
			return pullCompleteMsg{err: fmt.Errorf("failed to sync seeds: %w", err)}
		}
		totalSkillsNew := result.SkillsNew

		// Scan pending skills (newly scraped skills have PENDING status)
		pendingSkills, _ := m.db.GetPendingSkills()
		if len(pendingSkills) > 0 {
			scanner := security.NewScanner()
			for i := range pendingSkills {
				// Report scan progress
				select {
				case m.scanProgressCh <- scanProgressMsg{
					scanned:  i + 1,
					total:    len(pendingSkills),
					repoName: pendingSkills[i].Slug,
				}:
				default:
				}

				scanner.ScanAndClassify(&pendingSkills[i])
				_ = m.db.UpdateSkillSecurity(&pendingSkills[i])
			}
		}

		// Sync local skills from ~/.skulto/skills
		localSynced := m.syncLocalSkillsInternal()

		// Sync CWD skills from ./.skulto/skills
		cwdSynced := m.syncCwdSkillsInternal()

		return pullCompleteMsg{
			skillsFound: totalSkillsNew,
			skillsNew:   totalSkillsNew,
			localSynced: localSynced,
			cwdSynced:   cwdSynced,
			err:         nil,
		}
	}
}

// syncLocalSkillsInternal syncs local skills from ~/.skulto/skills and returns the count.
func (m *Model) syncLocalSkillsInternal() int {
	localSkills, err := skillgen.ScanSkills()
	if err != nil || len(localSkills) == 0 {
		return 0
	}

	parser := scraper.NewSkillParser()
	indexed := 0

	for _, skillInfo := range localSkills {
		skillID := "local-" + skillInfo.Slug

		existing, _ := m.db.GetSkill(skillID)
		if existing != nil {
			if !skillInfo.ModTime.After(existing.UpdatedAt) {
				continue // Up to date
			}
		}

		content, err := os.ReadFile(skillInfo.Path)
		if err != nil {
			continue
		}

		source := &scraper.SkillFile{
			ID:       skillID,
			Path:     skillInfo.Path,
			RepoName: "local",
		}

		skill, err := parser.Parse(string(content), source)
		if err != nil {
			continue
		}

		skill.IsLocal = true
		skill.IsInstalled = false // Not installed until user explicitly installs to AI tools
		skill.SourceID = nil

		if existing != nil {
			_ = m.db.HardDeleteSkill(skillID)
		}

		if err := m.db.CreateSkill(skill); err != nil {
			continue
		}

		_ = m.db.RecordSkillView(skill.ID)
		indexed++
	}

	return indexed
}

// syncCwdSkillsInternal syncs CWD skills from ./.skulto/skills and returns the count.
// Supports both flat (name/skill.md) and nested (category/name/skill.md) structures.
func (m *Model) syncCwdSkillsInternal() int {
	cwdSkills, err := skillgen.ScanCwdSkillsWithCategory()
	if err != nil || len(cwdSkills) == 0 {
		return 0
	}

	parser := scraper.NewSkillParser()
	mineTag := models.MineTag()
	indexed := 0

	for _, skillInfo := range cwdSkills {
		skillID := "cwd-" + skillInfo.Slug

		existing, _ := m.db.GetSkill(skillID)
		if existing != nil {
			if !skillInfo.ModTime.After(existing.UpdatedAt) {
				continue // Up to date
			}
		}

		content, err := os.ReadFile(skillInfo.Path)
		if err != nil {
			continue
		}

		source := &scraper.SkillFile{
			ID:       skillID,
			Path:     skillInfo.Path,
			RepoName: "cwd",
		}

		skill, err := parser.Parse(string(content), source)
		if err != nil {
			continue
		}

		skill.IsLocal = true
		skill.IsInstalled = false // Not installed until user explicitly installs to AI tools
		skill.SourceID = nil

		// If skill came from a nested folder, use that as the category
		if skillInfo.Category != "" {
			skill.Category = skillInfo.Category
		}

		tags := scraper.ExtractTagsWithContext(skill.Title, skill.Description, string(content))
		tags = append([]models.Tag{mineTag}, tags...)

		// Add category as a tag if present
		if skillInfo.Category != "" {
			categoryTag := models.Tag{
				ID:       strings.ToLower(skillInfo.Category),
				Name:     skillInfo.Category,
				Slug:     strings.ToLower(skillInfo.Category),
				Category: "domain",
			}
			tags = append(tags, categoryTag)
		}

		if existing != nil {
			_ = m.db.HardDeleteSkill(skillID)
		}

		if err := m.db.UpsertSkillWithTags(skill, tags); err != nil {
			continue
		}

		_ = m.db.RecordSkillView(skill.ID)
		indexed++
	}

	return indexed
}

// setError sets the error and tracks it for telemetry.
func (m *Model) setError(err error, errorType string) {
	m.err = err
	m.telemetry.TrackErrorDisplayed(errorType, m.currentView.String())
}

// trackSessionExit tracks session summary and app exit.
func (m *Model) trackSessionExit() {
	durationMs := time.Since(m.sessionStart).Milliseconds()
	m.telemetry.TrackSessionSummary(
		durationMs,
		m.viewsVisited,
		m.searchesPerformed,
		m.skillsInstalled,
		m.skillsUninstalled,
		m.reposAdded,
		m.reposRemoved,
	)
	m.telemetry.TrackAppExited("tui", durationMs, 0)
	m.telemetry.Close()
}

// completeOnboarding marks onboarding as complete.
func (m *Model) completeOnboarding() tea.Cmd {
	return m.completeOnboardingWithSkip(false, 3)
}

func (m *Model) completeOnboardingWithSkip(skipped bool, stepsViewed int) tea.Cmd {
	// Mark onboarding as completed in database
	if err := m.db.CompleteOnboarding(); err != nil {
		m.setError(fmt.Errorf("failed to save onboarding state: %w", err), "database")
		return nil
	}

	// Track onboarding completion
	m.telemetry.TrackOnboardingCompleted(stepsViewed, skipped)

	// Trigger auto-pull of seed repositories
	m.homeView.SetPulling(true)

	// Return the sync command to actually start pulling
	return tea.Batch(m.syncCmd(m.cfg.GitHub.Token), m.watchPullProgressCmd(), m.watchScanProgressCmd())
}

// startPrimarySkillsFetchCmd returns a command to fetch primary skills asynchronously.
func (m *Model) startPrimarySkillsFetchCmd() tea.Cmd {
	return func() tea.Msg {
		skills, err := m.fetchPrimarySkills()
		return views.PrimarySkillsFetchedMsg{Skills: skills, Err: err}
	}
}

// fetchPrimarySkills fetches skills from the primary repository.
// Uses git clone mode to ensure skill files are available for installation.
func (m *Model) fetchPrimarySkills() ([]models.Skill, error) {
	repo := scraper.PrimarySkillsRepo

	// Must use git clone mode so skill files are available for symlink-based installation
	s := scraper.NewScraperWithConfig(scraper.ScraperConfig{
		Token:        m.cfg.GitHub.Token,
		DataDir:      m.cfg.BaseDir,
		RepoCacheTTL: m.cfg.GitHub.RepoCacheTTL,
		UseGitClone:  true,
	}, m.db)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	_, err := s.ScrapeRepository(ctx, repo.Owner, repo.Repo)
	if err != nil {
		return nil, err
	}

	sourceID := fmt.Sprintf("%s/%s", repo.Owner, repo.Repo)
	return m.db.GetSkillsBySourceID(sourceID)
}

// primarySyncCompleteMsg is sent when the primary repo sync completes.
type primarySyncCompleteMsg struct{}

// syncPrimarySkillsCmd syncs skills from the primary repo in the background.
// This runs silently - errors are logged but don't affect the UI.
// Uses git clone mode to ensure skill files are available for installation.
func (m *Model) syncPrimarySkillsCmd() tea.Cmd {
	return func() tea.Msg {
		repo := scraper.PrimarySkillsRepo

		// Must use git clone mode so skill files are available for symlink-based installation
		s := scraper.NewScraperWithConfig(scraper.ScraperConfig{
			Token:        m.cfg.GitHub.Token,
			DataDir:      m.cfg.BaseDir,
			RepoCacheTTL: m.cfg.GitHub.RepoCacheTTL,
			UseGitClone:  true,
		}, m.db)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		_, err := s.ScrapeRepository(ctx, repo.Owner, repo.Repo)
		if err != nil {
			// Silent failure for background sync - don't disrupt UI
			return primarySyncCompleteMsg{}
		}

		// Skills are auto-saved to DB by scraper
		return primarySyncCompleteMsg{}
	}
}

// manageChangesCompleteMsg is sent when manage changes complete.
type manageChangesCompleteMsg struct {
	err error
}

// executeManageChangesCmd executes install/uninstall changes from the manage dialog.
func (m *Model) executeManageChangesCmd(skillSlug string, toInstall, toUninstall []installer.InstallLocation) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Execute uninstalls first
		if len(toUninstall) > 0 {
			if err := m.installService.Uninstall(ctx, skillSlug, toUninstall); err != nil {
				return manageChangesCompleteMsg{err: fmt.Errorf("uninstall failed: %w", err)}
			}
		}

		// Execute installs - one location at a time to avoid cross-product
		for _, loc := range toInstall {
			opts := installer.InstallOptions{
				Platforms: []string{string(loc.Platform)},
				Scopes:    []installer.InstallScope{loc.Scope},
				Confirm:   true,
			}

			if _, err := m.installService.Install(ctx, skillSlug, opts); err != nil {
				return manageChangesCompleteMsg{err: fmt.Errorf("install failed: %w", err)}
			}
		}

		return manageChangesCompleteMsg{err: nil}
	}
}

// batchInstallCompleteMsg is sent when batch installation from onboarding completes.
type batchInstallCompleteMsg struct {
	installed int
	failed    int
}

// installBatchSkillsCmd installs multiple skills from the onboarding flow.
func (m *Model) installBatchSkillsCmd(skills []models.Skill, locations []installer.InstallLocation) tea.Cmd {
	return func() tea.Msg {
		// Get source for primary repo skills
		sourceID := fmt.Sprintf("%s/%s", scraper.PrimarySkillsRepo.Owner, scraper.PrimarySkillsRepo.Repo)
		source, err := m.db.GetSource(sourceID)
		if err != nil || source == nil {
			return batchInstallCompleteMsg{installed: 0, failed: len(skills)}
		}

		ctx := context.Background()
		installed := 0
		failed := 0

		for _, skill := range skills {
			skillCopy := skill // avoid closure issue
			if err := m.installer.InstallTo(ctx, &skillCopy, source, locations); err != nil {
				failed++
				continue
			}
			installed++
		}

		return batchInstallCompleteMsg{installed: installed, failed: failed}
	}
}

// syncLocalSkillsCmd returns a command that scans ~/.skulto/skills and indexes any skills
// that exist on disk but are not in the database. This ensures local skills are always searchable.
func (m *Model) syncLocalSkillsCmd() tea.Cmd {
	return func() tea.Msg {
		// Debug log file
		debugLog := func(format string, args ...interface{}) {
			if os.Getenv("SKULTO_DEBUG") != "1" {
				return
			}
			f, err := os.OpenFile("/tmp/skulto-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return
			}
			defer func() { _ = f.Close() }()
			_, _ = fmt.Fprintf(f, "[sync] "+format+"\n", args...)
		}

		debugLog("Starting local skills sync")

		// Scan the local skills directory
		localSkills, err := skillgen.ScanSkills()
		if err != nil {
			debugLog("Error scanning skills: %v", err)
			return localSkillsSyncMsg{err: err}
		}

		debugLog("Found %d local skills on disk", len(localSkills))

		if len(localSkills) == 0 {
			return localSkillsSyncMsg{indexed: 0}
		}

		parser := scraper.NewSkillParser()
		indexed := 0

		for _, skillInfo := range localSkills {
			skillID := "local-" + skillInfo.Slug
			debugLog("Processing skill: %s (ID: %s)", skillInfo.Slug, skillID)

			// Check if skill already exists in database
			existing, _ := m.db.GetSkill(skillID)
			if existing != nil {
				debugLog("Skill exists in DB, checking mod time. File: %v, DB: %v", skillInfo.ModTime, existing.UpdatedAt)
				// Skill exists, check if file is newer
				if skillInfo.ModTime.After(existing.UpdatedAt) {
					debugLog("File is newer, will re-index")
					// File is newer, re-index it (fall through)
				} else {
					debugLog("Skill up to date, skipping")
					continue
				}
			} else {
				debugLog("Skill not in DB, will create")
			}

			// Read the skill file content
			content, err := os.ReadFile(skillInfo.Path)
			if err != nil {
				debugLog("Error reading file %s: %v", skillInfo.Path, err)
				continue
			}
			debugLog("Read %d bytes from %s", len(content), skillInfo.Path)

			// Create a source descriptor for the parser
			source := &scraper.SkillFile{
				ID:       skillID,
				Path:     skillInfo.Path,
				RepoName: "local",
			}

			// Parse the skill file
			skill, err := parser.Parse(string(content), source)
			if err != nil {
				debugLog("Error parsing skill: %v", err)
				continue
			}
			debugLog("Parsed skill: Title=%q, ID=%s", skill.Title, skill.ID)

			// Mark as local skill (no source ID to avoid foreign key constraint)
			skill.IsLocal = true
			skill.IsInstalled = false // Not installed until user explicitly installs to AI tools
			skill.SourceID = nil      // Local skills have no source

			// Hard delete existing and create fresh to ensure FTS triggers fire correctly
			if existing != nil {
				debugLog("Hard deleting existing skill to re-create")
				_ = m.db.HardDeleteSkill(skillID)
			}

			// Create the skill
			if err := m.db.CreateSkill(skill); err != nil {
				debugLog("Error creating skill: %v", err)
				continue
			}
			debugLog("Successfully created skill in DB")

			// Record as recently viewed
			if err := m.db.RecordSkillView(skill.ID); err != nil {
				debugLog("Error recording view: %v", err)
			} else {
				debugLog("Recorded as recently viewed")
			}

			indexed++
		}

		debugLog("Sync complete, indexed %d skills", indexed)
		return localSkillsSyncMsg{indexed: indexed}
	}
}

// syncCwdSkillsCmd returns a command that scans .skulto/skills in the current
// working directory and indexes skills with the "mine" tag.
// Supports both flat (name/skill.md) and nested (category/name/skill.md) structures.
func (m *Model) syncCwdSkillsCmd() tea.Cmd {
	return func() tea.Msg {
		// Always log to file for debugging - users can check /tmp/skulto-cwd-sync.log
		debugLog := func(format string, args ...any) {
			f, _ := os.OpenFile("/tmp/skulto-cwd-sync.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if f != nil {
				defer func() { _ = f.Close() }()
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				_, _ = fmt.Fprintf(f, "[%s] [cwd-sync] "+format+"\n", append([]any{timestamp}, args...)...)
			}
		}

		cwd, _ := os.Getwd()
		debugLog("Starting CWD skills sync from directory: %s", cwd)

		// Scan the cwd skills directory with category support
		cwdSkills, err := skillgen.ScanCwdSkillsWithCategory()
		if err != nil {
			debugLog("Error scanning CWD skills: %v", err)
			return cwdSkillsSyncMsg{err: err}
		}

		if len(cwdSkills) == 0 {
			debugLog("No CWD skills found")
			return cwdSkillsSyncMsg{indexed: 0}
		}

		debugLog("Found %d CWD skills on disk", len(cwdSkills))

		parser := scraper.NewSkillParser()
		mineTag := models.MineTag()
		indexed := 0

		for _, skillInfo := range cwdSkills {
			// Use "cwd-" prefix to distinguish from global local skills
			skillID := "cwd-" + skillInfo.Slug
			debugLog("Processing CWD skill: %s (category: %s, ID: %s)", skillInfo.Slug, skillInfo.Category, skillID)

			// Check if skill already exists
			existing, _ := m.db.GetSkill(skillID)
			if existing != nil {
				if skillInfo.ModTime.After(existing.UpdatedAt) {
					debugLog("File is newer, will re-index")
				} else {
					debugLog("Skill up to date, skipping")
					continue
				}
			}

			// Read the skill file
			content, err := os.ReadFile(skillInfo.Path)
			if err != nil {
				debugLog("Error reading file: %v", err)
				continue
			}

			// Parse the skill
			source := &scraper.SkillFile{
				ID:       skillID,
				Path:     skillInfo.Path,
				RepoName: "cwd",
			}
			skill, err := parser.Parse(string(content), source)
			if err != nil {
				debugLog("Error parsing skill: %v", err)
				continue
			}

			// Mark as local CWD skill (no source ID to avoid foreign key constraint)
			skill.IsLocal = true
			skill.IsInstalled = false // Not installed until user explicitly installs to AI tools
			skill.SourceID = nil      // CWD skills have no source

			// If skill came from a nested folder, use that as the category
			if skillInfo.Category != "" {
				skill.Category = skillInfo.Category
			}

			// Extract tags from content and add "mine" tag
			tags := scraper.ExtractTagsWithContext(skill.Title, skill.Description, string(content))
			tags = append([]models.Tag{mineTag}, tags...) // Prepend mine tag

			// Add category as a tag if present
			if skillInfo.Category != "" {
				categoryTag := models.Tag{
					ID:       strings.ToLower(skillInfo.Category),
					Name:     skillInfo.Category,
					Slug:     strings.ToLower(skillInfo.Category),
					Category: "domain",
				}
				tags = append(tags, categoryTag)
			}

			// Hard delete if exists, then create fresh
			if existing != nil {
				_ = m.db.HardDeleteSkill(skillID)
			}

			// Create skill with tags (including "mine")
			if err := m.db.UpsertSkillWithTags(skill, tags); err != nil {
				debugLog("Error creating skill: %v", err)
				continue
			}

			debugLog("Successfully indexed CWD skill: %s", skill.Title)

			// Record as recently viewed
			_ = m.db.RecordSkillView(skill.ID)
			indexed++
		}

		debugLog("CWD sync complete, indexed %d skills", indexed)
		return cwdSkillsSyncMsg{indexed: indexed}
	}
}

// addSourceCmd returns a command that adds a new source repository and syncs it.
func (m *Model) addSourceCmd(repoURL string) tea.Cmd {
	return func() tea.Msg {
		// Parse and validate the repository URL
		source, err := scraper.ParseRepositoryURL(repoURL)
		if err != nil {
			return fmt.Errorf("invalid repository URL: %w", err)
		}

		// Check if source already exists
		existing, err := m.db.GetSource(source.ID)
		if err != nil {
			return fmt.Errorf("failed to check existing source: %w", err)
		}
		if existing != nil {
			return fmt.Errorf("source %s already exists", source.ID)
		}

		// Add source to database
		if err := m.db.UpsertSource(source); err != nil {
			return fmt.Errorf("failed to add source: %w", err)
		}

		// Create scraper with configuration
		cfg := scraper.ScraperConfig{
			Token:        m.cfg.GitHub.Token,
			DataDir:      m.cfg.BaseDir,
			RepoCacheTTL: m.cfg.GitHub.RepoCacheTTL,
			UseGitClone:  m.cfg.GitHub.UseGitClone,
		}
		s := scraper.NewScraperWithConfig(cfg, m.db)

		// Set timeout for syncing this repository
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Scrape the new repository
		result, err := s.ScrapeRepository(ctx, source.Owner, source.Repo)
		if err != nil {
			return pullCompleteMsg{err: fmt.Errorf("failed to sync %s: %w", source.ID, err)}
		}

		// Track repo added
		m.telemetry.TrackRepoAdded(source.ID, result.SkillsNew)
		m.reposAdded++

		return pullCompleteMsg{
			skillsFound: result.SkillsNew,
			skillsNew:   result.SkillsNew,
			err:         nil,
		}
	}
}

// installCmd performs async skill installation or uninstallation.
func (m *Model) installCmd(skill *models.Skill) tea.Cmd {
	return func() tea.Msg {
		var err error

		if skill.IsInstalled {
			// Installing
			err = m.installer.Install(context.Background(), skill, skill.Source)
			if err == nil {
				m.telemetry.TrackSkillInstalled(skill.Title, skill.Category, skill.IsLocal, 1)
				m.skillsInstalled++
			}
		} else {
			// Uninstalling - use UninstallAll to handle both new and legacy installations
			err = m.installer.UninstallAll(context.Background(), skill)
			if err == nil {
				m.telemetry.TrackSkillUninstalled(skill.Title, skill.Category, skill.IsLocal)
				m.skillsUninstalled++
			}
		}

		return views.SkillInstalledMsg{
			Success: err == nil,
			Err:     err,
		}
	}
}

// installToLocationsCmd performs async skill installation to specific locations.
func (m *Model) installToLocationsCmd(skill *models.Skill, source *models.Source, locations []installer.InstallLocation) tea.Cmd {
	return func() tea.Msg {
		err := m.installer.InstallTo(context.Background(), skill, source, locations)
		return views.SkillInstalledMsg{
			Success: err == nil,
			Err:     err,
		}
	}
}

// scanSkillCmd returns a command that scans a skill and updates the database.
func (m *Model) scanSkillCmd(skillID string) tea.Cmd {
	return func() tea.Msg {
		// Get skill from database
		skill, err := m.db.GetSkill(skillID)
		if err != nil || skill == nil {
			return views.SkillScanCompleteMsg{SkillID: skillID, Err: err}
		}

		// Create scanner and scan
		scanner := security.NewScanner()
		scanner.ScanAndClassify(skill)

		// Save to database
		if err := m.db.UpdateSkillSecurity(skill); err != nil {
			return views.SkillScanCompleteMsg{SkillID: skillID, Err: err}
		}

		return views.SkillScanCompleteMsg{SkillID: skillID, Err: nil}
	}
}

// installLocalSkillCmd installs a local skill to the specified locations.
func (m *Model) installLocalSkillCmd(skillInfo skillgen.SkillInfo, locations []installer.InstallLocation) tea.Cmd {
	return func() tea.Msg {
		// Get the skill from database
		skillID := "local-" + skillInfo.Slug
		skill, err := m.db.GetSkill(skillID)
		if err != nil || skill == nil {
			return views.NewSkillInstallCompleteMsg{
				Success: false,
				Err:     fmt.Errorf("skill not found in database: %s", skillID),
			}
		}

		// Get the source directory (parent of skill.md)
		sourcePath := filepath.Dir(skillInfo.Path)

		// Install using the local skill method
		err = m.installer.InstallLocalSkillTo(context.Background(), skill, sourcePath, locations)
		if err != nil {
			return views.NewSkillInstallCompleteMsg{
				Success: false,
				Err:     err,
			}
		}

		return views.NewSkillInstallCompleteMsg{
			Success: true,
			Err:     nil,
		}
	}
}

// installLocalSkillFromDetailCmd installs a local skill from the detail view to the specified locations.
func (m *Model) installLocalSkillFromDetailCmd(skill *models.Skill, locations []installer.InstallLocation) tea.Cmd {
	return func() tea.Msg {
		if skill == nil {
			return views.SkillInstalledMsg{
				Success: false,
				Err:     fmt.Errorf("no skill provided"),
			}
		}

		// Get the source directory from the skill's FilePath
		sourcePath := filepath.Dir(skill.FilePath)
		if sourcePath == "" {
			return views.SkillInstalledMsg{
				Success: false,
				Err:     fmt.Errorf("skill has no file path"),
			}
		}

		// Install using the local skill method
		err := m.installer.InstallLocalSkillTo(context.Background(), skill, sourcePath, locations)
		if err != nil {
			return views.SkillInstalledMsg{
				Success: false,
				Err:     err,
			}
		}

		return views.SkillInstalledMsg{
			Success: true,
			Err:     nil,
		}
	}
}

// parsePlatformsFromState converts UserState.AITools to []installer.Platform.
func parsePlatformsFromState(state *models.UserState) []installer.Platform {
	if state == nil || len(state.GetAITools()) == 0 {
		return nil
	}

	toolNames := state.GetAITools()
	platforms := make([]installer.Platform, 0, len(toolNames))

	for _, name := range toolNames {
		name = strings.TrimSpace(name)
		if p := installer.PlatformFromString(name); p != "" {
			platforms = append(platforms, p)
		}
	}

	return platforms
}

// saveNewSkillCmd returns a command to save the quick-generated skill.
// Note: Legacy skill saving via skillbuilder has been removed.
// Skills are now saved directly to ~/.skulto/skills by Claude.
func (m *Model) saveNewSkillCmd() tea.Cmd {
	return func() tea.Msg {
		return views.NewSkillSavedMsg{Err: fmt.Errorf("skill saving disabled - use Claude to save skills directly")}
	}
}

// Run executes the TUI program.
func Run(database *db.DB, conf *config.Config, tc telemetry.Client) error {
	return RunWithIndexer(database, conf, nil, tc)
}

// RunWithIndexer executes the TUI program with an optional background indexer.
func RunWithIndexer(database *db.DB, conf *config.Config, indexer *search.BackgroundIndexer, tc telemetry.Client) error {
	model := NewModelWithIndexer(database, conf, indexer, tc)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Clean up background indexer when done
	defer func() {
		if indexer != nil {
			_ = indexer.Close()
		}
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
