.PHONY: all build test lint vet fmt tidy clean

GO      := go
GOFLAGS ?=
MODULE  := github.com/lignumqt/taskglow

all: vet lint test

## build: compile all packages (examples included)
build:
	$(GO) build $(GOFLAGS) ./...

## test: run all tests with race detector and coverage
test:
	$(GO) test -race -count=1 -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

## vet: run go vet
vet:
	$(GO) vet ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## fmt: format all Go files
fmt:
	$(GO) fmt ./...
	gofmt -s -w .

## tidy: tidy module dependencies
tidy:
	$(GO) mod tidy

## clean: remove build artifacts
clean:
	rm -f coverage.out coverage.html
	$(GO) clean ./...

## help: print this help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' |  sed -e 's/^/ /'
