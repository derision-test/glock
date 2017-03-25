package glock

import (
	"time"
)

type (
	realClock  struct{}
	realTicker struct {
		ticker *time.Ticker
	}
)

func NewRealClock() Clock {
	return &realClock{}
}

func (c *realClock) Now() time.Time {
	return time.Now()
}

func (c *realClock) After(duration time.Duration) <-chan time.Time {
	return time.After(duration)
}

func (c *realClock) NewTicker(duration time.Duration) Ticker {
	return &realTicker{
		ticker: time.NewTicker(duration),
	}
}

func (t *realTicker) Chan() <-chan time.Time {
	return t.ticker.C
}

func (t *realTicker) Stop() {
	t.ticker.Stop()
}
