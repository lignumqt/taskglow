// Package ansi provides minimal ANSI escape code helpers for terminal rendering.
package ansi

import "fmt"

// Reset resets all terminal attributes.
const Reset = "[0m"

// Bold text.
const Bold = "[1m"

// Color codes (foreground).
const (
	Black   = "[30m"
	Red     = "[31m"
	Green   = "[32m"
	Yellow  = "[33m"
	Blue    = "[34m"
	Magenta = "[35m"
	Cyan    = "[36m"
	White   = "[37m"

	BrightBlack   = "[90m"
	BrightRed     = "[91m"
	BrightGreen   = "[92m"
	BrightYellow  = "[93m"
	BrightBlue    = "[94m"
	BrightMagenta = "[95m"
	BrightCyan    = "[96m"
	BrightWhite   = "[97m"
)

// CursorUp moves the cursor up n lines.
func CursorUp(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("[%dA", n)
}

// CursorDown moves the cursor down n lines.
func CursorDown(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("[%dB", n)
}

// CursorColumn moves the cursor to column n (1-based).
func CursorColumn(n int) string {
	return fmt.Sprintf("[%dG", n)
}

// ClearLine erases the entire current line.
const ClearLine = "[2K"

// ClearLineRight erases from cursor to end of line.
const ClearLineRight = "[K"

// HideCursor hides the terminal cursor.
const HideCursor = "[?25l"

// ShowCursor shows the terminal cursor.
const ShowCursor = "[?25h"

// Colorize wraps text with a color code and reset.
func Colorize(color, text string) string {
	return color + text + Reset
}

// Truncate truncates s to max runes, appending "…" if truncated.
func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(runes[:max-1]) + "…"
}
