package glock

import "time"

type (
	Clock interface {
		Now() time.Time
		After(duration time.Duration) <-chan time.Time
		NewTicker(duration time.Duration) Ticker
	}

	Ticker interface {
		Chan() <-chan time.Time
		Stop()
	}
)
