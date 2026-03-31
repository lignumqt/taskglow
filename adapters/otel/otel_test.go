package otel_test

import (
	"context"
	"errors"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	taskglowotel "github.com/lignumqt/taskglow/adapters/otel"
	"github.com/lignumqt/taskglow/taskglow"
)

// newTestTP creates an in-memory tracing provider and returns it together with
// the span recorder so tests can inspect captured spans.
func newTestTP() (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	return tp, sr
}

func TestRunnerCreatesSpan(t *testing.T) {
	tp, sr := newTestTP()
	tracer := tp.Tracer("test")
	runner := taskglowotel.New(tracer)

	err := runner.Run(context.Background(), "my-task", func(ctx context.Context, _ *taskglow.Task) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name() != "my-task" {
		t.Errorf("span name = %q, want %q", spans[0].Name(), "my-task")
	}
}

func TestRunnerSpanStatusOnSuccess(t *testing.T) {
	tp, sr := newTestTP()
	runner := taskglowotel.New(tp.Tracer("test"))

	_ = runner.Run(context.Background(), "ok-task", func(_ context.Context, _ *taskglow.Task) error {
		return nil
	})

	spans := sr.Ended()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}
	span := spans[0]
	if span.Status().Code.String() != "Ok" {
		t.Errorf("span status = %q, want Ok", span.Status().Code.String())
	}
}

func TestRunnerSpanStatusOnError(t *testing.T) {
	tp, sr := newTestTP()
	runner := taskglowotel.New(tp.Tracer("test"))

	wantErr := errors.New("boom")
	err := runner.Run(context.Background(), "fail-task", func(_ context.Context, t *taskglow.Task) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wantErr, got %v", err)
	}

	spans := sr.Ended()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}
	span := spans[0]
	if span.Status().Code.String() != "Error" {
		t.Errorf("span status = %q, want Error", span.Status().Code.String())
	}
	if span.Status().Description != wantErr.Error() {
		t.Errorf("span status description = %q, want %q", span.Status().Description, wantErr.Error())
	}
}

func TestRunnerLogsBecomSpanEvents(t *testing.T) {
	tp, sr := newTestTP()
	runner := taskglowotel.New(tp.Tracer("test"))

	_ = runner.Run(context.Background(), "log-task", func(_ context.Context, task *taskglow.Task) error {
		task.Log("hello world")
		task.Warn("be careful")
		return nil
	})

	spans := sr.Ended()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}
	span := spans[0]
	events := span.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 span events, got %d: %v", len(events), events)
	}

	var foundLog, foundWarn bool
	for _, ev := range events {
		switch ev.Name {
		case "hello world":
			foundLog = true
		case "be careful":
			foundWarn = true
			for _, attr := range ev.Attributes {
				if string(attr.Key) == "warning" && attr.Value.AsBool() {
					foundWarn = true
				}
			}
		}
	}
	if !foundLog {
		t.Error("log event not found in span events")
	}
	if !foundWarn {
		t.Error("warn event not found in span events")
	}
}

func TestRunnerSpanAttributes(t *testing.T) {
	tp, sr := newTestTP()
	runner := taskglowotel.New(tp.Tracer("test"))

	_ = runner.Run(context.Background(), "attr-task", func(_ context.Context, _ *taskglow.Task) error {
		return nil
	})

	spans := sr.Ended()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}
	attrs := map[string]string{}
	for _, a := range spans[0].Attributes() {
		attrs[string(a.Key)] = a.Value.AsString()
	}
	if _, ok := attrs["task.state"]; !ok {
		t.Error("missing task.state attribute")
	}
	if _, ok := attrs["task.elapsed"]; !ok {
		t.Error("missing task.elapsed attribute")
	}
}

func TestGroupCreatesSpans(t *testing.T) {
	tp, sr := newTestTP()
	tracer := tp.Tracer("test")

	grp := taskglowotel.NewGroup(context.Background(), tracer)
	grp.Go("task-a", func(_ context.Context, _ *taskglow.Task) error { return nil })
	grp.Go("task-b", func(_ context.Context, _ *taskglow.Task) error { return nil })

	if err := grp.Wait(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := sr.Ended()
	// parent "group" span + 2 child spans
	if len(spans) < 3 {
		t.Fatalf("expected >= 3 spans, got %d", len(spans))
	}
	names := map[string]bool{}
	for _, s := range spans {
		names[s.Name()] = true
	}
	for _, want := range []string{"group", "task-a", "task-b"} {
		if !names[want] {
			t.Errorf("span %q not found; got %v", want, names)
		}
	}
}
