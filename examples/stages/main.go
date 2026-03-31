package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lignumqt/taskglow/taskglow"
)

var buildStages = []struct {
	name string
	dur  time.Duration
}{
	{"compile", 600 * time.Millisecond},
	{"link", 400 * time.Millisecond},
	{"package", 300 * time.Millisecond},
	{"upload", 700 * time.Millisecond},
}

func main() {
	err := taskglow.Wrap(context.Background(), "Building and releasing v1.2.3", func(ctx context.Context, t *taskglow.Task) error {
		n := len(buildStages)
		for i, s := range buildStages {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			t.Stage(s.name, i+1, n)
			t.Progress(float64(i)/float64(n), "")

			for elapsed := time.Duration(0); elapsed < s.dur; elapsed += 50 * time.Millisecond {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(50 * time.Millisecond):
				}
				t.Progress(
					float64(i)/float64(n)+float64(elapsed+50*time.Millisecond)/float64(s.dur)/float64(n),
					"",
				)
			}

			t.Log(fmt.Sprintf("stage %s complete", s.name))
		}
		t.Progress(1.0, "done")
		return nil
	})
	if err != nil {
		fmt.Printf("release failed: %v\n", err)
	}
}
