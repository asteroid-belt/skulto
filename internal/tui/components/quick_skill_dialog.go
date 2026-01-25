package components

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/skillgen"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NewSkillDialogState represents the dialog's current state.
type NewSkillDialogState int

const (
	// StateInput is the initial state where user enters prompt.
	StateInput NewSkillDialogState = iota
	// StateLaunching shows instructions before launching interactive CLI.
	StateLaunching
	// StateGenerating is when CLI is running.
	StateGenerating
	// StatePreview is when generation is complete (legacy, kept for compatibility).
	StatePreview
	// StateError is when an error occurred.
	StateError
	// StateResult shows the result after Claude session ends.
	StateResult
	// StateInstall shows install location dialog after successful creation.
	StateInstall
)

// NewSkillDialog is a popup for quick skill generation.
type NewSkillDialog struct {
	// Input state
	promptInput  textarea.Model
	selectedTool skillgen.AITool
	toolOptions  []skillgen.AITool
	toolIndex    int
	focusedField int // 0 = prompt, 1 = tool selector, 2 = generate button

	// Generation state
	executor         *skillgen.CLIExecutor
	streamingOutput  string
	generatedContent string
	state            NewSkillDialogState
	err              error
	cancelFunc       context.CancelFunc
	skillsBefore     []skillgen.SkillInfo // snapshot before launching Claude
	newSkills        []skillgen.SkillInfo // skills created/modified during session

	// Preview state
	previewScroll int
	previewLines  []string

	// Install state
	installDialog    *InstallLocationDialog
	installSkillInfo *skillgen.SkillInfo // The skill being installed
	installSuccess   bool                // Whether installation succeeded

	// UI state
	width       int
	height      int
	cancelled   bool
	confirmed   bool
	animTick    int
	launching   bool      // true when about to launch interactive Claude
	preparedCmd *exec.Cmd // command prepared for launch in StateLaunching
}

// NewNewSkillDialog creates a new quick skill dialog.
func NewNewSkillDialog() *NewSkillDialog {
	ta := textarea.New()
	ta.Placeholder = "Describe the skill you want to create..."
	ta.CharLimit = 2000
	ta.SetWidth(50)
	ta.SetHeight(5)
	ta.ShowLineNumbers = false

	// Style textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F1C40F")).
		Padding(0, 1)
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6B6B6B")).
		Padding(0, 1)

	executor := skillgen.NewCLIExecutor()
	allTools := executor.AvailableTools()

	// Filter to only supported tools (exclude opencode for now)
	var available []skillgen.AITool
	for _, tool := range allTools {
		if tool == skillgen.AIToolClaude || tool == skillgen.AIToolCodex {
			available = append(available, tool)
		}
	}

	// Default to claude if available
	selectedTool := skillgen.AIToolClaude
	if len(available) > 0 {
		selectedTool = available[0]
	}

	return &NewSkillDialog{
		promptInput:  ta,
		executor:     executor,
		toolOptions:  available,
		selectedTool: selectedTool,
		state:        StateInput,
		focusedField: 0,
	}
}

// SetSize sets the dialog dimensions.
func (d *NewSkillDialog) SetSize(w, h int) {
	d.width = w
	d.height = h

	// Adjust textarea width (minimum 30, maximum 60)
	inputWidth := max(30, min(60, w-10))
	d.promptInput.SetWidth(inputWidth)
}

// SetPlatforms sets the platforms for the install dialog.
// Call this before showing the dialog if platforms might have changed.
func (d *NewSkillDialog) SetPlatforms(platforms []installer.Platform) {
	if len(platforms) > 0 {
		d.installDialog = NewInstallLocationDialog(platforms)
	}
}

// Update handles keyboard input.
func (d *NewSkillDialog) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return d.handleKeyMsg(msg)
	case NewSkillClaudeFinishedMsg:
		return d.handleClaudeFinished(msg)
	case NewSkillStreamMsg:
		return d.handleStreamMsg(msg)
	case NewSkillTickMsg:
		d.animTick++
		if d.state == StateGenerating {
			return d.tickCmd()
		}
		return nil
	}
	return nil
}

