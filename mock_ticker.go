package glock

import (
	"sync"
	"time"
)

// Ticker is a wrapper around a time.Ticker, which allows interface
// access  to the underlying channel (instead of bare access like the
// time.Ticker struct allows).
type Ticker interface {
	// Chan returns the underlying ticker channel.
	Chan() <-chan time.Time

	// Stop stops the ticker.
	Stop()
}

type mockTicker struct {
	clock        *MockClock
	duration     time.Duration
	started      time.Time
	nextTick     time.Time
	processQueue []time.Time
	ch           chan time.Time
	wakeup       chan struct{}
	stopped      bool
	processLock  sync.Mutex
	stoppedLock  sync.RWMutex
}

var _ Ticker = &mockTicker{}

// NewTicker creates a new Ticker tied to the internal MockClock time that ticks
// at intervals similar to time.NewTicker().  It will also skip or drop ticks
// for slow readers similar to time.NewTicker() as well.
func (mc *MockClock) NewTicker(duration time.Duration) Ticker {
	if duration == 0 {
		panic("duration cannot be 0")
	}

	now := mc.Now()
	ticker := &mockTicker{
		clock:        mc,
		duration:     duration,
		started:      now,
		nextTick:     now.Add(duration),
		processQueue: make([]time.Time, 0),
		ch:           make(chan time.Time),
		wakeup:       make(chan struct{}, 1),
	}

	mc.tickerLock.Lock()
	defer mc.tickerLock.Unlock()

	mc.tickers = append(mc.tickers, ticker)
	mc.tickerArgs = append(mc.tickerArgs, duration)
	go ticker.process()
	return ticker
}

// Chan returns a channel which will receive the MockClock's internal time
// at the interval given when creating the ticker.
func (mt *mockTicker) Chan() <-chan time.Time {
	return mt.ch
}

// Stop will stop the ticker from ticking
func (mt *mockTicker) Stop() {
	mt.stoppedLock.Lock()
	defer mt.stoppedLock.Unlock()

	mt.stopped = true
	mt.wakeup <- struct{}{}
}

func (mt *mockTicker) publish(now time.Time) {
	if mt.isStopped() {
		return
	}

	mt.processLock.Lock()
	mt.processQueue = append(mt.processQueue, now)
	mt.processLock.Unlock()

	select {
	case mt.wakeup <- struct{}{}:
	default:
	}
}

func (mt *mockTicker) process() {
	defer close(mt.wakeup)

	for !mt.isStopped() {
		for {
			first, ok := mt.pop()
			if !ok {
				break
			}

			if mt.nextTick.After(first) {
				continue
			}

			mt.ch <- mt.nextTick

			durationMod := first.Sub(mt.started) % mt.duration

			if durationMod == 0 {
				mt.nextTick = first.Add(mt.duration)
			} else if first.Sub(mt.nextTick) > mt.duration {
				mt.nextTick = first.Add(mt.duration - durationMod)
			} else {
				mt.nextTick = mt.nextTick.Add(mt.duration)
			}
		}

		<-mt.wakeup
	}
}

func (mt *mockTicker) pop() (time.Time, bool) {
	mt.processLock.Lock()
	defer mt.processLock.Unlock()

	if len(mt.processQueue) == 0 {
		return time.Unix(0, 0), false
	}

	first := mt.processQueue[0]
	mt.processQueue = mt.processQueue[1:]
	return first, true
}

func (mt *mockTicker) isStopped() bool {
	mt.stoppedLock.RLock()
	defer mt.stoppedLock.RUnlock()

	return mt.stopped
}
