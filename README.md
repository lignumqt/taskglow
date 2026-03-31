# TaskGlow ✨

[![Go Reference](https://pkg.go.dev/badge/github.com/lignumqt/taskglow.svg)](https://pkg.go.dev/github.com/lignumqt/taskglow)
[![CI](https://github.com/lignumqt/taskglow/actions/workflows/ci.yml/badge.svg)](https://github.com/lignumqt/taskglow/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lignumqt/taskglow)](https://goreportcard.com/report/github.com/lignumqt/taskglow)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**TaskGlow** is a Go library that wraps long-running operations with automatic spinner/progress/log rendering in TTY terminals and graceful plain-text fallback in CI environments and pipes.

```
» Deploying application
  ⠿  Connecting to server...
  [████████░░░░░░░░░░░░]  40%  Uploading build artifacts
  ✓ Deploying application  1.5s
```

## Features

- **One-call integration** — `taskglow.Wrap(ctx, "title", fn)` is all you need
- **Auto-detection** — spinner + progress bar in TTY; timestamped plain lines in CI/pipes
- **Adapters** — drop-in wrappers for Cobra commands, `os/exec` subprocess streaming, and `net/http` handlers
- **Race-free** — tested with `-race`; goroutine lifecycle is deterministic
- **Zero heavy dependencies** — only `golang.org/x/term` for terminal detection

## Installation

```sh
go get github.com/lignumqt/taskglow
```

Requires Go 1.22+.

## Quick Start

```go
package main

import (
    "context"
    "time"

    "github.com/lignumqt/taskglow"
)

func main() {
    _ = taskglow.Wrap(context.Background(), "Building project", func(ctx context.Context, t *taskglow.Task) error {
        t.Log("Compiling sources...")
        time.Sleep(300 * time.Millisecond)

        t.Progress(0.5, "Linking")
        time.Sleep(300 * time.Millisecond)

        t.Progress(1.0, "Done")
        return nil
    })
}
```

## Task API

| Method | Description |
|---|---|
| `t.Log(msg)` | Append an informational log line |
| `t.Logf(format, args...)` | Formatted log line |
| `t.Warn(msg)` | Append a warning (shows in yellow; marks task as warned) |
| `t.Warnf(format, args...)` | Formatted warning |
| `t.Stage(name, current, total)` | Announce the current pipeline stage |
| `t.Progress(pct, msg)` | Update progress bar (0.0–1.0) with optional message |
| `t.SetHint(msg)` | Set a hint printed on failure |
| `t.Succeed()` / `t.Fail(err)` / `t.Cancel()` | Force a terminal state early |

## Runner Options

```go
runner := taskglow.New(
    taskglow.WithWriter(os.Stderr),          // output destination
    taskglow.WithMode(taskglow.ModePlain),   // force plain mode
    taskglow.WithRenderInterval(100*time.Millisecond),
    taskglow.WithSpinnerFrames([]string{"◐","◓","◑","◒"}),
    taskglow.WithNoColor(true),
    taskglow.WithSummary(true),             // print elapsed summary    taskglow.WithLogFile(logFile),          // write all logs to a file
    taskglow.WithOnFinish(func(s taskglow.Summary) { /* metrics, alerts */ }),
    taskglow.WithOnLog(func(msg string) { /* forward to slog */ }),
    taskglow.WithOnWarn(func(msg string) { /* forward to slog */ }),)
```

### Modes

| Constant | Description |
|---|---|
| `ModeAuto` | (default) TTY → spinner; otherwise → plain |
| `ModeTTY` | Force interactive renderer |
| `ModePlain` | Force plain-text renderer |
| `ModeQuiet` | Suppress all output |
| `ModeVerbose` | Spinner during execution; all logs and warnings remain on screen after finish |

`ModeVerbose` is useful for CLI tools that support a `--verbose` flag. Add it to any `cobra` command:

```go
var verbose bool
// in init(): cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "keep logs on screen after finish")

mode := taskglow.ModeAuto
if verbose {
    mode = taskglow.ModeVerbose
}

err := taskglow.New(taskglow.WithMode(mode)).Run(ctx, "Deploy", fn)
```

## Parallel Tasks (Group)

`Group` runs multiple tasks **concurrently**, rendering them as a live stacked
list of spinner rows in TTY or safely interleaved plain lines in CI.

```go
g := taskglow.NewGroup(ctx)
g.Go("Build frontend", buildFn)
g.Go("Build backend",  compileFn)
g.Go("Run migrations", migrateFn)

if err := g.Wait(); err != nil {
    log.Fatal(err)
}

// Per-task summaries available afterwards:
for _, r := range g.Results() {
    fmt.Printf("%s: %s\n", r.Title, r.Summary.State)
}
```

All `Option` values accepted by `New` (modes, hooks, log file) also work with `NewGroup`.

## Adapters

### Cobra

```go
import cobraadapter "github.com/lignumqt/taskglow/adapters/cobra"

cmd := &cobra.Command{
    RunE: cobraadapter.RunE("Deploying", func(ctx context.Context, t *taskglow.Task, cmd *cobra.Command, args []string) error {
        // ...
        return nil
    }),
}
```

### os/exec

```go
import execadapter "github.com/lignumqt/taskglow/adapters/exec"

_ = taskglow.Wrap(ctx, "Running tests", func(ctx context.Context, t *taskglow.Task) error {
    result, err := execadapter.Run(ctx, t, "go", "test", "./...")
    if err != nil {
        return err
    }
    t.Log(fmt.Sprintf("exit code: %d", result.ExitCode))
    return nil
})
```

### net/http

```go
import httpadapter "github.com/lignumqt/taskglow/adapters/http"

mux.HandleFunc("/api/deploy", httpadapter.Handler("API deploy",
    func(ctx context.Context, t *taskglow.Task, w http.ResponseWriter, r *http.Request) error {
        // ...
        return nil
    },
))

// Or wrap an existing handler with middleware:
http.Handle("/", httpadapter.Middleware()(existingHandler))
```

### OpenTelemetry

Each `Run` call opens a tracing span. Log lines become span events, warnings
become events with a `warning=true` attribute, and the span status reflects the
final task outcome. The span context is propagated into the task function so
child spans can be created normally.

```go
import (
    taskglowotel "github.com/lignumqt/taskglow/adapters/otel"
    "go.opentelemetry.io/otel"
)

tracer := otel.Tracer("my-service")
runner := taskglowotel.New(tracer, taskglow.WithMode(taskglow.ModeAuto))

err := runner.Run(ctx, "Deploy", func(ctx context.Context, t *taskglow.Task) error {
    t.Log("preparing release")
    // ctx carries the span — child spans nest automatically
    return deploy(ctx)
})
```

Parallel tasks via `Group`:

```go
grp := taskglowotel.NewGroup(ctx, tracer)
grp.Go("Build",  buildFn)
grp.Go("Test",   testFn)
err := grp.Wait()  // parent "group" span + one child span per task
```

> **Note:** do not pass `WithOnLog`, `WithOnWarn` or `WithOnFinish` in opts
> when using the OTel adapter — those hooks are used internally.

## Examples

```sh
go run ./examples/basic/...     # minimal Wrap() usage
go run ./examples/progress/...  # progress bar + stages
go run ./examples/stages/...    # multi-step pipeline
go run ./examples/verbose/...   # ModeVerbose: logs stay on screen after finish
go run ./examples/group/...     # parallel tasks with Group
go run ./examples/otel/...      # OpenTelemetry spans (stdout exporter)
```

## Development

```sh
make test       # go test -race -count=1 ./...
make lint       # golangci-lint run
make vet        # go vet ./...
make build      # go build ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## License

[MIT](LICENSE) © lignumqt