func (d *NewSkillDialog) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch d.state {
	case StateInput:
		return d.handleInputState(msg, key)
	case StateLaunching:
		return d.handleLaunchingState(key)
	case StateGenerating:
		return d.handleGeneratingState(key)
	case StatePreview:
		return d.handlePreviewState(key)
	case StateError:
		return d.handleErrorState(key)
	case StateResult:
		return d.handleResultState(key)
	case StateInstall:
		return d.handleInstallState(msg)
	}
	return nil
}

func (d *NewSkillDialog) handleInputState(msg tea.KeyMsg, key string) tea.Cmd {
	// Handle escape from anywhere
	if key == "esc" {
		d.cancelled = true
		return nil
	}

	// Handle tab navigation from anywhere
	if key == "tab" {
		d.focusedField = (d.focusedField + 1) % 3
		if d.focusedField == 0 {
			d.promptInput.Focus()
		} else {
			d.promptInput.Blur()
		}
		return nil
	}
	if key == "shift+tab" {
		d.focusedField = (d.focusedField + 2) % 3
		if d.focusedField == 0 {
			d.promptInput.Focus()
		} else {
			d.promptInput.Blur()
		}
		return nil
	}

	// When prompt textarea is focused, pass all other keys to it
	// This allows free typing including h, j, k, l and enter for newlines
	if d.focusedField == 0 {
		var cmd tea.Cmd
		d.promptInput, cmd = d.promptInput.Update(msg)
		return cmd
	}

	// Handle tool selector (focusedField == 1)
	if d.focusedField == 1 {
		switch key {
		case "left", "h":
			if len(d.toolOptions) > 0 {
				d.toolIndex = (d.toolIndex + len(d.toolOptions) - 1) % len(d.toolOptions)
				d.selectedTool = d.toolOptions[d.toolIndex]
			}
			return nil
		case "right", "l":
			if len(d.toolOptions) > 0 {
				d.toolIndex = (d.toolIndex + 1) % len(d.toolOptions)
				d.selectedTool = d.toolOptions[d.toolIndex]
			}
			return nil
		}
	}

	// Handle generate button (focusedField == 2)
	if d.focusedField == 2 && key == "enter" {
		return d.startGeneration()
	}

	return nil
}

func (d *NewSkillDialog) handleLaunchingState(key string) tea.Cmd {
	switch key {
	case "enter":
		// User confirmed, launch the interactive CLI
		return d.launchPreparedCmd()
	case "esc":
		// User cancelled, go back to input
		d.state = StateInput
		d.preparedCmd = nil
		d.launching = false
		return nil
	}
	return nil
}

func (d *NewSkillDialog) handleGeneratingState(key string) tea.Cmd {
	if key == "esc" {
		// Cancel context which will terminate the running command
		if d.cancelFunc != nil {
			d.cancelFunc()
			d.cancelFunc = nil
		}
		d.state = StateInput
		d.streamingOutput = ""
	}
	return nil
}

func (d *NewSkillDialog) handlePreviewState(key string) tea.Cmd {
	switch key {
	case "esc":
		d.state = StateInput
		d.generatedContent = ""
		d.previewScroll = 0
		return nil
	case "up", "k":
		if d.previewScroll > 0 {
			d.previewScroll--
		}
		return nil
	case "down", "j":
		maxScroll := max(0, len(d.previewLines)-10)
		if d.previewScroll < maxScroll {
			d.previewScroll++
		}
		return nil
	case "ctrl+s", "enter":
		d.confirmed = true
		return nil
	}
	return nil
}

func (d *NewSkillDialog) handleErrorState(key string) tea.Cmd {
	if key == "esc" || key == "enter" {
		d.state = StateInput
		d.err = nil
	}
	return nil
}

func (d *NewSkillDialog) handleResultState(key string) tea.Cmd {
	switch key {
	case "enter", "i":
		// If skills were created, transition to install state
		if len(d.newSkills) > 0 && d.installDialog != nil {
			d.installSkillInfo = &d.newSkills[0] // Install first skill (most common case)
			d.installDialog.Reset()
			d.state = StateInstall
			return nil
		}
		// No skills or no dialog, just close
		d.confirmed = true
		return nil
	case "l", "esc", "q":
		// Install Later / Cancel - close without installing
		d.confirmed = true
		return nil
	case "r":
		// Try again - go back to input
		d.state = StateInput
		d.newSkills = nil
		d.skillsBefore = nil
		return nil
	}
	return nil
}

