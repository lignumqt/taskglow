package taskglow

import (
	"context"
	"sync"
	"time"

	"github.com/lignumqt/taskglow/render"
)

// Task is the handle given to user callbacks to report progress, logs, and state.
// All methods are safe for concurrent use from multiple goroutines.
type Task struct {
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	renderer  render.Renderer
	state     State
	title     string
	hint      string
	logs      []string
	warnings  []string
	startedAt time.Time
}

func newTask(ctx context.Context, cancel context.CancelFunc, title string, r render.Renderer) *Task {
	return &Task{
		ctx:       ctx,
		cancel:    cancel,
		renderer:  r,
		state:     StateRunning,
		title:     title,
		startedAt: time.Now(),
	}
}

// Context returns the context associated with this task.
func (t *Task) Context() context.Context {
	return t.ctx
}

// Progress reports a progress update.
// pct must be in [0.0, 1.0]. msg is a human-readable status description.
func (t *Task) Progress(pct float64, msg string) {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	t.renderer.Update(render.Event{
		Kind:     render.EventProgress,
		Progress: pct,
		Message:  msg,
	})
}

// Log appends a log line to the live log area.
func (t *Task) Log(msg string) {
	t.mu.Lock()
	t.logs = append(t.logs, msg)
	t.mu.Unlock()
	t.renderer.Update(render.Event{Kind: render.EventLog, Message: msg})
}

// Stage reports a named stage transition.
// current is the 1-based index of the current stage; total is the total count.
func (t *Task) Stage(name string, current, total int) {
	t.renderer.Update(render.Event{
		Kind:    render.EventStage,
		Message: name,
		Current: current,
		Total:   total,
	})
}

// Warn reports a non-fatal warning.
func (t *Task) Warn(msg string) {
	t.mu.Lock()
	t.warnings = append(t.warnings, msg)
	t.mu.Unlock()
	t.renderer.Update(render.Event{Kind: render.EventWarn, Message: msg})
}

// SetHint sets an optional hint shown to the user when the task fails.
func (t *Task) SetHint(hint string) {
	t.mu.Lock()
	t.hint = hint
	t.mu.Unlock()
}

// Succeed marks the task as completed successfully.
func (t *Task) Succeed(msg string) {
	t.mu.Lock()
	t.state = StateSucceeded
	t.mu.Unlock()
	if msg != "" {
		t.renderer.Update(render.Event{Kind: render.EventLog, Message: msg})
	}
}

// Fail marks the task as failed with the provided error.
func (t *Task) Fail(err error) {
	t.mu.Lock()
	t.state = StateFailed
	t.mu.Unlock()
	t.renderer.Update(render.Event{Kind: render.EventFail, Err: err})
}

// Cancel marks the task as canceled and cancels the associated context.
func (t *Task) Cancel() {
	t.mu.Lock()
	t.state = StateCanceled
	t.mu.Unlock()
	t.cancel()
}

// snapshot returns a copy of the mutable task fields under the lock.
func (t *Task) snapshot() (state State, hint string, logs []string, warnings []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	logs = make([]string, len(t.logs))
	copy(logs, t.logs)
	warnings = make([]string, len(t.warnings))
	copy(warnings, t.warnings)
	return t.state, t.hint, logs, warnings
}
