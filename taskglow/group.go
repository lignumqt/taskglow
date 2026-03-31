package taskglow

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/lignumqt/taskglow/internal/terminal"
	"github.com/lignumqt/taskglow/render"
)

// GroupEntry describes a single task inside a [Group].
type GroupEntry struct {
	Title string
	Fn    TaskFunc
}

// GroupResult holds the outcome of one task in a [Group].
type GroupResult struct {
	Title   string
	Summary Summary
}

// Group runs multiple tasks concurrently, rendering them as a live stacked list
// of spinner rows (TTY) or sequential plain-text output (non-TTY / CI).
//
// All tasks start simultaneously. [Group.Wait] blocks until all finish and
// returns the first non-nil error, similar to errgroup.
//
// Example:
//
// g := taskglow.NewGroup(ctx)
// g.Go("Build frontend", buildFn)
// g.Go("Run migrations", migrateFn)
//
//	if err := g.Wait(); err != nil {
//	   log.Fatal(err)
//	}
type Group struct {
	ctx     context.Context
	opts    options
	entries []GroupEntry
	results []GroupResult
	mu      sync.Mutex
}

// NewGroup creates a Group that will run tasks with the provided options.
func NewGroup(ctx context.Context, opts ...Option) *Group {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &Group{ctx: ctx, opts: o}
}

// Go adds a task to the group.
func (g *Group) Go(title string, fn TaskFunc) {
	g.entries = append(g.entries, GroupEntry{Title: title, Fn: fn})
}

// Wait starts all registered tasks concurrently and blocks until they all
// complete. It returns the first non-nil error encountered, or nil.
func (g *Group) Wait() error {
	if len(g.entries) == 0 {
		return nil
	}

	if g.isTTY() {
		return g.runTTY()
	}
	return g.runPlain()
}

// Results returns the per-task summaries after Wait returns.
func (g *Group) Results() []GroupResult {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]GroupResult, len(g.results))
	copy(out, g.results)
	return out
}

func (g *Group) isTTY() bool {
	switch g.opts.mode {
	case ModeAuto:
		return terminal.IsStdoutTerminal() && !terminal.IsCI()
	case ModeTTY, ModeVerbose:
		if f, ok := g.opts.writer.(*os.File); ok {
			return terminal.IsTerminal(f.Fd())
		}
		return false
	default:
		return false
	}
}

func (g *Group) runTTY() error {
	width := terminal.StdoutWidth()
	verbose := g.opts.mode == ModeVerbose

	grp := render.NewGroupTTY(g.opts.writer, g.opts.spinnerFrames, g.opts.renderInterval, g.opts.noColor, width)

	rowRenderers := make([]render.Renderer, len(g.entries))
	for i, entry := range g.entries {
		rr := grp.AddRow(entry.Title)
		if verbose {
			rowRenderers[i] = &groupVerboseRenderer{GroupRowRenderer: rr}
		} else {
			rowRenderers[i] = rr
		}
	}

	grp.Start()

	type taskResult struct {
		idx     int
		summary Summary
		err     error
	}

	resultCh := make(chan taskResult, len(g.entries))

	for i, entry := range g.entries {
		i, entry, renderer := i, entry, rowRenderers[i]
		go func() {
			sum, err := g.runOne(g.ctx, entry.Title, entry.Fn, renderer)
			resultCh <- taskResult{idx: i, summary: sum, err: err}
		}()
	}

	results := make([]GroupResult, len(g.entries))
	var firstErr error
	for range g.entries {
		r := <-resultCh
		results[r.idx] = GroupResult{Title: g.entries[r.idx].Title, Summary: r.summary}
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
	}

	grp.Stop()
	grp.PrintFinal()

	g.mu.Lock()
	g.results = results
	g.mu.Unlock()

	g.fireHooks(results)

	return firstErr
}

func (g *Group) runPlain() error {
	type taskResult struct {
		idx     int
		summary Summary
		err     error
	}

	resultCh := make(chan taskResult, len(g.entries))

	// Wrap shared writer so concurrent Plain renderers don't race each other.
	sw := render.NewSyncWriter(g.opts.writer)

	for i, entry := range g.entries {
		i, entry := i, entry
		renderer := render.NewPlain(sw)
		go func() {
			sum, err := g.runOne(g.ctx, entry.Title, entry.Fn, renderer)
			resultCh <- taskResult{idx: i, summary: sum, err: err}
		}()
	}

	results := make([]GroupResult, len(g.entries))
	var firstErr error
	for range g.entries {
		r := <-resultCh
		results[r.idx] = GroupResult{Title: g.entries[r.idx].Title, Summary: r.summary}
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
	}

	g.mu.Lock()
	g.results = results
	g.mu.Unlock()

	g.fireHooks(results)

	return firstErr
}

// runOne runs a single task function and returns its Summary + error.
func (g *Group) runOne(ctx context.Context, title string, fn TaskFunc, renderer render.Renderer) (sum Summary, err error) {
	renderer.Start(title)

	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	t := newTask(taskCtx, cancel, title, renderer, g.opts.onLog, g.opts.onWarn)
	startedAt := time.Now()

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
	fnCtxErr := taskCtx.Err()
	cancel()
	<-watchDone

	state, hint, logs, warnings := t.snapshot()

	if fnErr != nil {
		if fnCtxErr != nil || ctx.Err() != nil {
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

	if g.opts.logFile != nil {
		r := &Runner{opts: g.opts}
		r.writeLogFile(g.opts.logFile, title, logs, warnings, fs)
	}

	sum = Summary{
		Title:    title,
		State:    state,
		Err:      fnErr,
		Hint:     hint,
		Elapsed:  elapsed,
		Logs:     logs,
		Warnings: warnings,
	}

	if state == StateFailed {
		return sum, fnErr
	}
	if state == StateCanceled {
		if ctx.Err() != nil {
			return sum, ctx.Err()
		}
		return sum, context.Canceled
	}
	return sum, nil
}

func (g *Group) fireHooks(results []GroupResult) {
	if g.opts.onFinish == nil {
		return
	}
	for _, r := range results {
		g.opts.onFinish(r.Summary)
	}
}

// groupVerboseRenderer is a placeholder for future verbose group rendering.
type groupVerboseRenderer struct {
	*render.GroupRowRenderer
}