func (d *NewSkillDialog) handleInstallState(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	// Handle Install Later button (L key)
	if key == "l" {
		d.confirmed = true
		return nil
	}

	// Pass key to install dialog
	d.installDialog.Update(msg)

	// Check dialog state after update
	if d.installDialog.IsConfirmed() {
		// User confirmed - mark for installation
		d.confirmed = true
		d.installSuccess = true
		return nil
	}

	if d.installDialog.IsCancelled() {
		// User pressed Esc - go back to result state
		d.state = StateResult
		d.installDialog.Reset()
		return nil
	}

	return nil
}

func (d *NewSkillDialog) startGeneration() tea.Cmd {
	prompt := strings.TrimSpace(d.promptInput.Value())
	if prompt == "" {
		d.err = fmt.Errorf("please enter a skill description")
		d.state = StateError
		return nil
	}

	if len(d.toolOptions) == 0 {
		d.err = fmt.Errorf("no AI CLI tools found. Install claude or codex")
		d.state = StateError
		return nil
	}

	// Snapshot current skills before launching Claude
	d.skillsBefore, _ = skillgen.ScanSkills()

	// Build the interactive command
	cmd, err := d.executor.InteractiveCommand(d.selectedTool, prompt)
	if err != nil {
		d.err = err
		d.state = StateError
		return nil
	}

	// Store the prepared command and transition to launching state
	// This shows the user instructions before actually launching
	d.preparedCmd = cmd
	d.state = StateLaunching
	d.launching = true

	return nil
}

// launchPreparedCmd actually launches the interactive CLI session
func (d *NewSkillDialog) launchPreparedCmd() tea.Cmd {
	if d.preparedCmd == nil {
		d.err = fmt.Errorf("no command prepared")
		d.state = StateError
		return nil
	}

	cmd := d.preparedCmd
	d.preparedCmd = nil
	d.state = StateGenerating

	// Use tea.ExecProcess to launch Claude interactively
	// This suspends the TUI and gives Claude full terminal control
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return NewSkillClaudeFinishedMsg{Err: err}
	})
}

func (d *NewSkillDialog) handleClaudeFinished(msg NewSkillClaudeFinishedMsg) tea.Cmd {
	d.launching = false

	// Scan for new/modified skills
	skillsAfter, _ := skillgen.ScanSkills()
	d.newSkills = skillgen.FindNewSkills(d.skillsBefore, skillsAfter)

	// Check if Claude exited with an error (but still check for skills)
	if msg.Err != nil && len(d.newSkills) == 0 {
		d.err = fmt.Errorf("session ended: %w", msg.Err)
		d.state = StateError
		return nil
	}

	// Show result state (success or no skills found)
	d.state = StateResult
	return nil
}

// handleStreamMsg handles streaming output (kept for potential future use)
func (d *NewSkillDialog) handleStreamMsg(msg NewSkillStreamMsg) tea.Cmd {
	if msg.Err != nil {
		d.err = msg.Err
		d.state = StateError
		d.cancelFunc = nil
		return nil
	}

	if msg.Done {
		d.generatedContent = msg.Content
		d.previewLines = strings.Split(msg.Content, "\n")
		d.previewScroll = 0
		d.state = StatePreview
		d.cancelFunc = nil
	} else {
		d.streamingOutput += msg.Content
	}

	return nil
}

func (d *NewSkillDialog) tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return NewSkillTickMsg{}
	})
}

// View renders the dialog.
func (d *NewSkillDialog) View() string {
	switch d.state {
	case StateInput:
		return d.viewInput()
	case StateLaunching:
		return d.viewLaunching()
	case StateGenerating:
		return d.viewGenerating()
	case StatePreview:
		return d.viewPreview()
	case StateError:
		return d.viewError()
	case StateResult:
		return d.viewResult()
	case StateInstall:
		return d.viewInstall()
	}
	return ""
}

