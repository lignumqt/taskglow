// Package throttle provides a simple rate limiter for render updates.
package throttle

import (
	"sync"
	"time"
)

// Throttle allows at most one action per interval.
// Subsequent calls within the interval are dropped, but the last dropped
// call is always flushed after the interval expires so no update is lost.
type Throttle struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	interval time.Duration
	last     time.Time
	pending  bool
	flush    func()
	timer    *time.Timer
}

// New creates a Throttle that calls flush at most once per interval.
func New(interval time.Duration, flush func()) *Throttle {
	return &Throttle{
		interval: interval,
		flush:    flush,
	}
}

// Trigger notifies the throttle that an update is available.
// If the interval has elapsed since the last flush, flush is called immediately
// (synchronously, under the internal lock). Otherwise, a single deferred flush
// is scheduled for the end of the current interval; duplicate calls within the
// same interval are coalesced.
func (t *Throttle) Trigger() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	if now.Sub(t.last) >= t.interval {
		// Enough time has elapsed: fire directly.
		t.last = now
		t.pending = false
		if t.timer != nil {
			// Cancel the pending deferred timer. Because wg.Add(1) was called
			// when the timer was created, we must call wg.Done() here if the
			// timer was successfully stopped before the goroutine ran.
			if stopped := t.timer.Stop(); stopped {
				t.wg.Done()
			}
			// If stopped == false the goroutine is already queued/running and
			// will call wg.Done() itself via defer.
			t.timer = nil
		}
		t.flush()
		return
	}

	if !t.pending {
		t.pending = true
		remaining := t.interval - now.Sub(t.last)
		t.wg.Add(1)
		t.timer = time.AfterFunc(remaining, func() {
			defer t.wg.Done()
			t.mu.Lock()
			t.last = time.Now()
			t.pending = false
			t.timer = nil
			fn := t.flush
			t.mu.Unlock()
			fn()
		})
	}
}

// Stop cancels any pending deferred flush and waits for any in-flight flush
// to complete before returning. After Stop returns it is safe to read data
// that the flush function was writing to.
func (t *Throttle) Stop() {
	t.mu.Lock()
	if t.timer != nil {
		if stopped := t.timer.Stop(); stopped {
			// Timer cancelled before the goroutine ran; balance the wg.Add(1).
			t.wg.Done()
		}
		// If stopped == false the goroutine is already queued/running; it will
		// call wg.Done() itself. wg.Wait() below will block until it finishes.
		t.timer = nil
		t.pending = false
	}
	t.mu.Unlock()
	// Wait for any in-flight flush goroutine to complete.
	t.wg.Wait()
}
