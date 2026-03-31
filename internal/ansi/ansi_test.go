package ansi_test

import (
	"strings"
	"testing"

	"github.com/lignumqt/taskglow/internal/ansi"
)

func TestCursorUp(t *testing.T) {
	if got := ansi.CursorUp(3); got != "\033[3A" {
		t.Errorf("CursorUp(3) = %q; want %q", got, "\033[3A")
	}
	if got := ansi.CursorUp(0); got != "" {
		t.Errorf("CursorUp(0) = %q; want empty", got)
	}
	if got := ansi.CursorUp(-1); got != "" {
		t.Errorf("CursorUp(-1) = %q; want empty", got)
	}
}

func TestCursorDown(t *testing.T) {
	if got := ansi.CursorDown(2); got != "\033[2B" {
		t.Errorf("CursorDown(2) = %q; want %q", got, "\033[2B")
	}
	if got := ansi.CursorDown(0); got != "" {
		t.Errorf("CursorDown(0) = %q; want empty", got)
	}
}

func TestColorize(t *testing.T) {
	s := ansi.Colorize(ansi.Green, "ok")
	if !strings.Contains(s, "ok") {
		t.Error("Colorize must contain the original text")
	}
	if !strings.HasPrefix(s, ansi.Green) {
		t.Error("Colorize must start with color code")
	}
	if !strings.HasSuffix(s, ansi.Reset) {
		t.Error("Colorize must end with Reset")
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello w\u2026"},
		{"hi", 1, "\u2026"},
		{"hello", 0, "\u2026"},
	}
	for _, c := range cases {
		got := ansi.Truncate(c.input, c.max)
		if got != c.want {
			t.Errorf("Truncate(%q, %d) = %q; want %q", c.input, c.max, got, c.want)
		}
	}
}