func (d *NewSkillDialog) viewInput() string {
	dialogWidth := min(70, d.width-4)
	contentWidth := dialogWidth - 6

	// Colors
	accentColor := lipgloss.Color("#DC143C")
	goldColor := lipgloss.Color("#F1C40F")
	mutedColor := lipgloss.Color("#6B6B6B")
	textColor := lipgloss.Color("#E5E5E5")

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)
	title := titleStyle.Render("Quick Skill Creator")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)
	subtitle := subtitleStyle.Render("Describe the skill you want to create")

	// Prompt label
	labelStyle := lipgloss.NewStyle().
		Foreground(textColor).
		Bold(d.focusedField == 0)
	promptLabel := labelStyle.Render("Skill Description:")

	// Textarea
	d.promptInput.SetWidth(contentWidth - 4)
	promptInput := d.promptInput.View()

	// Tool selector
	toolLabel := lipgloss.NewStyle().
		Foreground(textColor).
		Bold(d.focusedField == 1).
		MarginTop(1).
		Render("AI Tool:")

	var toolOptions []string
	for i, tool := range d.toolOptions {
		style := lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1)
		if i == d.toolIndex {
			style = style.
				Foreground(goldColor).
				Bold(true).
				Background(lipgloss.Color("#1A1A2E"))
		}
		toolOptions = append(toolOptions, style.Render(string(tool)))
	}

	toolSelector := ""
	if len(toolOptions) > 0 {
		toolSelector = lipgloss.JoinHorizontal(lipgloss.Left, toolOptions...)
	} else {
		toolSelector = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Render("No AI tools found")
	}

	// Generate button
	btnStyle := lipgloss.NewStyle().
		Padding(0, 2).
		MarginTop(1)
	if d.focusedField == 2 {
		btnStyle = btnStyle.
			Background(goldColor).
			Foreground(lipgloss.Color("#000000")).
			Bold(true)
	} else {
		btnStyle = btnStyle.
			Background(mutedColor).
			Foreground(textColor)
	}
	generateBtn := btnStyle.Render("Generate")

	cancelStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(0, 2).
		MarginTop(1)
	cancelBtn := cancelStyle.Render("Cancel (Esc)")

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, generateBtn, "  ", cancelBtn)

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1)
	footer := footerStyle.Render("Tab: switch to next | Arrow/h/l: tool | Enter: generate")

	// Compose content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		promptLabel,
		promptInput,
		"",
		toolLabel,
		toolSelector,
		"",
		buttons,
		"",
		footer,
	)

	// Dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	return dialogStyle.Render(content)
}

func (d *NewSkillDialog) viewLaunching() string {
	dialogWidth := min(70, d.width-4)
	contentWidth := dialogWidth - 6

	accentColor := lipgloss.Color("#DC143C")
	goldColor := lipgloss.Color("#F1C40F")
	mutedColor := lipgloss.Color("#6B6B6B")
	textColor := lipgloss.Color("#E5E5E5")
	hintBgColor := lipgloss.Color("#1A1A2E")

	titleStyle := lipgloss.NewStyle().
		Foreground(goldColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)
	title := titleStyle.Render("Ready to Launch " + string(d.selectedTool))

	subtitleStyle := lipgloss.NewStyle().
		Foreground(textColor).
		Width(contentWidth).
		Align(lipgloss.Center)
	subtitle := subtitleStyle.Render("You're about to start an interactive AI session")

	// Instructions box
	instructBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(goldColor).
		Background(hintBgColor).
		Padding(1, 2).
		Width(contentWidth - 4).
		Align(lipgloss.Center)

	instructTitleStyle := lipgloss.NewStyle().
		Foreground(goldColor).
		Bold(true)
	instructTitle := instructTitleStyle.Render("Important: How to Exit")

	instructTextStyle := lipgloss.NewStyle().
		Foreground(textColor)
	instructLines := []string{
		"",
		instructTextStyle.Render("When you're done building your skill:"),
		lipgloss.NewStyle().Foreground(goldColor).Bold(true).Render("  Press Ctrl+C in the terminal"),
		"",
		instructTextStyle.Render("The AI will create your skill in:"),
		lipgloss.NewStyle().Foreground(mutedColor).Italic(true).Render("  ~/.skulto/skills/<name>/skill.md"),
	}

	instructContent := lipgloss.JoinVertical(lipgloss.Center, append([]string{instructTitle}, instructLines...)...)
	instructBox := instructBoxStyle.Render(instructContent)

	// Buttons
	continueStyle := lipgloss.NewStyle().
		Background(goldColor).
		Foreground(lipgloss.Color("#000000")).
		Bold(true).
		Padding(0, 2)
	continueBtn := continueStyle.Render("Press Enter to Start")

	cancelStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(0, 2)
	cancelBtn := cancelStyle.Render("Esc: Cancel")

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, continueBtn, "  ", cancelBtn)
	buttonsCentered := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Render(buttons)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		instructBox,
		"",
		buttonsCentered,
	)

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	return dialogStyle.Render(content)
}

