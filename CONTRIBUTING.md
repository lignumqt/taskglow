# Contributing to TaskGlow

Thank you for your interest in contributing!

## Getting Started

1. Fork the repository and clone your fork.
2. Install dependencies:
   ```sh
   go mod download
   ```
3. Run tests to confirm everything works:
   ```sh
   make test
   ```

## Development Workflow

```sh
make build   # compile all packages
make test    # go test -race -count=1 ./...
make vet     # go vet ./...
make lint    # golangci-lint run (requires golangci-lint)
```

Install golangci-lint:
```sh
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Code Style

- Follow standard Go conventions (`gofmt`, `goimports`).
- New exported symbols must have doc comments.
- All new code must be covered by tests.
- Tests must pass with `-race`.

## Pull Requests

- Keep PRs focused and small.
- Add an entry to `CHANGELOG.md` under `[Unreleased]`.
- Describe *what* and *why* in the PR description.

## Reporting Issues

Use the GitHub issue templates:
- **Bug report** — unexpected behaviour with a minimal reproduction.
- **Feature request** — describe the use case and desired outcome.

## License

By contributing you agree that your contributions will be licensed under the [MIT License](LICENSE).
