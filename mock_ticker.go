package glock

import (
	"time"
)

// MockTicker is an implementation of Ticker that can be moved forward in time
// in increments for testing code that relies on timeouts or other time-sensitive
// constructs.
type MockTicker struct {
	*advanceable
	duration time.Duration
	deadline time.Time
	ch       chan time.Time
	stopped  bool
	// firing is set while the process goroutine has released the lock to deliver
	// a tick on ch. It exists so a concurrent Stop/BlockingAdvance can observe
	// that a delivery is in flight even though the lock is momentarily free.
	firing bool
}

var _ Ticker = &MockTicker{}
var _ Advanceable = &MockTicker{}

// NewTicker creates a new Ticker tied to the internal MockClock time that ticks
// at intervals similar to time.NewTicker().  It will also skip or drop ticks
// for slow readers similar to time.NewTicker() as well.
func (c *MockClock) NewTicker(duration time.Duration) Ticker {
	// Record the ticker args under the lock, then release it before
	// constructing the ticker: newMockTickerAt -> register acquires the same
	// mutex (c.m and the advanceable mutex are one and the same), and the mutex
	// is not reentrant.
	c.m.Lock()
	c.tickerArgs = append(c.tickerArgs, duration)
	c.m.Unlock()

	return newMockTickerAt(c.advanceable, duration)
}

// NewMockTicker creates a new MockTicker with the internal time set to time.Now().
func NewMockTicker(duration time.Duration) *MockTicker {
	return NewMockTickerAt(time.Now(), duration)
}

// NewMockTickerAt creates a new MockTicker with the internal time set to the given time.
func NewMockTickerAt(now time.Time, duration time.Duration) *MockTicker {
	return newMockTickerAt(newAdvanceableAt(now), duration)
}

func newMockTickerAt(advanceable *advanceable, duration time.Duration) *MockTicker {
	if duration == 0 {
		panic("duration cannot be 0")
	}

	t := &MockTicker{
		advanceable: advanceable,
		duration:    duration,
		deadline:    advanceable.now.Add(duration),
		ch:          make(chan time.Time),
	}

	go t.process()
	advanceable.register(t)
	return t
}

// Chan returns a channel which will receive the tickers's internal time at the
// interval given when creating the ticker.
func (t *MockTicker) Chan() <-chan time.Time {
	return t.ch
}

// Stop will stop the ticker from ticking.
func (t *MockTicker) Stop() {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	t.stopped = true
	t.cond.Broadcast()
}

// BlockingAdvance will bump the ticker's internal time by the given duration. If
// the new internal time passes the next tick threshold, a signal will be sent.
// This method will not return until the signal is read by a consumer of the
// ticker.
func (t *MockTicker) BlockingAdvance(duration time.Duration) {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	t.now = t.now.Add(duration)

	if t.stopped || t.now.Before(t.deadline) {
		return
	}

	// Deliver the scheduled tick with the lock released so a concurrent Stop can
	// still make progress, then drop any ticks that elapsed by the fire-time now
	// (matching time.Ticker's slow-reader behavior). Advancement is relative to
	// firedNow, not the live t.now, so a concurrent Advance during delivery does
	// not cause us to skip its tick.
	firedNow := t.now
	t.deliver(t.deadline)
	t.advanceDeadline(firedNow)
}

// deliver sends the given tick value on the channel with the lock released, then
// re-acquires it. It must be called with t.cond.L held; on return the lock is
// held again. Releasing the lock around the send is what prevents a deadlock:
// the send blocks on an unbuffered channel, and a concurrent Stop or channel
// consumer must be able to take the lock while that send is pending.
func (t *MockTicker) deliver(tick time.Time) {
	t.firing = true
	t.cond.L.Unlock()
	t.ch <- tick
	t.cond.L.Lock()
	t.firing = false
}

// advanceDeadline rolls the deadline forward past the given reference time,
// dropping any intermediate ticks. Must be called with the lock held. The
// reference time is passed explicitly (rather than read from t.now) so it
// reflects the instant the just-delivered tick fired, even if t.now has moved
// on while the delivery briefly released the lock.
func (t *MockTicker) advanceDeadline(ref time.Time) {
	for !ref.Before(t.deadline) {
		t.deadline = t.deadline.Add(t.duration)
	}
}

func (t *MockTicker) process() {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	for !t.stopped {
		if !t.now.Before(t.deadline) {
			// Snapshot the tick value and the fire-time now under the lock,
			// deliver with the lock released, then advance past any ticks dropped
			// for a slow reader relative to the fire-time now. A concurrent Stop
			// taken during delivery is observed on the next loop iteration.
			tick := t.deadline
			firedNow := t.now
			t.deliver(tick)
			if t.stopped {
				return
			}
			t.advanceDeadline(firedNow)
			continue
		}

		t.cond.Wait()
	}
}

// signal conforms to the subscriber interface.
func (t *MockTicker) signal(now time.Time) (requeue bool) {
	// setCurrent (the caller) updates now and broadcasts under the lock, which
	// wakes the process goroutine to re-check the deadline. We only report
	// whether we remain interested in future updates.
	return !t.stopped
}
