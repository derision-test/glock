package glock

import "time"

type Timer interface {
	Chan() <-chan time.Time

	Reset(d time.Duration) bool
	Stop() bool
}
