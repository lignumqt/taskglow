// Example demonstrating the OpenTelemetry adapter for taskglow.
//
// It sets up an in-process OTLP-stdout exporter so you can see the produced
// spans without running a collector.  In production you would replace the
// exporter with one that points at your OTel Collector / Jaeger / Tempo etc.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	taskglowotel "github.com/lignumqt/taskglow/adapters/otel"
	"github.com/lignumqt/taskglow/taskglow"
)

func main() {
	// ---------- OTel bootstrap ----------
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		fmt.Fprintln(os.Stderr, "otel exporter:", err)
		os.Exit(1)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	defer func() { _ = tp.Shutdown(context.Background()) }()
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("taskglow-example")

	// ---------- single task ----------
	runner := taskglowotel.New(tracer, taskglow.WithMode(taskglow.ModeAuto))

	ctx := context.Background()
	if err := runner.Run(ctx, "Build image", func(ctx context.Context, t *taskglow.Task) error {
		t.Log("pulling base layer")
		time.Sleep(100 * time.Millisecond)
		t.Progress(50, "compiling")
		time.Sleep(100 * time.Millisecond)
		t.Log("pushing to registry")
		return nil
	}); err != nil {
		fmt.Fprintln(os.Stderr, "task failed:", err)
	}

	// ---------- parallel group ----------
	grp := taskglowotel.NewGroup(ctx, tracer, taskglow.WithMode(taskglow.ModeAuto))
	grp.Go("Lint", func(_ context.Context, t *taskglow.Task) error {
		t.Log("running golangci-lint")
		time.Sleep(80 * time.Millisecond)
		return nil
	})
	grp.Go("Unit tests", func(_ context.Context, t *taskglow.Task) error {
		t.Log("go test ./...")
		time.Sleep(120 * time.Millisecond)
		t.Warn("1 test skipped")
		return nil
	})
	if err := grp.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, "group failed:", err)
	}
}
