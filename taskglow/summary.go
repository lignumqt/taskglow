package taskglow

import (
	"fmt"
	"strings"
	"time"
)

// StageInfo records information about a completed or active stage.
type StageInfo struct {
	// Name is the human-readable stage label.
	Name string
	// Current is the 1-based index of this stage.
	Current int
	// Total is the total number of stages.
	Total int
	// StartedAt records when the stage was first reported.
	StartedAt time.Time
}

// Summary is the structured result emitted at the end of a task run.
type Summary struct {
	// Title is the task title.
	Title string
	// State is the terminal state of the task.
	State State
	// Err holds the error if State == StateFailed.
	Err error
	// Hint is an optional hint for the user when the task fails.
	Hint string
	// Elapsed is the total wall-clock duration of the task.
	Elapsed time.Duration
	// Stages holds all stages that were reported during the task.
	Stages []StageInfo
	// Logs holds all log lines emitted during the task.
	Logs []string
	// Warnings holds all warning messages emitted during the task.
	Warnings []string
}

// FormatElapsed returns a human-friendly duration string.
func FormatElapsed(d time.Duration) string {
	d = d.Round(time.Millisecond)
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%02ds", m, s)
}

// String returns a compact, single-line summary suitable for plain output.
func (s *Summary) String() string {
	var b strings.Builder
	switch s.State {
	case StateSucceeded:
		fmt.Fprintf(&b, "\u2713 %s", s.Title)
	case StateFailed:
		fmt.Fprintf(&b, "\u2717 %s", s.Title)
		if s.Err != nil {
			fmt.Fprintf(&b, ": %v", s.Err)
		}
	case StateCanceled:
		fmt.Fprintf(&b, "\u2298 %s (canceled)", s.Title)
	default:
		fmt.Fprintf(&b, "? %s (%s)", s.Title, s.State)
	}
	fmt.Fprintf(&b, " [%s]", FormatElapsed(s.Elapsed))
	return b.String()
}
