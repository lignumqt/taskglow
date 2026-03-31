package render_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lignumqt/taskglow/render"
)

func TestTTYStart(t *testing.T) {
	var buf bytes.Buffer
	r := render.NewTTY(&buf, nil, 20*time.Millisecond, true, 80, false)
	r.Start("Test task")
	time.Sleep(60 * time.Millisecond)
	r.Stop()

	if buf.Len() == 0 {
		t.Error("TTY renderer wrote nothing after Start")
	}
}

func TestTTYProgress(t *testing.T) {
	var buf bytes.Buffer
	r := render.NewTTY(&buf, nil, 20*time.Millisecond, true, 80, false)
	r.Start("Uploading")
	r.Update(render.Event{Kind: render.EventProgress, Progress: 0.42, Message: "uploading"})
	time.Sleep(60 * time.Millisecond)
	r.Stop()

	out := buf.String()
	if !strings.Contains(out, "42") {
		t.Errorf("TTY output missing progress percentage, got: %q", out)
	}
}

func TestTTYLog(t *testing.T) {
	var buf bytes.Buffer
	r := render.NewTTY(&buf, nil, 20*time.Millisecond, true, 80, false)
	r.Start("Task")
	r.Update(render.Event{Kind: render.EventLog, Message: "hello from log"})
	time.Sleep(60 * time.Millisecond)
	r.Stop()

	if !strings.Contains(buf.String(), "hello from log") {
		t.Errorf("TTY output missing log line")
	}
}

func TestTTYFinishSucceeded(t *testing.T) {
	var buf bytes.Buffer
	r := render.NewTTY(&buf, nil, 20*time.Millisecond, true, 80, false)
	r.Start("Build")
	r.Finish(render.FinishState{
		Title:     "Build",
		StateCode: 4,
		StateStr:  "succeeded",
		Elapsed:   "2.5s",
	})

	if !strings.Contains(buf.String(), "Build") {
		t.Errorf("finish output missing title")
	}
}

func TestTTYFinishFailed(t *testing.T) {
	var buf bytes.Buffer
	r := render.NewTTY(&buf, nil, 20*time.Millisecond, true, 80, false)
	r.Start("Deploy")
	r.Finish(render.FinishState{
		Title:     "Deploy",
		StateCode: 5,
		StateStr:  "failed",
		Elapsed:   "0.8s",
		Err:       errors.New("exit code 1"),
	})

	if !strings.Contains(buf.String(), "exit code 1") {
		t.Errorf("finish failed output missing error")
	}
}

func TestTTYConcurrentUpdates(t *testing.T) {
	var buf bytes.Buffer
	r := render.NewTTY(&buf, nil, 10*time.Millisecond, true, 80, false)
	r.Start("concurrent")

	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func() {
			r.Update(render.Event{Kind: render.EventLog, Message: "msg"})
			done <- struct{}{}
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
	r.Stop()
}

func TestTTYStopIdempotent(t *testing.T) {
	var buf bytes.Buffer
	r := render.NewTTY(&buf, nil, 20*time.Millisecond, true, 80, false)
	r.Start("task")
	r.Stop()
	r.Stop()
}

func TestTTYVerbosePreservesLogs(t *testing.T) {
	var buf bytes.Buffer
	r := render.NewTTY(&buf, nil, 20*time.Millisecond, true, 80, true)
	r.Start("Build")
	r.Update(render.Event{Kind: render.EventLog, Message: "compiled main.go"})
	r.Update(render.Event{Kind: render.EventLog, Message: "linked binary"})
	r.Update(render.Event{Kind: render.EventWarn, Message: "deprecated API"})
	time.Sleep(40 * time.Millisecond)
	r.Finish(render.FinishState{
		Title:     "Build",
		StateCode: 4,
		StateStr:  "succeeded",
		Elapsed:   "1.0s",
		Logs:      []string{"compiled main.go", "linked binary"},
		Warnings:  []string{"deprecated API"},
	})

	out := buf.String()
	if !strings.Contains(out, "compiled main.go") {
		t.Errorf("verbose: expected log line 'compiled main.go' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "linked binary") {
		t.Errorf("verbose: expected log line 'linked binary' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "deprecated API") {
		t.Errorf("verbose: expected warning line 'deprecated API' in output, got:\n%s", out)
	}
}
