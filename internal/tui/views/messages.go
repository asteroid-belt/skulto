package views

import (
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/skillgen"
)

// NewSkillSavedMsg is sent when a quick-generated skill is saved.
type NewSkillSavedMsg struct {
	Success      bool
	Err          error
	SkillDir     string
	FilesCount   int
	BackupDir    string
	SavedToFiles bool
}

// NewSkillInstallMsg is sent when user wants to install a newly created skill.
type NewSkillInstallMsg struct {
	SkillInfo skillgen.SkillInfo
	Locations []installer.InstallLocation
}

// NewSkillInstallCompleteMsg is sent when installation completes.
type NewSkillInstallCompleteMsg struct {
	Success bool
	Err     error
}
