package cli

import (
	"fmt"
	"io"

	"github.com/asteroid-belt/skulto/internal/db"
)

// showStartupNotification displays notification for unnotified discoveries.
// Returns true if any notifications were shown.
func showStartupNotification(database *db.DB, w io.Writer) bool {
	if database == nil {
		return false
	}

	discoveries, err := database.ListUnnotifiedDiscoveredSkills()
	if err != nil || len(discoveries) == 0 {
		return false
	}

	// Print header with count
	skillWord := "skill"
	if len(discoveries) > 1 {
		skillWord = "skills"
	}
	_, _ = fmt.Fprintf(w, "\nFound %d unmanaged %s:\n", len(discoveries), skillWord)

	// Collect IDs for marking as notified
	var ids []string
	for _, d := range discoveries {
		scope := fmt.Sprintf("(%s)", d.Scope)
		_, _ = fmt.Fprintf(w, "  %-40s %s\n", d.Path, scope)
		ids = append(ids, d.ID)
	}

	// Print help message
	_, _ = fmt.Fprintf(w, "\nRun `skulto ingest` or use Manage view to import.\n\n")

	// Mark as notified (non-fatal if this fails)
	_ = database.MarkDiscoveredSkillsNotified(ids)

	return true
}
