package glock

import "time"

// Clock is a wrapper around common functions in the time package. This interface
// is designed to allow easy mocking of time functions.
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

	// NewTimer will construct a timer which will fire once after the
	// given duration.
	NewTimer(duration time.Duration) Timer

	// AfterFunc waits for the duration to elapse and then calls f in
	// its own goroutine. It returns a Timer that can be used to cancel the call
	// using its Stop method.
	AfterFunc(duration time.Duration, f func()) Timer
}
