package render

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/lignumqt/taskglow/internal/ansi"
	"github.com/lignumqt/taskglow/internal/throttle"
)

const (
	maxLogLines = 5
	barWidth    = 30
)

// TTY is an interactive renderer that uses ANSI escape codes to draw
// a spinner, progress bar, and live log area in the terminal.
type TTY struct {
	mu     sync.Mutex
	w      io.Writer
	opts   ttyOptions
	th     *throttle.Throttle
	ticker *time.Ticker
	stop   chan struct{}
	wg     sync.WaitGroup

	// mutable state (protected by mu)
	title        string
	progress     float64
	msg          string
	stageLine    string
	logLines     []string
	warnings     []string
	spinIdx      int
	linesWritten int
	started      time.Time
}

type ttyOptions struct {
	spinnerFrames  []string
	renderInterval time.Duration
	noColor        bool
	width          int
	verbose        bool
}

// NewTTY creates an interactive TTY renderer.
// When verbose is true, all accumulated log and warning lines are printed
// to the terminal after the task finishes instead of being discarded.
func NewTTY(w io.Writer, spinnerFrames []string, renderInterval time.Duration, noColor bool, width int, verbose bool) *TTY {
	frames := spinnerFrames
	if len(frames) == 0 {
		frames = []string{"⠋"}
	}
	r := &TTY{
		w:    w,
		stop: make(chan struct{}),
		opts: ttyOptions{
			spinnerFrames:  frames,
			renderInterval: renderInterval,
			noColor:        noColor,
			width:          width,
			verbose:        verbose,
		},
	}
	r.th = throttle.New(renderInterval, r.redraw)
	return r
}

// Start begins the spinner goroutine and prints the first frame.
func (r *TTY) Start(title string) {
	r.mu.Lock()
	r.title = title
	r.started = time.Now()
	r.mu.Unlock()

	r.ticker = time.NewTicker(r.opts.renderInterval)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-r.stop:
				return
			case <-r.ticker.C:
				r.mu.Lock()
				r.spinIdx = (r.spinIdx + 1) % len(r.opts.spinnerFrames)
				r.mu.Unlock()
				r.th.Trigger()
			}
		}
	}()
	r.th.Trigger()
}

// Update applies an event to the renderer state and schedules a redraw.
func (r *TTY) Update(e Event) {
	r.mu.Lock()
	switch e.Kind {
	case EventProgress:
		r.progress = e.Progress
		r.msg = e.Message
	case EventLog:
		if e.Message != "" {
			r.logLines = appendCapped(r.logLines, e.Message, maxLogLines)
		}
	case EventWarn:
		if e.Message != "" {
			r.warnings = appendCapped(r.warnings, "⚠ "+e.Message, maxLogLines)
		}
	case EventStage:
		r.stageLine = fmt.Sprintf("[%d/%d] %s", e.Current, e.Total, e.Message)
	}
	r.mu.Unlock()
	r.th.Trigger()
}

// Finish stops the spinner, clears the live area, and prints the summary.
func (r *TTY) Finish(fs FinishState) {
	r.Stop()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.clearLiveArea()

	// In verbose mode, print every accumulated log and warning so they are
	// preserved on screen after the live spinner area is gone.
	if r.opts.verbose {
		for _, w := range fs.Warnings {
			fmt.Fprintf(r.w, "  %s\n", r.colorize(ansi.Yellow, "⚠ "+w))
		}
		for _, l := range fs.Logs {
			fmt.Fprintf(r.w, "  %s\n", r.dim(l))
		}
	}

	const (
		codeSucceeded = 4
		codeFailed    = 5
		codeCanceled  = 6
	)

	switch fs.StateCode {
	case codeSucceeded:
		icon := r.colorize(ansi.BrightGreen, "✓")
		fmt.Fprintf(r.w, "%s %s %s\n", icon, r.bold(fs.Title), r.dim("["+fs.Elapsed+"]"))
	case codeFailed:
		icon := r.colorize(ansi.BrightRed, "✗")
		fmt.Fprintf(r.w, "%s %s %s\n", icon, r.bold(fs.Title), r.dim("["+fs.Elapsed+"]"))
		if fs.Err != nil {
			fmt.Fprintf(r.w, "  %s %v\n", r.colorize(ansi.Red, "error:"), fs.Err)
		}
		if fs.Hint != "" {
			fmt.Fprintf(r.w, "  %s %s\n", r.colorize(ansi.Yellow, "hint:"), fs.Hint)
		}
	case codeCanceled:
		icon := r.colorize(ansi.Yellow, "⊘")
		fmt.Fprintf(r.w, "%s %s (canceled) %s\n", icon, r.bold(fs.Title), r.dim("["+fs.Elapsed+"]"))
	default:
		fmt.Fprintf(r.w, "? %s (%s) [%s]\n", fs.Title, fs.StateStr, fs.Elapsed)
	}
}

