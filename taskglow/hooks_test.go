package taskglow_test

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/lignumqt/taskglow/taskglow"
)

func TestWithOnFinish(t *testing.T) {
	var mu sync.Mutex
	var got taskglow.Summary

	runner := taskglow.New(
		taskglow.WithWriter(&bytes.Buffer{}),
		taskglow.WithMode(taskglow.ModePlain),
		taskglow.WithOnFinish(func(s taskglow.Summary) {
			mu.Lock()
			got = s
			mu.Unlock()
		}),
	)

	_ = runner.Run(context.Background(), "hook task", func(_ context.Context, t *taskglow.Task) error {
		t.Log("hello")
		return nil
	})

	mu.Lock()
	defer mu.Unlock()
	if got.Title != "hook task" {
		t.Errorf("expected title 'hook task', got %q", got.Title)
	}
	if got.State != taskglow.StateSucceeded {
		t.Errorf("expected StateSucceeded, got %v", got.State)
	}
	if len(got.Logs) == 0 || got.Logs[0] != "hello" {
		t.Errorf("expected log 'hello', got %v", got.Logs)
	}
}

func TestWithOnLog(t *testing.T) {
	var mu sync.Mutex
	var logged []string

	runner := taskglow.New(
		taskglow.WithWriter(&bytes.Buffer{}),
		taskglow.WithMode(taskglow.ModePlain),
		taskglow.WithOnLog(func(msg string) {
			mu.Lock()
			logged = append(logged, msg)
			mu.Unlock()
		}),
	)

	_ = runner.Run(context.Background(), "log task", func(_ context.Context, t *taskglow.Task) error {
		t.Log("line one")
		t.Log("line two")
		return nil
	})

	mu.Lock()
	defer mu.Unlock()
	if len(logged) != 2 {
		t.Fatalf("expected 2 log callbacks, got %d: %v", len(logged), logged)
	}
}

func TestWithOnWarn(t *testing.T) {
	var mu sync.Mutex
	var warned []string

	runner := taskglow.New(
		taskglow.WithWriter(&bytes.Buffer{}),
		taskglow.WithMode(taskglow.ModePlain),
		taskglow.WithOnWarn(func(msg string) {
			mu.Lock()
			warned = append(warned, msg)
			mu.Unlock()
		}),
	)

	_ = runner.Run(context.Background(), "warn task", func(_ context.Context, t *taskglow.Task) error {
		t.Warn("something off")
		return nil
	})

	mu.Lock()
	defer mu.Unlock()
	if len(warned) != 1 || warned[0] != "something off" {
		t.Errorf("unexpected warned: %v", warned)
	}
}

func TestWithLogFile(t *testing.T) {
	var renderBuf, logBuf bytes.Buffer

	runner := taskglow.New(
		taskglow.WithWriter(&renderBuf),
		taskglow.WithMode(taskglow.ModePlain),
		taskglow.WithLogFile(&logBuf),
	)

	_ = runner.Run(context.Background(), "file task", func(_ context.Context, t *taskglow.Task) error {
		t.Log("important event")
		t.Warn("low disk space")
		return nil
	})

	out := logBuf.String()
	if !strings.Contains(out, "important event") {
		t.Errorf("logFile missing log line, got:\n%s", out)
	}
	if !strings.Contains(out, "low disk space") {
		t.Errorf("logFile missing warning, got:\n%s", out)
	}
	if !strings.Contains(out, "file task") {
		t.Errorf("logFile missing task title, got:\n%s", out)
	}
}
