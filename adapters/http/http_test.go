package http_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/lignumqt/taskglow/adapters/http"
	"github.com/lignumqt/taskglow/taskglow"
)

var plainOpts = []taskglow.Option{
	taskglow.WithWriter(&bytes.Buffer{}),
	taskglow.WithMode(taskglow.ModePlain),
}

func TestHandler_Success(t *testing.T) {
	h := httpadapter.Handler("test", func(_ context.Context, task *taskglow.Task, w http.ResponseWriter, _ *http.Request) error {
		task.Log("handling request")
		w.WriteHeader(http.StatusOK)
		return nil
	}, plainOpts...)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_Error(t *testing.T) {
	h := httpadapter.Handler("test", func(_ context.Context, _ *taskglow.Task, _ http.ResponseWriter, _ *http.Request) error {
		return fmt.Errorf("internal error")
	}, plainOpts...)

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	handler := httpadapter.Middleware(plainOpts...)(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rec.Code)
	}
}

func TestTimeoutHandler_Timeout(t *testing.T) {
	h := httpadapter.TimeoutHandler(1*time.Millisecond, "slow", func(ctx context.Context, _ *taskglow.Task, _ http.ResponseWriter, _ *http.Request) error {
		<-ctx.Done()
		return ctx.Err()
	}, plainOpts...)

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504, got %d (body: %s)", rec.Code, strings.TrimSpace(rec.Body.String()))
	}
}
