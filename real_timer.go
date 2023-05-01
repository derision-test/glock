package glock

import "time"

type realTimer struct {
	timer *time.Timer
}

var _ Timer = &realTimer{}

func NewRealTimer(duration time.Duration) Timer {
	return newRealTimer(time.NewTimer(duration))
}
func newRealTimer(timer *time.Timer) Timer {
	return &realTimer{
		timer: timer,
	}
}

func (t *realTimer) Chan() <-chan time.Time {
	return t.timer.C
}

func (t *realTimer) Reset(duration time.Duration) bool {
	return t.timer.Reset(duration)
}

func (t *realTimer) Stop() bool {
	return t.timer.Stop()
}
