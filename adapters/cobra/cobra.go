// Package cobra provides a helper for integrating taskglow with cobra commands.
//
// This adapter does not import cobra as a dependency. It uses generics to work
// with any command type, allowing projects that already depend on cobra to use
// taskglow without adding a transitive cobra dependency to taskglow itself.
//
// Usage example (in your project that has cobra as a dependency):
//
//	var deployCmd = &cobra.Command{
//	   Use:   "deploy",
//	   Short: "Deploy the application",
//	   RunE: cobraadapter.RunE("Deploying", func(ctx context.Context, t *taskglow.Task, cmd *cobra.Command, args []string) error {
//	       t.Progress(0.5, "pushing containers")
//	       return nil
//	   }),
//	}
package cobra

import (
	"context"

	"github.com/lignumqt/taskglow/taskglow"
)

// CommandFunc is the function signature for cobra-style command handlers that
// use taskglow. C is the cobra.Command type in your project.
type CommandFunc[C any] func(ctx context.Context, t *taskglow.Task, cmd C, args []string) error

// RunE wraps a CommandFunc so it can be assigned to cobra.Command.RunE.
// It creates a taskglow Runner, starts a session titled title, and calls fn.
func RunE[C any](title string, fn CommandFunc[C], opts ...taskglow.Option) func(cmd C, args []string) error {
	return func(cmd C, args []string) error {
		r := taskglow.New(opts...)
		return r.Run(context.Background(), title, func(ctx context.Context, t *taskglow.Task) error {
			return fn(ctx, t, cmd, args)
		})
	}
}

// RunEContext is like RunE but the context is provided externally.
func RunEContext[C any](ctx context.Context, title string, fn CommandFunc[C], opts ...taskglow.Option) func(cmd C, args []string) error {
	return func(cmd C, args []string) error {
		r := taskglow.New(opts...)
		return r.Run(ctx, title, func(ctx context.Context, t *taskglow.Task) error {
			return fn(ctx, t, cmd, args)
		})
	}
}
