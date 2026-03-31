package throttle_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/lignumqt/taskglow/internal/throttle"
)

func TestThrottleImmediate(t *testing.T) {
	var count int64
	th := throttle.New(50*time.Millisecond, func() { atomic.AddInt64(&count, 1) })

	th.Trigger()
	if c := atomic.LoadInt64(&count); c != 1 {
		t.Errorf("first Trigger: count = %d; want 1", c)
	}
}

func TestThrottleDrop(t *testing.T) {
	var count int64
	th := throttle.New(200*time.Millisecond, func() { atomic.AddInt64(&count, 1) })

	th.Trigger()
	th.Trigger()
	th.Trigger()

	if c := atomic.LoadInt64(&count); c != 1 {
		t.Errorf("after 3 rapid Triggers: count = %d; want 1", c)
	}

	time.Sleep(300 * time.Millisecond)

	if c := atomic.LoadInt64(&count); c != 2 {
		t.Errorf("after deferred flush: count = %d; want 2", c)
	}
	th.Stop()
}

func TestThrottleStop(t *testing.T) {
	var count int64
	th := throttle.New(100*time.Millisecond, func() { atomic.AddInt64(&count, 1) })

	th.Trigger()
	th.Trigger()
	th.Stop()

	time.Sleep(150 * time.Millisecond)

	if c := atomic.LoadInt64(&count); c != 1 {
		t.Errorf("after Stop: count = %d; want 1 (deferred canceled)", c)
	}
}
