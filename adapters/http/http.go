// Package http provides middleware and helpers for integrating taskglow with
// net/http long-running handlers.
package http

import (
	"context"
	"net/http"
	"time"

	"github.com/lignumqt/taskglow/taskglow"
)

// HandlerFunc is the signature of an HTTP handler that uses taskglow.
type HandlerFunc func(ctx context.Context, task *taskglow.Task, w http.ResponseWriter, r *http.Request) error

// Handler wraps a HandlerFunc in a taskglow session and returns an http.HandlerFunc.
// Errors returned by fn are written as 500 responses; context cancellations as 499.
func Handler(title string, fn HandlerFunc, opts ...taskglow.Option) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runner := taskglow.New(opts...)
		err := runner.Run(r.Context(), title, func(ctx context.Context, task *taskglow.Task) error {
			return fn(ctx, task, w, r)
		})

		if err != nil {
			if r.Context().Err() != nil {
				// Client disconnected.
				http.Error(w, "request canceled", 499)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// Middleware wraps an existing http.Handler with request-scoped taskglow sessions.
// Each request gets a separate session titled with the request method and path.
func Middleware(opts ...taskglow.Option) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			title := r.Method + " " + r.URL.Path
			runner := taskglow.New(opts...)
			_ = runner.Run(r.Context(), title, func(ctx context.Context, _ *taskglow.Task) error {
				next.ServeHTTP(w, r.WithContext(ctx))
				return nil
			})
		})
	}
}

// TimeoutHandler wraps a handler with a per-request timeout.
func TimeoutHandler(d time.Duration, title string, fn HandlerFunc, opts ...taskglow.Option) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		runner := taskglow.New(opts...)
		err := runner.Run(ctx, title, func(ctx context.Context, task *taskglow.Task) error {
			return fn(ctx, task, w, r)
		})
		if err != nil {
			if ctx.Err() != nil {
				http.Error(w, "handler timeout", http.StatusGatewayTimeout)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
