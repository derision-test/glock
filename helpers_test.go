package glock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockTestingT struct{}

func (mockTestingT) Errorf(format string, args ...interface{}) {}

func eventually(t *testing.T, cond func() bool) bool {
	return assert.Eventually(t, cond, time.Second, 10*time.Millisecond)
}

func consistently(t *testing.T, cond func() bool) bool {
	if !assert.Eventually(mockTestingT{}, func() bool { return !cond() }, 100*time.Millisecond, 10*time.Millisecond) {
		return true
	}

	return assert.Fail(t, "Condition not met during test period")
}

func consistentlyNot(t *testing.T, cond func() bool) bool {
	return consistently(t, func() bool { return !cond() })
}

func chanClosed(ch <-chan time.Time) func() bool {
	return func() bool {
		select {
		case _, ok := <-ch:
			return !ok
		default:
			return false
		}
	}
}

func chanReceives(ch <-chan time.Time, expected time.Time) func() bool {
	return func() bool {
		select {
		case v := <-ch:
			return v == expected
		default:
			return false
		}
	}
}

func chanDoesNotReceive(ch <-chan time.Time) func() bool {
	return func() bool {
		select {
		case <-ch:
			return false
		default:
			return true
		}
	}
}

func structChanReceives(ch <-chan struct{}) func() bool {
	return func() bool {
		select {
		case <-ch:
			return true
		default:
			return false
		}
	}
}

func structChanDoesNotReceive(ch <-chan struct{}) func() bool {
	return func() bool {
		select {
		case <-ch:
			return false
		default:
			return true
		}
	}
}
