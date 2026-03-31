package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lignumqt/taskglow/taskglow"
)

func main() {
	err := taskglow.Wrap(context.Background(), "Processing dataset", func(ctx context.Context, t *taskglow.Task) error {
		total := 100
		for i := 0; i < total; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			t.Progress(float64(i+1)/float64(total), fmt.Sprintf("record %d/%d", i+1, total))
			time.Sleep(20 * time.Millisecond)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("processing failed: %v\n", err)
	}
}
