package taskglow_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lignumqt/taskglow/taskglow"
)

func newPlainRunner(buf *bytes.Buffer) *taskglow.Runner {
	return taskglow.New(
		taskglow.WithWriter(buf),
		taskglow.WithMode(taskglow.ModePlain),
	)
}

func TestRunSuccess(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	err := r.Run(context.Background(), "Build", func(_ context.Context, t *taskglow.Task) error {
		t.Log("compiled 5 files")
		return nil
	})
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Build") {
		t.Errorf("output %q does not contain task title", out)
	}
}

func TestRunFailure(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	want := errors.New("build failed")
	err := r.Run(context.Background(), "Build", func(_ context.Context, t *taskglow.Task) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Run error = %v; want %v", err, want)
	}
}

func TestRunContextCancel(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.Run(ctx, "Deploy", func(ctx context.Context, _ *taskglow.Task) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	})

	if err == nil {
		t.Fatal("expected error on canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRunContextTimeout(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Run(ctx, "Slow", func(ctx context.Context, _ *taskglow.Task) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	})

	if err == nil {
		t.Fatal("expected error on timeout, got nil")
	}
}

func TestRunPanicRecovery(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	err := r.Run(context.Background(), "Panic task", func(_ context.Context, _ *taskglow.Task) error {
		panic("something went wrong")
	})

	if err == nil {
		t.Fatal("expected error from panic, got nil")
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Errorf("error %q does not mention panic", err.Error())
	}
}

func TestRunProgress(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	err := r.Run(context.Background(), "Upload", func(_ context.Context, t *taskglow.Task) error {
		t.Progress(0.0, "starting")
		t.Progress(0.5, "midway")
		t.Progress(1.0, "done")
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "50") {
		t.Errorf("output %q does not contain progress percentage", out)
	}
}

func TestRunStages(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	err := r.Run(context.Background(), "Pipeline", func(_ context.Context, t *taskglow.Task) error {
		t.Stage("Checkout", 1, 3)
		t.Stage("Build", 2, 3)
		t.Stage("Deploy", 3, 3)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "3/3") {
		t.Errorf("output %q does not contain stage number 3/3", out)
	}
}

func TestRunWarn(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	err := r.Run(context.Background(), "Scan", func(_ context.Context, t *taskglow.Task) error {
		t.Warn("deprecated API usage")
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "deprecated API usage") {
		t.Errorf("output does not contain warning")
	}
}

func TestRunHint(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	err := r.Run(context.Background(), "Connect", func(_ context.Context, t *taskglow.Task) error {
		t.SetHint("check your firewall rules")
		return errors.New("timeout")
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(buf.String(), "check your firewall rules") {
		t.Errorf("output does not contain hint")
	}
}

func TestWrap(t *testing.T) {
	err := taskglow.Wrap(context.Background(), "Quick task", func(_ context.Context, _ *taskglow.Task) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Wrap returned unexpected error: %v", err)
	}
}

func TestRunQuietMode(t *testing.T) {
	var buf bytes.Buffer
	r := taskglow.New(
		taskglow.WithWriter(&buf),
		taskglow.WithMode(taskglow.ModeQuiet),
	)

	err := r.Run(context.Background(), "Silent", func(_ context.Context, t *taskglow.Task) error {
		t.Log("this should not appear")
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("quiet mode should produce no output, got %q", buf.String())
	}
}

func TestRunConcurrentUpdates(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	err := r.Run(context.Background(), "Concurrent", func(_ context.Context, t *taskglow.Task) error {
		done := make(chan struct{})
		for i := 0; i < 30; i++ {
			go func(i int) {
				t.Log("msg")
				t.Progress(float64(i)/30, "working")
				done <- struct{}{}
			}(i)
		}
		for i := 0; i < 30; i++ {
			<-done
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskExplicitSucceed(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	err := r.Run(context.Background(), "Task", func(_ context.Context, t *taskglow.Task) error {
		t.Succeed("all done")
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "\u2713") {
		t.Errorf("output does not contain success marker")
	}
}

func TestTaskExplicitCancel(t *testing.T) {
	var buf bytes.Buffer
	r := newPlainRunner(&buf)

	err := r.Run(context.Background(), "Task", func(_ context.Context, t *taskglow.Task) error {
		t.Cancel()
		return nil
	})
	if err == nil {
		t.Fatal("expected error on explicit cancel")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestFormatElapsed(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1.5s"},
		{90 * time.Second, "1m30s"},
	}
	for _, c := range cases {
		got := taskglow.FormatElapsed(c.d)
		if got != c.want {
			t.Errorf("FormatElapsed(%v) = %q; want %q", c.d, got, c.want)
		}
	}
}
