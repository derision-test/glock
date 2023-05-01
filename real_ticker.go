package glock

import "time"

type realTicker struct {
	ticker *time.Ticker
}

var _ Ticker = &realTicker{}

func NewRealTicker(duration time.Duration) Ticker {
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
