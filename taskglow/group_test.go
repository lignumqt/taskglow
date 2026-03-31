package taskglow_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lignumqt/taskglow/taskglow"
)

func TestGroupAllSucceed(t *testing.T) {
	var buf bytes.Buffer
	g := taskglow.NewGroup(context.Background(),
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModePlain),
	)
	g.Go("task A", func(_ context.Context, t *taskglow.Task) error {
		t.Log("a done")
		return nil
	})
	g.Go("task B", func(_ context.Context, t *taskglow.Task) error {
		t.Log("b done")
		return nil
	})

	if err := g.Wait(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "task A") {
		t.Errorf("output missing 'task A': %q", out)
	}
	if !strings.Contains(out, "task B") {
		t.Errorf("output missing 'task B': %q", out)
	}
}

func TestGroupFirstError(t *testing.T) {
	var buf bytes.Buffer
	g := taskglow.NewGroup(context.Background(),
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModePlain),
	)
	g.Go("ok", func(_ context.Context, _ *taskglow.Task) error { return nil })
	g.Go("fail", func(_ context.Context, _ *taskglow.Task) error {
		return errors.New("boom")
	})

	err := g.Wait()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected 'boom' in error, got: %v", err)
	}
}

func TestGroupRunsConcurrently(t *testing.T) {
	var buf bytes.Buffer
	var active atomic.Int32
	var maxActive atomic.Int32

	g := taskglow.NewGroup(context.Background(),
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModePlain),
	)

	for i := 0; i < 4; i++ {
		g.Go("task", func(_ context.Context, _ *taskglow.Task) error {
			cur := active.Add(1)
			for {
				if old := maxActive.Load(); cur > old {
					if maxActive.CompareAndSwap(old, cur) {
						break
					}
				} else {
					break
				}
			}
			time.Sleep(80 * time.Millisecond)
			active.Add(-1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
	if maxActive.Load() < 2 {
		t.Errorf("tasks do not appear to run concurrently (max concurrent: %d)", maxActive.Load())
	}
}

func TestGroupResults(t *testing.T) {
	var buf bytes.Buffer
	g := taskglow.NewGroup(context.Background(),
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModePlain),
	)
	g.Go("alpha", func(_ context.Context, t *taskglow.Task) error {
		t.Log("alpha log")
		return nil
	})
	g.Wait() //nolint:errcheck
	results := g.Results()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "alpha" {
		t.Errorf("unexpected title: %q", results[0].Title)
	}
}

func TestGroupContextCancel(t *testing.T) {
	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())

	g := taskglow.NewGroup(ctx,
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModePlain),
	)
	g.Go("long task", func(ctx context.Context, _ *taskglow.Task) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	})

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := g.Wait()
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}