func (d *NewSkillDialog) viewGenerating() string {
	dialogWidth := min(70, d.width-4)
	contentWidth := dialogWidth - 6

	accentColor := lipgloss.Color("#DC143C")
	goldColor := lipgloss.Color("#F1C40F")
	mutedColor := lipgloss.Color("#6B6B6B")

	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)
	title := titleStyle.Render("Building Skill with " + string(d.selectedTool))

	// Spinner animation
	frames := []string{"*", "**", "***", "****", "*****", "****", "***", "**", "*"}
	spinner := frames[d.animTick%len(frames)]

	spinnerStyle := lipgloss.NewStyle().
		Foreground(goldColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center)
	spinnerView := spinnerStyle.Render(spinner + " Interactive session running " + spinner)

	// Show streaming output if any
	outputStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(contentWidth).
		MaxHeight(6)
	outputView := ""
	if d.streamingOutput != "" {
		lines := strings.Split(d.streamingOutput, "\n")
		if len(lines) > 6 {
			lines = lines[len(lines)-6:]
		}
		outputView = outputStyle.Render(strings.Join(lines, "\n"))
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1)
	footer := footerStyle.Render("Your skill will be saved to ~/.skulto/skills/")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		spinnerView,
		"",
		outputView,
		"",
		footer,
	)

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	return dialogStyle.Render(content)
}

func (d *NewSkillDialog) viewPreview() string {
	dialogWidth := min(80, d.width-4)
	contentWidth := dialogWidth - 6

	accentColor := lipgloss.Color("#DC143C")
	goldColor := lipgloss.Color("#F1C40F")
	mutedColor := lipgloss.Color("#6B6B6B")
	successColor := lipgloss.Color("#10B981")

	titleStyle := lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)
	title := titleStyle.Render("Skill Generated")

	// Preview area
	previewHeight := min(15, d.height-15)
	startLine := d.previewScroll
	endLine := min(startLine+previewHeight, len(d.previewLines))

	visibleLines := d.previewLines[startLine:endLine]
	previewContent := strings.Join(visibleLines, "\n")

	previewStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedColor).
		Width(contentWidth).
		Height(previewHeight).
		Padding(0, 1)
	preview := previewStyle.Render(previewContent)

	// Scroll indicator
	scrollInfo := ""
	if len(d.previewLines) > previewHeight {
		scrollInfo = fmt.Sprintf("Line %d-%d of %d", startLine+1, endLine, len(d.previewLines))
	}
	scrollStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(contentWidth).
		Align(lipgloss.Right)
	scrollView := scrollStyle.Render(scrollInfo)

	// Buttons
	saveStyle := lipgloss.NewStyle().
		Background(goldColor).
		Foreground(lipgloss.Color("#000000")).
		Bold(true).
		Padding(0, 2)
	saveBtn := saveStyle.Render("Save (Ctrl+S)")

	backStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(0, 2)
	backBtn := backStyle.Render("Back (Esc)")

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, saveBtn, "  ", backBtn)

	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1)
	footer := footerStyle.Render("Up/Down or j/k: scroll | Ctrl+S: save | Esc: back")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		preview,
		scrollView,
		"",
		buttons,
		"",
		footer,
	)

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	return dialogStyle.Render(content)
}

func (d *NewSkillDialog) viewError() string {
	dialogWidth := min(60, d.width-4)
	contentWidth := dialogWidth - 6

	accentColor := lipgloss.Color("#DC143C")
	errorColor := lipgloss.Color("#FF6B6B")
	mutedColor := lipgloss.Color("#6B6B6B")

	titleStyle := lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)
	title := titleStyle.Render("Error")

	errStyle := lipgloss.NewStyle().
		Foreground(errorColor).
		Width(contentWidth).
		Align(lipgloss.Center)
	errText := "Unknown error"
	if d.err != nil {
		errText = d.err.Error()
	}
	errMsg := errStyle.Render(errText)

	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1)
	footer := footerStyle.Render("Press Enter or Esc to continue")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		errMsg,
		"",
		footer,
	)

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	return dialogStyle.Render(content)
}

