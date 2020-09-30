package glock

import (
	"runtime"
	"sort"
	"sync"
	"time"
)

// Clock is a wrapper around common functions in the time package.
// This interface is designed to allow easy mocking of time functions.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// After returns a channel which receives the current time after
	// the given duration elapses.
	After(duration time.Duration) <-chan time.Time

	// Sleep blocks until the given duration elapses.
	Sleep(duration time.Duration)

	// Since returns the time elapsed since t.
	Since(t time.Time) time.Duration

	// Until returns the duration until t.
	Until(t time.Time) time.Duration

	// NewTicker will construct a ticker which will continually fire,
	// pausing for the given duration in between invocations.
	NewTicker(duration time.Duration) Ticker
}

// MockClock is an implementation of Clock that can be moved forward in time
// in increments for testing code that relies on timeouts or other time-sensitive
// constructs.
type MockClock struct {
	fakeTime   time.Time
	triggers   mockTriggers
	tickers    mockTickers
	afterArgs  []time.Duration
	tickerArgs []time.Duration
	nowLock    sync.RWMutex
	afterLock  sync.RWMutex
	tickerLock sync.Mutex
}

var _ Clock = &MockClock{}

type mockTrigger struct {
	trigger time.Time
	ch      chan time.Time
}

type mockTriggers []*mockTrigger
type mockTickers []*mockTicker

// NewMockClock creates a new MockClock with the internal time set
// to time.Now()
func NewMockClock() *MockClock {
	return NewMockClockAt(time.Now())
}

// NewMockClockAt creates a new MockClick with the internal time set
// to the provided time.
func NewMockClockAt(now time.Time) *MockClock {
	return &MockClock{
		fakeTime:   now,
		tickers:    make([]*mockTicker, 0),
		afterArgs:  make([]time.Duration, 0),
		tickerArgs: make([]time.Duration, 0),
	}
}

// SetCurrent sets the internal MockClock time to the supplied time.
func (mc *MockClock) SetCurrent(current time.Time) {
	mc.nowLock.Lock()
	defer mc.nowLock.Unlock()

	mc.fakeTime = current
}

// Advance will advance the internal MockClock time by the supplied time.
func (mc *MockClock) Advance(duration time.Duration) {
	mc.nowLock.Lock()
	now := mc.fakeTime.Add(duration)
	mc.fakeTime = now
	mc.nowLock.Unlock()

	mc.processTriggers(now)
	mc.processTickers(now)
}

func (mc *MockClock) processTriggers(now time.Time) {
	mc.afterLock.Lock()
	defer mc.afterLock.Unlock()

	triggered := 0
	for _, trigger := range mc.triggers {
		if trigger.trigger.Before(now) || trigger.trigger.Equal(now) {
			trigger.ch <- trigger.trigger
			triggered++
		}
	}

	mc.triggers = mc.triggers[triggered:]
}

func (mc *MockClock) processTickers(now time.Time) {
	mc.tickerLock.Lock()
	defer mc.tickerLock.Unlock()

	for _, ticker := range mc.tickers {
		ticker.publish(now)
	}
}

// BlockingAdvance will call Advance but only after there is another routine
// which is blocking on the channel result of a call to After.
func (mc *MockClock) BlockingAdvance(duration time.Duration) {
	for mc.BlockedOnAfter() == 0 {
		runtime.Gosched()
	}

	mc.Advance(duration)
}

// Now returns the current time internal to the MockClock
func (mc *MockClock) Now() time.Time {
	mc.nowLock.RLock()
	defer mc.nowLock.RUnlock()

	return mc.fakeTime
}

// After returns a channel that will be sent the current internal MockClock
// time once the MockClock's internal time is at or past the provided duration
func (mc *MockClock) After(duration time.Duration) <-chan time.Time {
	mc.nowLock.RLock()
	triggerTime := mc.fakeTime.Add(duration)
	mc.nowLock.RUnlock()

	mc.afterLock.Lock()
	defer mc.afterLock.Unlock()

	trigger := &mockTrigger{
		trigger: triggerTime,
		ch:      make(chan time.Time, 1),
	}

	mc.triggers = append(mc.triggers, trigger)
	mc.afterArgs = append(mc.afterArgs, duration)

	sort.Slice(mc.triggers, func(i, j int) bool {
		return mc.triggers[i].trigger.Before(mc.triggers[j].trigger)
	})

	return trigger.ch
}

// BlockedOnAfter returns the number of calls to After that are blocked
// waiting for a call to Advance to trigger them.
func (mc *MockClock) BlockedOnAfter() int {
	mc.afterLock.RLock()
	defer mc.afterLock.RUnlock()

	return len(mc.triggers)
}

// Sleep will block until the internal MockClock time is at or past the
// provided duration
func (mc *MockClock) Sleep(duration time.Duration) {
	<-mc.After(duration)
}

// Since returns the time elapsed since t.
func (mc *MockClock) Since(t time.Time) time.Duration {
	return mc.Now().Sub(t)
}

// Until returns the duration until t.
func (mc *MockClock) Until(t time.Time) time.Duration {
	return t.Sub(mc.Now())
}

// GetAfterArgs returns the duration of each call to After in the
// same order as they were called. The list is cleared each time
// GetAfterArgs is called.
func (mc *MockClock) GetAfterArgs() []time.Duration {
	mc.afterLock.Lock()
	defer mc.afterLock.Unlock()

	args := mc.afterArgs
	mc.afterArgs = mc.afterArgs[:0]
	return args
}

// GetTickerArgs returns the duration of each call to create a new
// ticker in the same order as they were called. The list is cleared
// each time GetTickerArgs is called.
func (mc *MockClock) GetTickerArgs() []time.Duration {
	mc.tickerLock.Lock()
	defer mc.tickerLock.Unlock()

	args := mc.tickerArgs
	mc.tickerArgs = mc.tickerArgs[:0]
	return args
}
