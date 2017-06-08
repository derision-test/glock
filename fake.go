package glock

import (
	"sort"
	"sync"
	"time"
)

type fakeTriggers []*fakeTrigger

func (ft fakeTriggers) Len() int {
	return len(ft)
}
func (ft fakeTriggers) Less(i, j int) bool {
	return ft[i].trigger.Before(ft[j].trigger)
}
func (ft fakeTriggers) Swap(i, j int) {
	ft[i], ft[j] = ft[j], ft[i]
}

type fakeTrigger struct {
	trigger time.Time
	ch      chan time.Time
}

type FakeClock struct {
	fakeTime time.Time

	triggers fakeTriggers
	tickers  []*fakeTicker
}

func NewFakeClock() *FakeClock {
	return &FakeClock{
		fakeTime: time.Now(),
	}
}

func (fc *FakeClock) processTickers() {
	now := fc.Now()
	for _, ticker := range fc.tickers {
		ticker.process(now)
	}
}

func (fc *FakeClock) processTriggers() {
	now := fc.Now()
	triggered := 0
	for _, trigger := range fc.triggers {
		if trigger.trigger.Before(now) || trigger.trigger.Equal(now) {
			trigger.ch <- trigger.trigger
			triggered++
		}
	}

	fc.triggers = fc.triggers[triggered:]
}

func (fc *FakeClock) SetCurrent(current time.Time) {
	fc.fakeTime = current
}

func (fc *FakeClock) Advance(duration time.Duration) {
	fc.fakeTime = fc.fakeTime.Add(duration)
	fc.processTickers()
	fc.processTriggers()
}

func (fc *FakeClock) Now() time.Time {
	return fc.fakeTime
}

func (fc *FakeClock) After(duration time.Duration) <-chan time.Time {
	trigger := &fakeTrigger{
		trigger: fc.fakeTime.Add(duration),
		ch:      make(chan time.Time, 1),
	}
	fc.triggers = append(fc.triggers, trigger)
	sort.Sort(fc.triggers)

	return trigger.ch
}

func (fc *FakeClock) Sleep(duration time.Duration) {
	<-fc.After(duration)
}

type fakeTicker struct {
	clock    *FakeClock
	duration time.Duration

	started  time.Time
	nextTick time.Time

	processLock  sync.Mutex
	processQueue []time.Time

	writeLock sync.Mutex
	writing   bool
	ch        chan time.Time

	stopped bool
}

func (fc *FakeClock) NewTicker(duration time.Duration) Ticker {
	if duration == 0 {
		panic("duration cannot be 0")
	}

	now := fc.Now()

	ft := &fakeTicker{
		clock:    fc,
		duration: duration,

		started:  now,
		nextTick: now.Add(duration),

		processQueue: make([]time.Time, 0),
		ch:           make(chan time.Time),
	}
	fc.tickers = append(fc.tickers, ft)

	return ft
}

func (ft *fakeTicker) process(now time.Time) {
	if ft.stopped {
		return
	}

	ft.processLock.Lock()
	ft.processQueue = append(ft.processQueue, now)
	ft.processLock.Unlock()

	if !ft.writing && (ft.nextTick.Before(now) || ft.nextTick.Equal(now)) {
		ft.writeLock.Lock()

		ft.writing = true
		go func() {
			defer ft.writeLock.Unlock()

			for {
				ft.processLock.Lock()
				if len(ft.processQueue) == 0 {
					ft.processLock.Unlock()
					break
				}

				procTime := ft.processQueue[0]
				ft.processQueue = ft.processQueue[1:]

				ft.processLock.Unlock()

				if ft.nextTick.After(procTime) {
					continue
				}

				ft.ch <- ft.nextTick

				durationMod := procTime.Sub(ft.started) % ft.duration

				if durationMod == 0 {
					ft.nextTick = procTime.Add(ft.duration)
				} else if procTime.Sub(ft.nextTick) > ft.duration {
					ft.nextTick = procTime.Add(ft.duration - durationMod)
				} else {
					ft.nextTick = ft.nextTick.Add(ft.duration)
				}
			}

			ft.writing = false
		}()
	}
}

func (ft *fakeTicker) Chan() <-chan time.Time {
	return ft.ch
}

func (ft *fakeTicker) Stop() {
	ft.stopped = true
}
