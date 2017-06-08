package glock

import "time"

type (
	// Clock is a wrapper around common functions in the time package.
	// This interface is designed to allow easy mocking of time functions.
	Clock interface {
		// Now returns the current time.
		Now() time.Time

		// After returns a channel which receives the current time after
		// the given duration elapses.
		After(duration time.Duration) <-chan time.Time

		// Sleep blocks until the given duration elapses.
		Sleep(duration time.Duration)

		// NewTicker will construct a ticker which will continually fire,
		// pausing for the given duration in between invocations.
		NewTicker(duration time.Duration) Ticker
	}

	// Ticker is a wrapper around a time.Ticker, which allows interface
	// access  to the underlying channel (instead of bare access like the
	// time.Ticker struct allows).
	Ticker interface {
		// Chan returns the underlying ticker channel.
		Chan() <-chan time.Time

		// Stop stops the ticker.
		Stop()
	}
)
