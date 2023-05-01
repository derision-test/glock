package glock

import "time"

// Ticker is a wrapper around a time.Ticker, which allows interface access to the
// underlying channel (instead of bare access like the time.Ticker struct allows).
type Ticker interface {
	// Chan returns the underlying ticker channel.
	Chan() <-chan time.Time

	// Stop stops the ticker.
	Stop()
}
