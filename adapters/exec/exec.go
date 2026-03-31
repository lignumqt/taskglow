// Package exec provides helpers for running external commands with taskglow progress reporting.
package exec

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/lignumqt/taskglow/taskglow"
)

// Result holds the output from a completed command.
type Result struct {
	// Stdout contains all standard output lines.
	Stdout []string
	// Stderr contains all standard error lines.
	Stderr []string
	// ExitCode is the process exit code (0 on success).
	ExitCode int
}

// Run executes the named program with args inside a taskglow session.
// stdout and stderr lines are streamed as log messages to the task.
func Run(ctx context.Context, task *taskglow.Task, name string, args ...string) (*Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", name, err)
	}

	res := &Result{}
	doneCh := make(chan struct{}, 2)

	readLines := func(r io.Reader, dest *[]string) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			*dest = append(*dest, line)
			task.Log(line)
		}
		doneCh <- struct{}{}
	}

	go readLines(stdoutPipe, &res.Stdout)
	go readLines(stderrPipe, &res.Stderr)

	<-doneCh
	<-doneCh

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			return res, fmt.Errorf("%s exited with code %d: %s",
				name, res.ExitCode, strings.TrimSpace(strings.Join(res.Stderr, "\n")))
		}
		return res, fmt.Errorf("wait %s: %w", name, err)
	}
	return res, nil
}
