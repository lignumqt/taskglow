package taskglow_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/lignumqt/taskglow/taskglow"
)

func TestTaskProgress_Clamp(t *testing.T) {
	var buf bytes.Buffer
	r := taskglow.New(
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModePlain),
	)

	err := r.Run(context.Background(), "Clamp", func(_ context.Context, t *taskglow.Task) error {
		t.Progress(-0.5, "negative")
		t.Progress(1.5, "above one")
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskContext_Propagated(t *testing.T) {
	var buf bytes.Buffer
	r := taskglow.New(
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModePlain),
	)

	type ctxKey struct{}
	parent := context.WithValue(context.Background(), ctxKey{}, "hello")

	err := r.Run(parent, "ValueTask", func(_ context.Context, t *taskglow.Task) error {
		if t.Context().Value(ctxKey{}) != "hello" {
			return errors.New("context value not propagated")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStateString(t *testing.T) {
	cases := map[taskglow.State]string{
		taskglow.StateIdle:      "idle",
		taskglow.StateRunning:   "running",
		taskglow.StateWarning:   "warning",
		taskglow.StatePaused:    "paused",
		taskglow.StateSucceeded: "succeeded",
		taskglow.StateFailed:    "failed",
		taskglow.StateCanceled:  "canceled",
	}
	for state, want := range cases {
		if got := state.String(); got != want {
			t.Errorf("State(%d).String() = %q; want %q", state, got, want)
		}
	}
}

func TestStateIsTerminal(t *testing.T) {
	terminals := []taskglow.State{
		taskglow.StateSucceeded,
		taskglow.StateFailed,
		taskglow.StateCanceled,
	}
	nonTerminals := []taskglow.State{
		taskglow.StateIdle,
		taskglow.StateRunning,
		taskglow.StateWarning,
		taskglow.StatePaused,
	}
	for _, s := range terminals {
		if !s.IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}
	for _, s := range nonTerminals {
		if s.IsTerminal() {
			t.Errorf("%s should NOT be terminal", s)
		}
	}
}
