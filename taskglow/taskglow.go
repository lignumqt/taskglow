// Package taskglow provides a terminal UX layer for long-running operations.
//
// The primary entry point is [Wrap], which wraps any function with a managed
// terminal session displaying spinners, progress bars, and live logs.
// It automatically detects whether the output is a TTY, CI, or plain pipe and
// chooses the appropriate renderer.
//
// Basic usage:
//
//	err := taskglow.Wrap(ctx, "Deploying", func(ctx context.Context, t *taskglow.Task) error {
//	  t.Progress(0.3, "fetching artifacts")
//	  t.Stage("Build", 1, 3)
//	  t.Log("compiled 42 files")
//	  return nil
//	})
package taskglow

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/lignumqt/taskglow/internal/terminal"
	"github.com/lignumqt/taskglow/render"
)

// TaskFunc is the signature of the callback passed to Run/Wrap.
type TaskFunc func(ctx context.Context, t *Task) error

// Runner manages task execution and rendering.
type Runner struct {
	opts options
}

// New creates a Runner with the given options.
func New(opts ...Option) *Runner {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &Runner{opts: o}
}

// Run executes fn under a managed terminal session with the given title.
// It blocks until fn returns, or the context is canceled, or fn panics.
// Run returns the error from fn, a wrapped context error, or a panic as an error.
func (r *Runner) Run(ctx context.Context, title string, fn TaskFunc) (err error) {
	renderer := r.buildRenderer()
	renderer.Start(title)

	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	t := newTask(taskCtx, cancel, title, renderer, r.opts.onLog, r.opts.onWarn)
	startedAt := time.Now()

	// Catch panics in the user callback and convert them to errors.
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic in task %q: %v", title, p)
		}
	}()

	// Watch parent context cancellation in a separate goroutine.
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		select {
		case <-ctx.Done():
			t.mu.Lock()
			if t.state == StateRunning || t.state == StateWarning {
				t.state = StateCanceled
			}
			t.mu.Unlock()
		case <-taskCtx.Done():
		}
	}()

	fnErr := fn(taskCtx, t)

	// Capture context error BEFORE calling cancel(), so we can distinguish
	// "context was already canceled before fn returned" (user/parent cancel)
	// from "fn returned normally and we cancel the child ctx ourselves".
	fnCtxErr := taskCtx.Err()

	cancel()
	<-watchDone

	state, hint, logs, warnings := t.snapshot()

	if fnErr != nil {
		if fnCtxErr != nil || ctx.Err() != nil {
			// fn returned an error because the context was canceled.
			state = StateCanceled
			fnErr = nil
		} else {
			state = StateFailed
		}
	} else if state == StateRunning || state == StateWarning {
		state = StateSucceeded
	}

	elapsed := time.Since(startedAt)

	fs := render.FinishState{
		Title:     title,
		StateCode: int(state),
		StateStr:  state.String(),
		Elapsed:   FormatElapsed(elapsed),
		Err:       fnErr,
		Hint:      hint,
		Logs:      logs,
		Warnings:  warnings,
	}
	renderer.Finish(fs)

	// Write structured log to file if configured.
	if r.opts.logFile != nil {
		r.writeLogFile(r.opts.logFile, title, logs, warnings, fs)
	}

	// Fire OnFinish hook.
	if r.opts.onFinish != nil {
		r.opts.onFinish(Summary{
			Title:    title,
			State:    state,
			Err:      fnErr,
			Hint:     hint,
			Elapsed:  elapsed,
			Logs:     logs,
			Warnings: warnings,
		})
	}

	if state == StateFailed {
		return fnErr
	}
	if state == StateCanceled {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return context.Canceled
	}
	return nil
}

// writeLogFile writes a structured plain-text log of the task to w.
func (r *Runner) writeLogFile(w io.Writer, title string, logs, warnings []string, fs render.FinishState) {
	fmt.Fprintf(w, "» %s\n", title)
	for _, warn := range warnings {
		fmt.Fprintf(w, "  warn: %s\n", warn)
	}
	for _, l := range logs {
		fmt.Fprintf(w, "  log: %s\n", l)
	}
	switch fs.StateCode {
	case int(StateSucceeded):
		fmt.Fprintf(w, "✓ %s [%s]\n", fs.Title, fs.Elapsed)
	case int(StateFailed):
		if fs.Err != nil {
			fmt.Fprintf(w, "✗ %s: %v [%s]\n", fs.Title, fs.Err, fs.Elapsed)
		} else {
			fmt.Fprintf(w, "✗ %s [%s]\n", fs.Title, fs.Elapsed)
		}
		if fs.Hint != "" {
			fmt.Fprintf(w, "  hint: %s\n", fs.Hint)
		}
	case int(StateCanceled):
		fmt.Fprintf(w, "⊘ %s (canceled) [%s]\n", fs.Title, fs.Elapsed)
	}
}

// buildRenderer selects and constructs the appropriate renderer.
func (r *Runner) buildRenderer() render.Renderer {
	mode := r.opts.mode
	if mode == ModeAuto {
		if terminal.IsStdoutTerminal() && !terminal.IsCI() {
			mode = ModeTTY
		} else {
			mode = ModePlain
		}
	}

	switch mode {
	case ModeTTY:
		w := r.opts.writer
		if f, ok := w.(*os.File); !ok || !terminal.IsTerminal(f.Fd()) {
			return render.NewPlain(w)
		}
		width := terminal.StdoutWidth()
		return render.NewTTY(w, r.opts.spinnerFrames, r.opts.renderInterval, r.opts.noColor, width, false)
	case ModeVerbose:
		w := r.opts.writer
		if f, ok := w.(*os.File); ok && terminal.IsTerminal(f.Fd()) {
			width := terminal.StdoutWidth()
			return render.NewTTY(w, r.opts.spinnerFrames, r.opts.renderInterval, r.opts.noColor, width, true)
		}
		return render.NewPlain(w)
	case ModeQuiet:
		return &quietRenderer{}
	default:
		return render.NewPlain(r.opts.writer)
	}
}

// Wrap is the top-level convenience function. It creates a Runner with default
// options and calls Run. This is the primary killer feature of taskglow.
func Wrap(ctx context.Context, title string, fn TaskFunc) error {
	return New().Run(ctx, title, fn)
}

// quietRenderer suppresses all output.
type quietRenderer struct{}

func (q *quietRenderer) Start(_ string)              {}
func (q *quietRenderer) Update(_ render.Event)       {}
func (q *quietRenderer) Finish(_ render.FinishState) {}
func (q *quietRenderer) Stop()                       {}

var _ render.Renderer = (*quietRenderer)(nil)

// WriterFrom returns the io.Writer configured for the runner.
func (r *Runner) WriterFrom() io.Writer {
	return r.opts.writer
}
