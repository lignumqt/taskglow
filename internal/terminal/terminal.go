// Package terminal provides helpers for detecting terminal capabilities.
package terminal

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether fd refers to a terminal.
func IsTerminal(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

// IsStdoutTerminal reports whether os.Stdout is a terminal.
func IsStdoutTerminal() bool {
	return IsTerminal(os.Stdout.Fd())
}

// Width returns the terminal width in columns for the given fd.
// Returns 80 as a safe fallback when the width cannot be determined.
func Width(fd uintptr) int {
	w, _, err := term.GetSize(int(fd))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

// StdoutWidth returns the column width of os.Stdout.
func StdoutWidth() int {
	return Width(os.Stdout.Fd())
}

// IsCI reports whether the process is running inside a CI environment.
func IsCI() bool {
	ciVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
		"JENKINS_URL",
		"BUILDKITE",
		"DRONE",
		"TEAMCITY_VERSION",
		"TF_BUILD",
	}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// NoColor reports whether color output should be suppressed.
func NoColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return true
	}
	return false
}
