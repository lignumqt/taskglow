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

// rowRenderer tracks live display state for a single task within a Group.
type rowRenderer struct {
	mu        sync.Mutex
	title     string
	progress  float64
	msg       string
	stageLine string
	logLine   string // last log line
	warnLine  string // last warning line
	spinIdx   int
	stateCode int // 0 = running, set to FinishState.StateCode on finish
	elapsed   string
	finishErr error
}

// GroupTTY renders multiple concurrent tasks as a stacked list of spinner rows.
// Each task gets one line; the list is redrawn in place on every tick.
type GroupTTY struct {
	mu     sync.Mutex
	w      io.Writer
	rows   []*rowRenderer
	th     *throttle.Throttle
	ticker *time.Ticker
	stop   chan struct{}
	wg     sync.WaitGroup

	spinnerFrames []string
	noColor       bool
	width         int
	linesWritten  int
}

// NewGroupTTY creates a renderer that writes parallel-task rows to w.
func NewGroupTTY(w io.Writer, spinnerFrames []string, renderInterval time.Duration, noColor bool, width int) *GroupTTY {
	frames := spinnerFrames
	if len(frames) == 0 {
		frames = []string{"⠋"}
	}
	g := &GroupTTY{
		w:             w,
		spinnerFrames: frames,
		noColor:       noColor,
		width:         width,
		stop:          make(chan struct{}),
	}
	g.th = throttle.New(renderInterval, g.redraw)
	return g
}

// AddRow registers a new task row and returns its renderer handle.
// Must be called before Start.
func (g *GroupTTY) AddRow(title string) *GroupRowRenderer {
	row := &rowRenderer{title: title}
	g.mu.Lock()
	g.rows = append(g.rows, row)
	g.mu.Unlock()
	return &GroupRowRenderer{row: row, group: g}
}

// Start begins the spinner ticker goroutine.
func (g *GroupTTY) Start() {
	g.ticker = time.NewTicker(80 * time.Millisecond)
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		for {
			select {
			case <-g.stop:
				return
			case <-g.ticker.C:
				g.mu.Lock()
				for _, row := range g.rows {
					row.mu.Lock()
					if row.stateCode == 0 {
						row.spinIdx = (row.spinIdx + 1) % len(g.spinnerFrames)
					}
					row.mu.Unlock()
				}
				g.mu.Unlock()
				g.th.Trigger()
			}
		}
	}()
	g.th.Trigger()
}

// Stop stops the ticker and throttle.
func (g *GroupTTY) Stop() {
	select {
	case <-g.stop:
		return
	default:
	}
	close(g.stop)
	g.ticker.Stop()
	g.wg.Wait()
	g.th.Stop()
}

// PrintFinal clears the live area and prints static summary lines for all rows.
func (g *GroupTTY) PrintFinal() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.clearLiveArea()
	for _, row := range g.rows {
		row.mu.Lock()
		fmt.Fprint(g.w, g.buildRowLine(row)+"\n")
		row.mu.Unlock()
	}
}

func (g *GroupTTY) redraw() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.clearLiveArea()
	var sb strings.Builder
	for _, row := range g.rows {
		row.mu.Lock()
		sb.WriteString(g.buildRowLine(row))
		sb.WriteByte('\n')
		row.mu.Unlock()
	}
	out := sb.String()
	fmt.Fprint(g.w, out)
	g.linesWritten = strings.Count(out, "\n")
}

func (g *GroupTTY) clearLiveArea() {
	if g.linesWritten == 0 {
		return
	}
	fmt.Fprint(g.w, ansi.CursorUp(g.linesWritten))
	for i := 0; i < g.linesWritten; i++ {
		fmt.Fprint(g.w, ansi.ClearLine+"\n")
	}
	fmt.Fprint(g.w, ansi.CursorUp(g.linesWritten))
}

func (g *GroupTTY) buildRowLine(row *rowRenderer) string {
	width := g.width
	if width <= 0 {
		width = 80
	}

	const (
		codeSucceeded = 4
		codeFailed    = 5
		codeCanceled  = 6
	)

	// Finished row.
	if row.stateCode != 0 {
		switch row.stateCode {
		case codeSucceeded:
			icon := g.colorize(ansi.BrightGreen, "✓")
			return fmt.Sprintf("%s %s %s", icon, g.bold(row.title), g.dim("["+row.elapsed+"]"))
		case codeFailed:
			icon := g.colorize(ansi.BrightRed, "✗")
			line := fmt.Sprintf("%s %s %s", icon, g.bold(row.title), g.dim("["+row.elapsed+"]"))
			if row.finishErr != nil {
				line += " " + g.colorize(ansi.Red, row.finishErr.Error())
			}
			return line
		case codeCanceled:
			icon := g.colorize(ansi.Yellow, "⊘")
			return fmt.Sprintf("%s %s (canceled) %s", icon, g.bold(row.title), g.dim("["+row.elapsed+"]"))
		}
	}

	// Running row.
	frame := g.spinnerFrames[row.spinIdx%len(g.spinnerFrames)]
	spinner := g.colorize(ansi.Cyan, frame)

	line := fmt.Sprintf("%s %s", spinner, g.bold(row.title))
	if row.stageLine != "" {
		line += " " + g.colorize(ansi.BrightBlack, row.stageLine)
	}
	if row.progress > 0 {
		pct := fmt.Sprintf(" %3d%%", int(row.progress*100))
		line += g.colorize(ansi.White, pct)
	}
	status := row.logLine
	if row.warnLine != "" {
		status = g.colorize(ansi.Yellow, "⚠ "+row.warnLine)
	}
	if status != "" {
		line += "  " + g.dim(ansi.Truncate(status, width-40))
	}
	return ansi.Truncate(line, width+40)
}

func (g *GroupTTY) colorize(color, text string) string {
	if g.noColor {
		return text
	}
	return ansi.Colorize(color, text)
}

func (g *GroupTTY) bold(text string) string {
	if g.noColor {
		return text
	}
	return ansi.Bold + text + ansi.Reset
}

func (g *GroupTTY) dim(text string) string {
	if g.noColor {
		return text
	}
	return ansi.BrightBlack + text + ansi.Reset
}

// GroupRowRenderer is a Renderer for one task inside a GroupTTY.
type GroupRowRenderer struct {
	row   *rowRenderer
	group *GroupTTY
}

func (r *GroupRowRenderer) Start(_ string) { r.group.th.Trigger() }

func (r *GroupRowRenderer) Update(e Event) {
	r.row.mu.Lock()
	switch e.Kind {
	case EventProgress:
		r.row.progress = e.Progress
		r.row.msg = e.Message
	case EventLog:
		r.row.logLine = e.Message
	case EventWarn:
		r.row.warnLine = e.Message
	case EventStage:
		r.row.stageLine = fmt.Sprintf("[%d/%d] %s", e.Current, e.Total, e.Message)
	}
	r.row.mu.Unlock()
	r.group.th.Trigger()
}

func (r *GroupRowRenderer) Finish(fs FinishState) {
	r.row.mu.Lock()
	r.row.stateCode = fs.StateCode
	r.row.elapsed = fs.Elapsed
	r.row.finishErr = fs.Err
	r.row.mu.Unlock()
	r.group.th.Trigger()
}

func (r *GroupRowRenderer) Stop() {}

var _ Renderer = (*GroupRowRenderer)(nil)
