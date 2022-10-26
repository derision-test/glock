package glock

import (
	"time"
)

func sendTime(t *MockTimer) {
	t.ch <- t.now
}

type MockTimer struct {
	*advanceable
	deadline time.Time
	ch       chan time.Time
	stopped  bool
	f        func(*MockTimer)
}

var _ Timer = &MockTimer{}
var _ Advanceable = &MockTimer{}

func (c *MockClock) NewTimer(duration time.Duration) Timer {
	return newMockTimerAt(c.advanceable, duration, sendTime)
}

func (c *MockClock) AfterFunc(duration time.Duration, f func()) Timer {
	return newMockTimerAt(c.advanceable, duration, func(mt *MockTimer) {
		go f()
	})
}

func NewMockTimer(duration time.Duration) *MockTimer {
	return NewMockTimerAt(time.Now(), duration)
}

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

func (t *MockTimer) Chan() <-chan time.Time {
	return t.ch
}

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

func (t *MockTimer) BlockingAdvance(duration time.Duration) {
	t.m.Lock()
	defer t.m.Unlock()

	t.now = t.now.Add(duration)

	t.tryExecute()
	// TODO Test this
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

func (t *MockTimer) signal(now time.Time) bool {
	return !t.stopped
}
