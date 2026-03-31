// Package render defines the Renderer interface and event types used to
// communicate task state changes from core to output backends.
package render

import "github.com/lignumqt/taskglow/internal/terminal"

// EventKind identifies the type of render event.
type EventKind int

const (
	// EventStart signals that a task has begun.
	EventStart EventKind = iota
	// EventProgress signals a progress update (0.0-1.0).
	EventProgress
	// EventLog signals a new log line.
	EventLog
	// EventWarn signals a warning message.
	EventWarn
	// EventStage signals a stage transition.
	EventStage
	// EventSucceed signals successful completion.
	EventSucceed
	// EventFail signals failure.
	EventFail
	// EventCancel signals cancellation.
	EventCancel
)

// Event carries a single render update from the task to the renderer.
type Event struct {
	// Kind identifies what changed.
	Kind EventKind
	// Title is the task title (set on EventStart, carried through).
	Title string
	// Message is a human-readable message associated with this event.
	Message string
	// Progress is a value in [0.0, 1.0] (only meaningful for EventProgress).
	Progress float64
	// Current is the current stage index (only meaningful for EventStage).
	Current int
	// Total is the total stage count (only meaningful for EventStage).
	Total int
	// Err is the error (only meaningful for EventFail).
	Err error
}

// FinishState carries the data that renderers need to draw a final summary.
type FinishState struct {
	// Title is the task title.
	Title string
	// StateStr is the human-readable state name (e.g. "succeeded").
	StateStr string
	// StateCode is the numeric state value (matches taskglow.State constants).
	StateCode int
	// Elapsed is a human-friendly elapsed time string.
	Elapsed string
	// Err is the error, if any.
	Err error
	// Hint is an optional user-facing hint for the failed state.
	Hint string
	// Logs holds all log lines collected during the task.
	Logs []string
	// Warnings holds all warning messages collected during the task.
	Warnings []string
}

// Renderer is the interface implemented by all output backends.
// Implementations must be safe for concurrent use.
type Renderer interface {
	// Start initialises the renderer and displays the initial task state.
	Start(title string)
	// Update applies an event to the current display.
	Update(e Event)
	// Finish renders the final summary and cleans up terminal state.
	Finish(fs FinishState)
	// Stop cleans up resources without rendering a final summary.
	Stop()
}

// CapabilityWriter is an optional interface that renderers may implement to
// expose safe log writing so external code can inject log lines directly.
type CapabilityWriter interface {
	// Write writes a pre-formatted log line to the renderer's log area.
	Write(line string)
}

// DetectMode returns true when the environment supports interactive TTY output.
func DetectMode() bool {
	return terminal.IsStdoutTerminal() && !terminal.IsCI() && !terminal.NoColor()
}
