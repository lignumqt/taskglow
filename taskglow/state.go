package taskglow

// State represents the lifecycle state of a task.
type State int

const (
	// StateIdle means the task has not started yet.
	StateIdle State = iota
	// StateRunning means the task is actively executing.
	StateRunning
	// StateWarning means the task is running but encountered a non-fatal issue.
	StateWarning
	// StatePaused means the task execution is temporarily suspended.
	StatePaused
	// StateSucceeded means the task completed successfully.
	StateSucceeded
	// StateFailed means the task completed with a fatal error.
	StateFailed
	// StateCanceled means the task was canceled via context or user request.
	StateCanceled
)

// IsTerminal reports whether the state is a final (non-recoverable) state.
func (s State) IsTerminal() bool {
	return s == StateSucceeded || s == StateFailed || s == StateCanceled
}

// String returns a human-readable name for the state.
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StateWarning:
		return "warning"
	case StatePaused:
		return "paused"
	case StateSucceeded:
		return "succeeded"
	case StateFailed:
		return "failed"
	case StateCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}
