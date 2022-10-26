package glock

import "time"

// Timer is a wrapper around a time.Timer, which allows interface access to
// the underlying Timer.
type Timer interface {
	// Chan returns the underlying timer channel.
	Chan() <-chan time.Time

	// Reset will reset the duration of the timer to the new duration. If
	// the Timer was running when Reset was called it will return true. For
	// more information about when Reset can be called, see the time.Timer
	// documentation.
	Reset(d time.Duration) bool
	// Stop stops the timer. If the timer was running when Stop was
	// called it will return true.
	Stop() bool
}
