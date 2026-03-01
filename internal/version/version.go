package version

import "fmt"

// Build-time variables set via -ldflags
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info returns the short version string.
func Info() string {
	return Version
}

// Full returns detailed version information.
func Full() string {
	return fmt.Sprintf("version: %s\ncommit: %s\nbuilt: %s", Version, Commit, Date)
}
