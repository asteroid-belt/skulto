package installer

import "errors"

var (
	// ErrNoToolsSelected is returned when user has not selected any AI tools.
	ErrNoToolsSelected = errors.New("no AI tools selected - please complete onboarding")

	// ErrSkillNotFound is returned when a skill cannot be found.
	ErrSkillNotFound = errors.New("skill not found")

	// ErrInvalidSkill is returned when a skill is missing required fields.
	ErrInvalidSkill = errors.New("skill missing required fields (slug or content)")

	// ErrInstallFailed is returned when installation fails.
	ErrInstallFailed = errors.New("installation failed")

	// ErrSymlinkFailed is returned when symlink operations fail.
	ErrSymlinkFailed = errors.New("symlink operation failed")

	// ErrPermissionDenied is returned when permission is denied.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrPlatformUnsupported is returned for unsupported platforms.
	ErrPlatformUnsupported = errors.New("platform not supported")

	// ErrTranslationFailed is returned when translation fails.
	ErrTranslationFailed = errors.New("translation failed")

	// ErrInvalidScope is returned when an invalid installation scope is specified.
	ErrInvalidScope = errors.New("invalid installation scope")
)