// Stop stops the spinner goroutine and waits for all background work to finish.
// It is safe to call Stop multiple times.
func (r *TTY) Stop() {
	select {
	case <-r.stop:
		return
	default:
	}
	close(r.stop)
	if r.ticker != nil {
		r.ticker.Stop()
	}
	// Wait for the spinner goroutine to exit BEFORE stopping the throttle.
	// The spinner goroutine calls Trigger(); if we stopped the throttle first,
	// a racing Trigger() could schedule a new flush goroutine after Stop() returned.
	r.wg.Wait()
	r.th.Stop()
}

// Write implements CapabilityWriter.
func (r *TTY) Write(line string) {
	r.mu.Lock()
	r.logLines = appendCapped(r.logLines, line, maxLogLines)
	r.mu.Unlock()
	r.th.Trigger()
}

func (r *TTY) redraw() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clearLiveArea()
	lines := r.buildFrame()
	fmt.Fprint(r.w, lines)
	r.linesWritten = strings.Count(lines, "\n")
}

func (r *TTY) clearLiveArea() {
	if r.linesWritten == 0 {
		return
	}
	fmt.Fprint(r.w, ansi.CursorUp(r.linesWritten))
	for i := 0; i < r.linesWritten; i++ {
		fmt.Fprint(r.w, ansi.ClearLine+"\n")
	}
	fmt.Fprint(r.w, ansi.CursorUp(r.linesWritten))
}

func (r *TTY) buildFrame() string {
	var b strings.Builder

	frame := r.opts.spinnerFrames[r.spinIdx%len(r.opts.spinnerFrames)]
	spinner := r.colorize(ansi.Cyan, frame)

	width := r.opts.width
	if width <= 0 {
		width = 80
	}

	titleLine := fmt.Sprintf("%s %s", spinner, r.bold(r.title))
	if r.stageLine != "" {
		titleLine += "  " + r.colorize(ansi.BrightBlack, r.stageLine)
	}
	fmt.Fprintln(&b, ansi.Truncate(titleLine, width+30))

	if r.progress > 0 || r.msg != "" {
		bar := buildBar(r.progress, barWidth)
		pct := fmt.Sprintf("%3d%%", int(r.progress*100))
		line := fmt.Sprintf("  %s %s", r.colorize(ansi.Blue, bar), r.colorize(ansi.White, pct))
		if r.msg != "" {
			line += "  " + ansi.Truncate(r.msg, width-len(pct)-barWidth-8)
		}
		fmt.Fprintln(&b, line)
	}

	for _, l := range r.logLines {
		fmt.Fprintln(&b, "  "+r.dim(ansi.Truncate(l, width-4)))
	}

	for _, w := range r.warnings {
		fmt.Fprintln(&b, "  "+r.colorize(ansi.Yellow, ansi.Truncate(w, width-4)))
	}

	return b.String()
}

func buildBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	filled := int(float64(width) * pct)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return "[" + bar + "]"
}

func (r *TTY) colorize(color, text string) string {
	if r.opts.noColor {
		return text
	}
	return ansi.Colorize(color, text)
}

func (r *TTY) bold(text string) string {
	if r.opts.noColor {
		return text
	}
	return ansi.Bold + text + ansi.Reset
}

func (r *TTY) dim(text string) string {
	if r.opts.noColor {
		return text
	}
	return ansi.BrightBlack + text + ansi.Reset
}

func appendCapped(lines []string, line string, cap int) []string {
	lines = append(lines, line)
	if len(lines) > cap {
		lines = lines[len(lines)-cap:]
	}
	return lines
}

// ensure TTY satisfies Renderer at compile time.
var _ Renderer = (*TTY)(nil)
