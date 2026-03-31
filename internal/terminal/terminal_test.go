package terminal_test

import (
	"testing"

	"github.com/lignumqt/taskglow/internal/terminal"
)

func TestWidthFallback(t *testing.T) {
	w := terminal.Width(999)
	if w != 80 {
		t.Errorf("Width(invalid fd) = %d; want 80 as fallback", w)
	}
}

func TestNoColor(t *testing.T) {
	_ = terminal.NoColor()
}

func TestIsCI(t *testing.T) {
	_ = terminal.IsCI()
}
