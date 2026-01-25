// Package version provides build version information.
package version

import (
	"fmt"
	"runtime"
)

// These are set via ldflags at build time.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info returns formatted version information.
func Info() string {
	commitShort := Commit
	if len(commitShort) > 7 {
		commitShort = commitShort[:7]
	}
	return fmt.Sprintf(
		"skulto %s (%s) built on %s with %s",
		Version,
		commitShort,
		BuildDate,
		runtime.Version(),
	)
}

// Short returns just the version number.
func Short() string {
	return Version
}

// Full returns all version details.
func Full() string {
	return fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s\nGo Version: %s\nOS/Arch: %s/%s",
		Version,
		Commit,
		BuildDate,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}
