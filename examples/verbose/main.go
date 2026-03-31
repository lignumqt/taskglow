package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lignumqt/taskglow/taskglow"
)

// Run with: go run ./examples/verbose/...
//
// ModeVerbose shows the spinner/progress-bar during execution (just like the
// default TTY mode) but keeps all log and warning lines on screen after the
// task finishes instead of erasing them.
//
// Typical use: `mytool deploy --verbose`
func main() {
	runner := taskglow.New(taskglow.WithMode(taskglow.ModeVerbose))

	err := runner.Run(context.Background(), "Deploying application", func(ctx context.Context, t *taskglow.Task) error {
		steps := []struct {
			msg string
			dur time.Duration
		}{
			{"Connecting to server", 300 * time.Millisecond},
			{"Uploading build artifacts", 600 * time.Millisecond},
			{"Running database migrations", 400 * time.Millisecond},
			{"Restarting services", 200 * time.Millisecond},
		}

		for i, s := range steps {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			t.Stage(s.msg, i+1, len(steps))
			t.Progress(float64(i)/float64(len(steps)), s.msg)
			t.Log(s.msg + "...")

			time.Sleep(s.dur)
		}

		t.Warn("config.yaml uses a deprecated key 'timeout_ms', use 'timeout' instead")
		t.Progress(1.0, "done")
		return nil
	})

	if err != nil {
		fmt.Printf("deploy failed: %v\n", err)
	}
}
