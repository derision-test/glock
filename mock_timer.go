package glock

import (
	"time"
)

func sendTime(t *MockTimer, now time.Time) {
	t.ch <- now
}

// MockTimer is an implementation of Timer that can be moved forward in time
// in increments for testing code that relies on timeouts or other time-sensitive
// constructs.
type MockTimer struct {
	*advanceable
	deadline time.Time
	ch       chan time.Time
	stopped  bool
	// fired records whether the current arming has already elapsed and had its
	// signal delivered. It gates the (deadline-reached) trigger so a single
	// arming fires exactly once, and lets Stop/Reset report whether the timer
	// was still running (see the time.Timer semantics they mirror).
	fired bool
	// firing is set while the process goroutine has released the lock to
	// deliver a signal on ch (or invoke the AfterFunc). It exists so a
	// concurrent Stop/Reset/BlockingAdvance can tell an in-flight delivery is
	// underway even though the lock is momentarily free.
	firing bool
	f      func(*MockTimer, time.Time)
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
	return newMockTimerAt(c.advanceable, duration, func(mt *MockTimer, _ time.Time) {
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
	f func(*MockTimer, time.Time),
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

	// A single, long-lived process goroutine owns delivery for this timer for
	// its whole lifetime. Reset re-arms it by clearing state and broadcasting
	// rather than spawning another goroutine (which would leak goroutines and
	// race multiple deliverers on the same channel).
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

	// The timer was "running" iff it had neither been stopped nor already
	// fired. An in-flight delivery (firing) still counts as running.
	wasRunning := !t.stopped && !t.fired

	t.deadline = t.now.Add(duration)
	t.stopped = false
	t.fired = false

	// Re-register with the advanceable so a future Advance re-notifies this
	// timer. register is idempotent enough for our purposes: a timer only
	// remains subscribed while signal() reports interest, and an expired timer
	// drops itself, so a re-armed timer must re-subscribe.
	t.advanceable.registerLocked(t)

	// Wake the long-lived process goroutine to re-evaluate the (possibly
	// already-passed) new deadline.
	t.cond.Broadcast()

	return wasRunning
}

// Stop will stop the Timer from running.
func (t *MockTimer) Stop() bool {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	// Stop reports false when the timer has already fired or was already
	// stopped, matching time.Timer.Stop.
	if t.stopped || t.fired {
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
	t.cond.L.Lock()
	t.now = t.now.Add(duration)

	// If this advance does not cross the deadline (or the timer is not running),
	// there is nothing to deliver: release and return without blocking.
	if t.stopped || t.fired || t.now.Before(t.deadline) {
		t.cond.L.Unlock()
		return
	}

	// The deadline is crossed. Mark the arming as fired and deliver the signal
	// directly here (releasing the lock around the delivery) so the caller
	// blocks until a consumer reads the channel — matching the documented
	// BlockingAdvance contract. The AfterFunc variant does not write to the
	// channel, so it does not block the caller.
	t.fired = true
	t.cond.Broadcast()
	t.deliver()

	t.cond.L.Unlock()
}

// deliver invokes the timer's trigger function with the lock released, then
// re-acquires it. It must be called with t.cond.L held; on return the lock is
// held again. Releasing the lock around t.f is what prevents a deadlock: t.f
// (sendTime) blocks on an unbuffered channel send, and a concurrent
// Stop/Reset/Chan-drain must be able to take the lock while that send is
// pending.
func (t *MockTimer) deliver() {
	// Snapshot the value to deliver while the lock is still held; t.f runs with
	// the lock released, so it must not touch shared fields (t.now is written by
	// Advance under the same lock).
	now := t.now
	t.firing = true
	t.cond.L.Unlock()
	t.f(t, now)
	t.cond.L.Lock()
	t.firing = false
}

func (t *MockTimer) process() {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	for {
		switch {
		case t.stopped:
			// Park until Reset re-arms us (or the process is abandoned when the
			// clock and all references are dropped).
			t.cond.Wait()

		case !t.fired && !t.now.Before(t.deadline):
			// Deadline reached and not yet delivered for this arming. Mark fired
			// first so Stop/Reset observe consistent state, then deliver with the
			// lock released. If a concurrent Reset re-armed us during delivery,
			// t.fired is false again on return and we loop to re-evaluate.
			t.fired = true
			t.cond.Broadcast()
			t.deliver()

		default:
			// Either waiting for the deadline, or already fired and waiting for a
			// Reset. Sleep until state changes.
			t.cond.Wait()
		}
	}
}

// signal conforms to the subscriber interface.
func (t *MockTimer) signal(now time.Time) bool {
	// setCurrent (the caller) updates now and broadcasts under the lock, which
	// wakes the process goroutine to re-check the deadline. Remain subscribed
	// only while the timer is still live (not stopped and not already fired for
	// the current arming); Reset re-subscribes on re-arm.
	return !t.stopped && !t.fired
}
