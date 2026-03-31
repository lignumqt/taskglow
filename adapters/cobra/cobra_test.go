package cobra_test

import (
	"context"
	"errors"
	"testing"

	cobraadapter "github.com/lignumqt/taskglow/adapters/cobra"
	"github.com/lignumqt/taskglow/taskglow"
)

// fakeCmd simulates cobra.Command for testing without importing cobra.
type fakeCmd struct{ Name string }

func TestRunE_Success(t *testing.T) {
	var buf []string
	fn := cobraadapter.RunE[*fakeCmd]("Build", func(_ context.Context, task *taskglow.Task, cmd *fakeCmd, args []string) error {
		buf = append(buf, "called")
		task.Log("log line")
		return nil
	})

	if err := fn(&fakeCmd{Name: "test"}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buf) != 1 || buf[0] != "called" {
		t.Error("RunE did not call the wrapped function")
	}
}

func TestRunE_Error(t *testing.T) {
	want := errors.New("build failed")
	fn := cobraadapter.RunE[*fakeCmd]("Build", func(_ context.Context, _ *taskglow.Task, _ *fakeCmd, _ []string) error {
		return want
	})

	err := fn(&fakeCmd{}, nil)
	if !errors.Is(err, want) {
		t.Fatalf("RunE error = %v; want %v", err, want)
	}
}

func TestRunEContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fn := cobraadapter.RunEContext[*fakeCmd](ctx, "Canceled", func(ctx context.Context, _ *taskglow.Task, _ *fakeCmd, _ []string) error {
		<-ctx.Done()
		return ctx.Err()
	})

	err := fn(&fakeCmd{}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