func (d *NewSkillDialog) viewResult() string {
	dialogWidth := min(70, d.width-4)
	contentWidth := dialogWidth - 6

	accentColor := lipgloss.Color("#DC143C")
	successColor := lipgloss.Color("#10B981")
	warningColor := lipgloss.Color("#F59E0B")
	mutedColor := lipgloss.Color("#6B6B6B")
	textColor := lipgloss.Color("#E5E5E5")

	var title, message string
	var titleColor lipgloss.Color

	if len(d.newSkills) > 0 {
		titleColor = successColor
		title = "Skill Created & Indexed!"
		if len(d.newSkills) == 1 {
			message = fmt.Sprintf("New skill: %s\nIndexed and searchable in Skulto", d.newSkills[0].Slug)
		} else {
			var slugs []string
			for _, s := range d.newSkills {
				slugs = append(slugs, s.Slug)
			}
			message = fmt.Sprintf("Skills: %s\nIndexed and searchable in Skulto", strings.Join(slugs, ", "))
		}
	} else {
		titleColor = warningColor
		title = "Session Ended"
		message = "No new skills were detected.\nClaude may not have saved the skill, or you exited early."
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)
	titleView := titleStyle.Render(title)

	msgStyle := lipgloss.NewStyle().
		Foreground(textColor).
		Width(contentWidth).
		Align(lipgloss.Center)
	msgView := msgStyle.Render(message)

	// Show skill location hint
	hintStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1)
	hint := hintStyle.Render("Skills are saved to: ~/.skulto/skills/<name>/skill.md")

	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1)

	// Show install prompt if skills were created and platforms are configured
	var footerText string
	if len(d.newSkills) > 0 && d.installDialog != nil {
		footerText = "Enter/i: install | L: install later | r: try again"
	} else {
		footerText = "Enter: close | r: try again"
	}
	footer := footerStyle.Render(footerText)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleView,
		"",
		msgView,
		hint,
		"",
		footer,
	)

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	return dialogStyle.Render(content)
}

func (d *NewSkillDialog) viewInstall() string {
	dialogWidth := min(70, d.width-4)
	contentWidth := dialogWidth - 6

	accentColor := lipgloss.Color("#DC143C")
	goldColor := lipgloss.Color("#F1C40F")
	mutedColor := lipgloss.Color("#6B6B6B")
	textColor := lipgloss.Color("#E5E5E5")

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(goldColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	skillName := "skill"
	if d.installSkillInfo != nil {
		skillName = d.installSkillInfo.Slug
	}
	title := titleStyle.Render("Install \"" + skillName + "\"?")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(textColor).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)
	subtitle := subtitleStyle.Render("Select where to install this skill")

	// Embed install dialog view (without its own border)
	d.installDialog.SetWidth(contentWidth)
	installView := d.renderInstallOptions()

	// Install Later button
	laterStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(0, 2).
		MarginTop(1)
	laterBtn := laterStyle.Render("[L] Install Later")

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1)
	footerLine1 := "↑/↓: navigate  •  Space: toggle  •  Enter: confirm"
	footerLine2 := "a: all  •  n: none  •  g: global  •  p: project"
	footerLine3 := "L: later  •  Esc: back"
	footer := footerStyle.Render(footerLine1 + "\n" + footerLine2 + "\n" + footerLine3)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		installView,
		"",
		laterBtn,
		"",
		footer,
	)

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	return dialogStyle.Render(content)
}

