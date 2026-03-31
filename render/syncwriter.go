package render

import (
	"io"
	"sync"
)

// SyncWriter wraps an io.Writer with a mutex, making it safe for concurrent use.
type SyncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// NewSyncWriter returns a SyncWriter wrapping w.
func NewSyncWriter(w io.Writer) *SyncWriter { return &SyncWriter{w: w} }

func (s *SyncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
