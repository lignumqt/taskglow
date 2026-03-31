// Package otel integrates taskglow with OpenTelemetry tracing.
//
// Each [Runner.Run] call creates a span for the task duration. Log lines are
// recorded as span events, warnings as events with a "warning=true" attribute,
// and the span status is set according to the final task state.
// The span context is propagated into the task function's ctx so the callback
// can start child spans with standard OTel calls.
//
// Basic usage:
//
// runner := taskglowotel.New(tracer)
// err := runner.Run(ctx, "Deploy", fn)
//
// Parallel tasks are supported via [NewGroup].
package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lignumqt/taskglow/taskglow"
)

// Runner wraps [taskglow.Runner] and automatically opens an OTel span for
// every task execution.
type Runner struct {
	tracer trace.Tracer
	opts   []taskglow.Option
}

// New creates a Runner that starts a [trace.Span] via tracer for every Run
// call. Options are forwarded to the underlying [taskglow.Runner].
//
// Note: do not pass [taskglow.WithOnLog], [taskglow.WithOnWarn] or
// [taskglow.WithOnFinish] in opts — the adapter uses those hooks internally.
func New(tracer trace.Tracer, opts ...taskglow.Option) *Runner {
	return &Runner{tracer: tracer, opts: opts}
}

// Run starts a span named title, executes fn and ends the span on return.
// Logs and warnings are emitted as span events; span status reflects the
// final task outcome.
func (r *Runner) Run(ctx context.Context, title string, fn taskglow.TaskFunc) error {
	ctx, span := r.tracer.Start(ctx, title)
	defer span.End()

	otelOpts := make([]taskglow.Option, 0, len(r.opts)+3)
	otelOpts = append(otelOpts, r.opts...)
	otelOpts = append(otelOpts,
		taskglow.WithOnLog(func(msg string) {
			span.AddEvent(msg)
		}),
		taskglow.WithOnWarn(func(msg string) {
			span.AddEvent(msg, trace.WithAttributes(
				attribute.Bool("warning", true),
			))
		}),
		taskglow.WithOnFinish(func(s taskglow.Summary) {
			span.SetAttributes(
				attribute.String("task.state", s.State.String()),
				attribute.String("task.elapsed", taskglow.FormatElapsed(s.Elapsed)),
			)
			if len(s.Warnings) > 0 {
				span.SetAttributes(attribute.Int("task.warnings", len(s.Warnings)))
			}
			if s.State == taskglow.StateFailed && s.Err != nil {
				span.RecordError(s.Err)
				span.SetStatus(codes.Error, s.Err.Error())
			} else {
				span.SetStatus(codes.Ok, "")
			}
		}),
	)

	return taskglow.New(otelOpts...).Run(ctx, title, fn)
}

// groupTask is an internal record storing a pending task for [Group].
type groupTask struct {
	title string
	fn    taskglow.TaskFunc
}

// Group mirrors [taskglow.Group] but wraps each task in an individual OTel
// child span. A parent span named "group" covers the whole [Group.Wait] call.
//
// Note: do not pass [taskglow.WithOnLog], [taskglow.WithOnWarn] or
// [taskglow.WithOnFinish] in opts.
type Group struct {
	tracer trace.Tracer
	opts   []taskglow.Option
	ctx    context.Context
	tasks  []groupTask
}

// NewGroup creates a Group whose tasks are each wrapped in an OTel span under a
// shared parent "group" span.
func NewGroup(ctx context.Context, tracer trace.Tracer, opts ...taskglow.Option) *Group {
	return &Group{tracer: tracer, opts: opts, ctx: ctx}
}

// Go registers a task.
func (g *Group) Go(title string, fn taskglow.TaskFunc) {
	g.tasks = append(g.tasks, groupTask{title: title, fn: fn})
}

// Wait runs all registered tasks concurrently under a parent "group" span and
// blocks until all tasks finish. Returns the first non-nil error.
func (g *Group) Wait() error {
	ctx, span := g.tracer.Start(g.ctx, "group")
	defer span.End()

	ng := taskglow.NewGroup(ctx, g.opts...)
	for _, t := range g.tasks {
		title, fn := t.title, t.fn
		ng.Go(title, func(ctx context.Context, task *taskglow.Task) error {
			childCtx, childSpan := g.tracer.Start(ctx, title)
			defer childSpan.End()

			err := fn(childCtx, task)
			if err != nil {
				childSpan.RecordError(err)
				childSpan.SetStatus(codes.Error, err.Error())
			} else {
				childSpan.SetStatus(codes.Ok, "")
			}
			return err
		})
	}

	err := ng.Wait()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}
