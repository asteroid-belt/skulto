// Package cli provides the command-line interface for Skulto.
package cli

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/pkg/version"
	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var telemetryClient telemetry.Client

var commandStartTime time.Time

var rootCmd = &cobra.Command{
	Use:   "skulto",
	Short: "Cross-Platform AI Skills Management",
	Long: `Cross-Platform AI Skills Management

An offline-first CLI tool for managing AI coding skills
across multiple platforms (Claude, Cursor, Copilot, Codex, OpenCode, Windsurf).

Run without arguments to launch the interactive TUI.

Telemetry:
  Telemetry is enabled by default, always anonymous, and will never track
  personal information, custom/local data, or IP addresses.

  It will only be used to improve Skulto.

  Opt-out with:
  	SKULTO_TELEMETRY_TRACKING_ENABLED=false`,
	SilenceUsage: true,
	RunE:         runTUI,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		commandStartTime = time.Now()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Track command execution (skip for root TUI command)
		if cmd.Name() != "skulto" {
			durationMs := time.Since(commandStartTime).Milliseconds()
			hasFlags := cmd.Flags().NFlag() > 0
			telemetryClient.TrackCLICommandExecuted(cmd.Name(), hasFlags, durationMs)
		}

		// Track help viewed if --help was used
		if cmd.Flags().Changed("help") {
			telemetryClient.TrackCLIHelpViewed(cmd.Name(), os.Args[1:])
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(favoritesCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(updateCmd)
}

// Execute runs the CLI with fang enhancements.
func Execute(ctx context.Context, tc telemetry.Client) error {
	if tc == nil {
		tc = telemetry.New(nil)
	}
	telemetryClient = tc

	err := fang.Execute(
		ctx,
		rootCmd,
		fang.WithVersion(version.Short()),
		fang.WithCommit(version.Commit),
	)

	// Track app exit for CLI mode (non-TUI subcommands)
	if rootCmd.CalledAs() != "" && rootCmd.CalledAs() != "skulto" {
		durationMs := time.Since(commandStartTime).Milliseconds()
		telemetryClient.TrackAppExited("cli", durationMs, 1)
	}

	return err
}

// trackCLIError wraps an error with telemetry tracking.
// Call this before returning errors from CLI commands.
func trackCLIError(cmdName string, err error) error {
	if err == nil {
		return nil
	}
	errorType := classifyError(err)
	telemetryClient.TrackCLIError(cmdName, errorType)
	return err
}

// classifyError determines the error type for telemetry.
func classifyError(err error) string {
	errStr := err.Error()
	switch {
	case containsAny(errStr, "config", "configuration"):
		return "config_error"
	case containsAny(errStr, "database", "db"):
		return "database_error"
	case containsAny(errStr, "network", "timeout", "connection"):
		return "network_error"
	case containsAny(errStr, "permission", "access denied"):
		return "permission_error"
	case containsAny(errStr, "not found", "does not exist"):
		return "not_found_error"
	case containsAny(errStr, "invalid", "parse", "format"):
		return "validation_error"
	default:
		return "unknown_error"
	}
}

// containsAny checks if s contains any of the substrings (case-insensitive).
func containsAny(s string, substrs ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range substrs {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}
