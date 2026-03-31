# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-01-01

### Added

- `taskglow.Wrap` / `taskglow.New` / `Runner.Run` public API
- `Task` handle: `Log`, `Warn`, `Stage`, `Progress`, `SetHint`, `Succeed`, `Fail`, `Cancel`
- Auto-detection of TTY vs plain mode via `golang.org/x/term`
- TTY renderer with braille spinner, dynamic progress bar, live log area, and ANSI colors
- Plain renderer for CI/pipe environments (timestamped lines)
- Quiet (`ModeQuiet`) and verbose (`ModeVerbose`) modes
- Functional options: `WithWriter`, `WithMode`, `WithRenderInterval`, `WithSpinnerFrames`, `WithNoColor`, `WithSummary`
- `adapters/cobra` — generic `RunE` / `RunEContext` wrappers for Cobra commands
- `adapters/exec` — `os/exec` subprocess runner with stdout/stderr streaming as log lines
- `adapters/http` — `Handler`, `Middleware`, `TimeoutHandler` for `net/http`
- `internal/ansi` — ANSI escape code helpers
- `internal/terminal` — terminal/CI/NO_COLOR detection
- `internal/throttle` — rate-limiting render flush with race-safe shutdown
- Examples: `basic`, `progress`, `stages`
- Full test suite with `-race` coverage
- GitHub Actions CI workflow
- MIT License

[Unreleased]: https://github.com/lignumqt/taskglow/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/lignumqt/taskglow/releases/tag/v0.1.0