// renderInstallOptions renders the install location options without dialog chrome.
func (d *NewSkillDialog) renderInstallOptions() string {
	if d.installDialog == nil {
		return ""
	}

	goldColor := lipgloss.Color("#F1C40F")
	mutedColor := lipgloss.Color("#6B6B6B")
	textColor := lipgloss.Color("#E5E5E5")
	successColor := lipgloss.Color("#10B981")
	selectedBgColor := lipgloss.Color("#1A1A2E")

	contentWidth := min(60, d.width-10)
	var optionViews []string

	for i, opt := range d.installDialog.options {
		isCurrent := i == d.installDialog.currentIndex

		// Checkbox
		var checkbox string
		if opt.Selected {
			checkbox = lipgloss.NewStyle().Foreground(successColor).Render("[x]")
		} else {
			checkbox = lipgloss.NewStyle().Foreground(mutedColor).Render("[ ]")
		}

		// Option text
		nameStyle := lipgloss.NewStyle().Foreground(textColor)
		descStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)

		if isCurrent {
			nameStyle = nameStyle.Foreground(goldColor).Bold(true)
		}

		line := checkbox + " " + nameStyle.Render(opt.DisplayName)
		desc := "    " + descStyle.Render(opt.Description)

		optContent := lipgloss.JoinVertical(lipgloss.Left, line, desc)

		optStyle := lipgloss.NewStyle().
			Width(contentWidth).
			Padding(0, 1)

		if isCurrent {
			optStyle = optStyle.Background(selectedBgColor)
		}

		optionViews = append(optionViews, optStyle.Render(optContent))
	}

	return lipgloss.JoinVertical(lipgloss.Left, optionViews...)
}

// CenteredView renders the dialog centered in the terminal.
func (d *NewSkillDialog) CenteredView(width, height int) string {
	dialog := d.View()
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, dialog)
}

// IsCancelled returns true if the dialog was cancelled.
func (d *NewSkillDialog) IsCancelled() bool {
	return d.cancelled
}

// IsConfirmed returns true if the user confirmed (wants to close/save).
func (d *NewSkillDialog) IsConfirmed() bool {
	return d.confirmed
}

// NeedsSave returns true if the dialog needs to save content via saveNewSkillCmd.
// This is only true for the legacy StatePreview flow. In StateResult (Claude flow),
// skills are already saved by Claude so no additional save is needed.
func (d *NewSkillDialog) NeedsSave() bool {
	return d.state == StatePreview && d.confirmed
}

// GetGeneratedContent returns the generated skill content.
func (d *NewSkillDialog) GetGeneratedContent() string {
	return d.generatedContent
}

// GetNewSkills returns the skills that were created/modified during the session.
func (d *NewSkillDialog) GetNewSkills() []skillgen.SkillInfo {
	return d.newSkills
}

// GetInstallLocations returns the selected install locations (if any).
func (d *NewSkillDialog) GetInstallLocations() []installer.InstallLocation {
	if d.installDialog == nil || !d.installSuccess {
		return nil
	}
	return d.installDialog.GetSelectedLocations()
}

// GetInstallSkillInfo returns the skill info for installation.
func (d *NewSkillDialog) GetInstallSkillInfo() *skillgen.SkillInfo {
	return d.installSkillInfo
}

// WantsInstall returns true if user confirmed installation.
func (d *NewSkillDialog) WantsInstall() bool {
	return d.installSuccess && len(d.GetInstallLocations()) > 0
}

// Reset clears the dialog state for reuse.
func (d *NewSkillDialog) Reset() {
	// Cancel any running generation to prevent context leak
	if d.cancelFunc != nil {
		d.cancelFunc()
		d.cancelFunc = nil
	}
	d.promptInput.Reset()
	d.state = StateInput
	d.streamingOutput = ""
	d.generatedContent = ""
	d.previewScroll = 0
	d.previewLines = nil
	d.err = nil
	d.cancelled = false
	d.confirmed = false
	d.launching = false
	d.skillsBefore = nil
	d.newSkills = nil
	d.focusedField = 0
	d.promptInput.Focus()
	// Clear install state
	d.installSkillInfo = nil
	d.installSuccess = false
	if d.installDialog != nil {
		d.installDialog.Reset()
	}
}

// NewSkillStreamMsg carries streaming output from CLI.
type NewSkillStreamMsg struct {
	Content string
	Done    bool
	Err     error
}

// NewSkillTickMsg is for animation updates.
type NewSkillTickMsg struct{}

// NewSkillClaudeFinishedMsg is sent when interactive Claude session ends.
type NewSkillClaudeFinishedMsg struct {
	Err error
}
