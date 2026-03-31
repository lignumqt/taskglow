package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lignumqt/taskglow/taskglow"
)

func main() {
	err := taskglow.Wrap(context.Background(), "Deploying application", func(ctx context.Context, t *taskglow.Task) error {
		t.Log("Connecting to server...")
		time.Sleep(500 * time.Millisecond)

		t.Log("Uploading build artifacts")
		time.Sleep(700 * time.Millisecond)

		t.Log("Restarting services")
		time.Sleep(300 * time.Millisecond)

		return nil
	})
	if err != nil {
		fmt.Printf("deploy failed: %v\n", err)
	}
}
