package render_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/lignumqt/taskglow/render"
)

func TestPlainStart(t *testing.T) {
	var buf bytes.Buffer
	p := render.NewPlain(&buf)
	p.Start("Deploy service")

	if !strings.Contains(buf.String(), "Deploy service") {
		t.Errorf("Start output %q does not contain task title", buf.String())
	}
}

func TestPlainUpdateProgress(t *testing.T) {
	var buf bytes.Buffer
	p := render.NewPlain(&buf)
	p.Start("test")
	p.Update(render.Event{
		Kind:     render.EventProgress,
		Progress: 0.5,
		Message:  "halfway done",
	})

	out := buf.String()
	if !strings.Contains(out, "50") {
		t.Errorf("progress output %q missing percentage", out)
	}
	if !strings.Contains(out, "halfway done") {
		t.Errorf("progress output %q missing message", out)
	}
}

func TestPlainUpdateLog(t *testing.T) {
	var buf bytes.Buffer
	p := render.NewPlain(&buf)
	p.Start("test")
	p.Update(render.Event{Kind: render.EventLog, Message: "compiled 42 files"})

	if !strings.Contains(buf.String(), "compiled 42 files") {
		t.Errorf("log output missing message")
	}
}

func TestPlainUpdateWarn(t *testing.T) {
	var buf bytes.Buffer
	p := render.NewPlain(&buf)
	p.Start("test")
	p.Update(render.Event{Kind: render.EventWarn, Message: "low disk space"})

	if !strings.Contains(buf.String(), "low disk space") {
		t.Errorf("warn output missing message")
	}
}

func TestPlainUpdateStage(t *testing.T) {
	var buf bytes.Buffer
	p := render.NewPlain(&buf)
	p.Start("test")
	p.Update(render.Event{Kind: render.EventStage, Message: "Build", Current: 2, Total: 5})

	out := buf.String()
	if !strings.Contains(out, "2/5") {
		t.Errorf("stage output %q missing stage numbers", out)
	}
	if !strings.Contains(out, "Build") {
		t.Errorf("stage output %q missing stage name", out)
	}
}

func TestPlainFinishSucceeded(t *testing.T) {
	var buf bytes.Buffer
	p := render.NewPlain(&buf)
	p.Start("Deploy")
	p.Finish(render.FinishState{
		Title:     "Deploy",
		StateCode: 4,
		StateStr:  "succeeded",
		Elapsed:   "1.2s",
	})

	out := buf.String()
	if !strings.Contains(out, "\u2713") {
		t.Errorf("succeeded output %q missing checkmark", out)
	}
	if !strings.Contains(out, "1.2s") {
		t.Errorf("succeeded output %q missing elapsed", out)
	}
}

func TestPlainFinishFailed(t *testing.T) {
	var buf bytes.Buffer
	p := render.NewPlain(&buf)
	p.Start("Deploy")

	err := errors.New("connection refused")
	p.Finish(render.FinishState{
		Title:     "Deploy",
		StateCode: 5,
		StateStr:  "failed",
		Elapsed:   "0.5s",
		Err:       err,
		Hint:      "check network connectivity",
	})

	out := buf.String()
	if !strings.Contains(out, "\u2717") {
		t.Errorf("failed output %q missing failure marker", out)
	}
	if !strings.Contains(out, "connection refused") {
		t.Errorf("failed output %q missing error message", out)
	}
	if !strings.Contains(out, "check network connectivity") {
		t.Errorf("failed output %q missing hint", out)
	}
}

func TestPlainFinishCanceled(t *testing.T) {
	var buf bytes.Buffer
	p := render.NewPlain(&buf)
	p.Start("Deploy")
	p.Finish(render.FinishState{
		Title:     "Deploy",
		StateCode: 6,
		StateStr:  "canceled",
		Elapsed:   "0.1s",
	})

	if !strings.Contains(buf.String(), "canceled") {
		t.Errorf("canceled output missing canceled text")
	}
}

func TestPlainConcurrentUpdates(t *testing.T) {
	var buf bytes.Buffer
	p := render.NewPlain(&buf)
	p.Start("concurrent")

	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func() {
			p.Update(render.Event{Kind: render.EventLog, Message: "line"})
			done <- struct{}{}
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}
