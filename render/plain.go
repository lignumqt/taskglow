package render

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Plain is a renderer that writes simple, non-interactive text.
// It is appropriate for CI, piped output, and redirected streams.
type Plain struct {
	mu      sync.Mutex
	w       io.Writer
	title   string
	started time.Time
}

// NewPlain creates a Plain renderer writing to w.
func NewPlain(w io.Writer) *Plain {
	return &Plain{w: w}
}

// Start prints the task start line.
func (p *Plain) Start(title string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.title = title
	p.started = time.Now()
	fmt.Fprintf(p.w, "\u00bb %s\n", title)
}

// Update prints task events as plain text lines.
func (p *Plain) Update(e Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch e.Kind {
	case EventProgress:
		if e.Message != "" {
			pct := int(e.Progress * 100)
			fmt.Fprintf(p.w, "  [%3d%%] %s\n", pct, e.Message)
		}
	case EventLog:
		if e.Message != "" {
			fmt.Fprintf(p.w, "  log: %s\n", e.Message)
		}
	case EventWarn:
		if e.Message != "" {
			fmt.Fprintf(p.w, "  warn: %s\n", e.Message)
		}
	case EventStage:
		if e.Message != "" {
			fmt.Fprintf(p.w, "  [%d/%d] %s\n", e.Current, e.Total, e.Message)
		}
	}
}

// Finish prints the final result line.
func (p *Plain) Finish(fs FinishState) {
	p.mu.Lock()
	defer p.mu.Unlock()

	const (
		codeSucceeded = 4
		codeFailed    = 5
		codeCanceled  = 6
	)

	switch fs.StateCode {
	case codeSucceeded:
		fmt.Fprintf(p.w, "\u2713 %s [%s]\n", fs.Title, fs.Elapsed)
	case codeFailed:
		if fs.Err != nil {
			fmt.Fprintf(p.w, "\u2717 %s: %v [%s]\n", fs.Title, fs.Err, fs.Elapsed)
		} else {
			fmt.Fprintf(p.w, "\u2717 %s [%s]\n", fs.Title, fs.Elapsed)
		}
		if fs.Hint != "" {
			fmt.Fprintf(p.w, "  hint: %s\n", fs.Hint)
		}
	case codeCanceled:
		fmt.Fprintf(p.w, "\u2298 %s (canceled) [%s]\n", fs.Title, fs.Elapsed)
	default:
		fmt.Fprintf(p.w, "? %s (%s) [%s]\n", fs.Title, fs.StateStr, fs.Elapsed)
	}
}

// Stop is a no-op for Plain (no terminal state to clean up).
func (p *Plain) Stop() {}

// ensure Plain satisfies Renderer at compile time.
var _ Renderer = (*Plain)(nil)
