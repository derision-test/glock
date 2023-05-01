package glock

import (
	"time"
)

func sendTime(t *MockTimer) {
	t.ch <- t.now
}

// MockTimer is an implementation of Timer that can be moved forward in time
// in increments for testing code that relies on timeouts or other time-sensitive
// constructs.
type MockTimer struct {
	*advanceable
	deadline time.Time
	ch       chan time.Time
	stopped  bool
	f        func(*MockTimer)
}

var _ Timer = &MockTimer{}
var _ Advanceable = &MockTimer{}

// NewTimer creates a new Timer tied to the internal MockClock time that functions
// similar to time.NewTimer().
func (c *MockClock) NewTimer(duration time.Duration) Timer {
	return newMockTimerAt(c.advanceable, duration, sendTime)
}

// AfterFunc creates a new Timer tied to the internal MockClock time that functions
// similar to time.AfterFunc().
func (c *MockClock) AfterFunc(duration time.Duration, f func()) Timer {
	return newMockTimerAt(c.advanceable, duration, func(mt *MockTimer) {
		go f()
	})
}

// NewMockTimer creates a new MockTimer with the internal time set to time.Now().
func NewMockTimer(duration time.Duration) *MockTimer {
	return NewMockTimerAt(time.Now(), duration)
}

// NewMockTimerAt creates a new MockTimer with the internal time set to the given time.
func NewMockTimerAt(now time.Time, duration time.Duration) *MockTimer {
	return newMockTimerAt(newAdvanceableAt(now), duration, sendTime)
}

func newMockTimerAt(
	advanceable *advanceable,
	duration time.Duration,
	f func(*MockTimer),
) *MockTimer {
	if duration == 0 {
		panic("duration cannot be 0")
	}

	t := &MockTimer{
		advanceable: advanceable,
		deadline:    advanceable.now.Add(duration),
		ch:          make(chan time.Time),
		f:           f,
	}

	go t.process()
	advanceable.register(t)

	return t
}

// Chan returns a channel which will receive the Timer's internal time at the
// point where the Timer is triggered.
func (t *MockTimer) Chan() <-chan time.Time {
	return t.ch
}

// Reset will reset the deadline of the Timer to trigger after the new duration
// based on the Timer's internal current time. If the Timer was running when Reset
// was called it will return true.
func (t *MockTimer) Reset(duration time.Duration) bool {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	wasRunning := !t.stopped

	t.deadline = t.now.Add(duration)
	t.stopped = false

	if !wasRunning {
		go t.process()
		t.advanceable.register(t)
	}

	t.cond.Broadcast()

	return wasRunning
}

// Stop will stop the Timer from running.
func (t *MockTimer) Stop() bool {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	if t.stopped {
		return false
	}

	t.stopped = true
	t.cond.Broadcast()

	return true
}

// BlockingAdvance will bump the timer's internal time by the given duration. If
// the new internal time passes the timer's trigger threshold, a signal will be sent.
// This method will not return until the signal is read by a consumer of the Timer's
// channel.
func (t *MockTimer) BlockingAdvance(duration time.Duration) {
	t.m.Lock()
	defer t.m.Unlock()

	t.now = t.now.Add(duration)

	t.tryExecute()
}

func (t *MockTimer) tryExecute() {
	if !t.now.Before(t.deadline) {
		t.stopped = true
		t.cond.Broadcast()

		t.f(t)
	}
}

func (t *MockTimer) process() {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	for !t.stopped {
		t.tryExecute()
		t.cond.Wait()
	}

}

// signal conforms to the subscriber interface.
func (t *MockTimer) signal(now time.Time) bool {
	return !t.stopped
}
