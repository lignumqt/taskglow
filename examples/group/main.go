package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lignumqt/taskglow/taskglow"
)

// Run with: go run ./examples/group/...
//
// Demonstrates running multiple tasks in parallel with taskglow.Group.
// In a TTY each task renders as its own spinner row; in CI/pipes they
// interleave as plain text through a shared mutex-protected writer.
func main() {
	ctx := context.Background()

	g := taskglow.NewGroup(ctx)

	g.Go("Build frontend", func(ctx context.Context, t *taskglow.Task) error {
		steps := []string{"lint", "compile TypeScript", "bundle assets", "optimise images"}
		for i, s := range steps {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			t.Stage(s, i+1, len(steps))
			t.Progress(float64(i+1)/float64(len(steps)), s)
			time.Sleep(300 * time.Millisecond)
		}
		return nil
	})

	g.Go("Build backend", func(ctx context.Context, t *taskglow.Task) error {
		steps := []string{"vet", "compile", "link"}
		for i, s := range steps {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			t.Stage(s, i+1, len(steps))
			t.Progress(float64(i+1)/float64(len(steps)), s)
			time.Sleep(250 * time.Millisecond)
		}
		return nil
	})

	g.Go("Run DB migrations", func(ctx context.Context, t *taskglow.Task) error {
		t.Log("connecting to database")
		time.Sleep(100 * time.Millisecond)
		for i := 1; i <= 5; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			t.Logf("applying migration %03d", i)
			t.Progress(float64(i)/5.0, fmt.Sprintf("migration %d/5", i))
			time.Sleep(120 * time.Millisecond)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		fmt.Printf("group failed: %v\n", err)
	}
}
