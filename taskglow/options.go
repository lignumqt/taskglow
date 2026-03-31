package taskglow

import (
	"io"
	"os"
	"time"

	"github.com/lignumqt/taskglow/internal/terminal"
)

// Mode controls how output is rendered.
type Mode int

const (
	// ModeAuto detects the appropriate mode based on the environment.
	ModeAuto Mode = iota
	// ModeTTY forces the interactive spinner/progress UI.
	ModeTTY
	// ModePlain forces plain text output suitable for CI and pipes.
	ModePlain
	// ModeQuiet suppresses all output except the final summary.
	ModeQuiet
	// ModeVerbose enables verbose output including all log messages.
	ModeVerbose
)

// Option is a functional option for configuring a Runner.
type Option func(*options)

type options struct {
	writer          io.Writer
	mode            Mode
	renderInterval  time.Duration
	spinnerFrames   []string
	noColor         bool
	indentSize      int
	summaryOnFinish bool
}

func defaultOptions() options {
	return options{
		writer:          os.Stdout,
		mode:            ModeAuto,
		renderInterval:  80 * time.Millisecond,
		spinnerFrames:   defaultSpinnerFrames,
		noColor:         terminal.NoColor(),
		indentSize:      2,
		summaryOnFinish: true,
	}
}

// WithWriter sets the output writer.
func WithWriter(w io.Writer) Option {
	return func(o *options) { o.writer = w }
}

// WithMode sets the rendering mode explicitly.
func WithMode(m Mode) Option {
	return func(o *options) { o.mode = m }
}

// WithRenderInterval sets how often the TTY renderer refreshes.
func WithRenderInterval(d time.Duration) Option {
	return func(o *options) {
		if d > 0 {
			o.renderInterval = d
		}
	}
}

// WithSpinnerFrames sets custom spinner animation frames.
func WithSpinnerFrames(frames []string) Option {
	return func(o *options) {
		if len(frames) > 0 {
			o.spinnerFrames = frames
		}
	}
}

// WithNoColor disables ANSI color output regardless of terminal support.
func WithNoColor(v bool) Option {
	return func(o *options) { o.noColor = v }
}

// WithSummary controls whether a summary is printed after task completion.
func WithSummary(v bool) Option {
	return func(o *options) { o.summaryOnFinish = v }
}

// defaultSpinnerFrames is the default braille-dot spinner animation.
var defaultSpinnerFrames = []string{
	"\u280b", "\u2819", "\u2839", "\u2838", "\u283c", "\u2834", "\u2826", "\u2827", "\u2807", "\u280f",
}
