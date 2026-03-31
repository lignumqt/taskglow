package exec_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	execadapter "github.com/lignumqt/taskglow/adapters/exec"
	"github.com/lignumqt/taskglow/taskglow"
)

func TestRunSuccess(t *testing.T) {
	var buf bytes.Buffer
	runner := taskglow.New(
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModePlain),
	)

	err := runner.Run(context.Background(), "echo", func(ctx context.Context, task *taskglow.Task) error {
		res, err := execadapter.Run(ctx, task, "echo", "hello")
		if err != nil {
			return err
		}
		if len(res.Stdout) == 0 || !strings.Contains(res.Stdout[0], "hello") {
			t.Errorf("expected 'hello' in stdout, got %v", res.Stdout)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFailure(t *testing.T) {
	var buf bytes.Buffer
	runner := taskglow.New(
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModePlain),
	)

	err := runner.Run(context.Background(), "false", func(ctx context.Context, task *taskglow.Task) error {
		_, err := execadapter.Run(ctx, task, "false")
		return err
	})
	if err == nil {
		t.Fatal("expected error from 'false' command, got nil")
	}
}

func TestRunContextCancel(t *testing.T) {
	var buf bytes.Buffer
	runner := taskglow.New(
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModePlain),
	)

	ctx, cancel := context.WithCancel(context.Background())

	err := runner.Run(ctx, "sleep", func(ctx context.Context, task *taskglow.Task) error {
		cancel()
		_, err := execadapter.Run(ctx, task, "sleep", "10")
		return err
	})
	if err == nil {
		t.Fatal("expected error after context cancel")
	}
}
