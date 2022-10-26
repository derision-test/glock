package glock

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMockTimer(t *testing.T) {
	t.Parallel()

	t.Run("new timer", func(t *testing.T) {
		t.Run("no duration", func(t *testing.T) {
			clock := NewMockClock()
			assert.Panics(t, func() { clock.NewTimer(0) })
		})
	})
	t.Run("blocking advance", func(t *testing.T) {
		t.Run("blocks until channel is read", func(t *testing.T) {
			clock := NewMockClockAt(time.Unix(0, 0))
			timer := clock.NewTimer(1 * time.Second).(*MockTimer)

			finishedBlocking := make(chan struct{})
			go func() {
				// This should close immediately because it doesn't
				// trigger the timer.
				timer.BlockingAdvance(500 * time.Millisecond)
				close(finishedBlocking)
			}()
			eventually(t, chanClosed(finishedBlocking))
			assert.Equal(t, time.Unix(0, 500000000), clock.Now())

			finishedBlocking = make(chan struct{})
			go func() {
				// This should block until the signal chan is read
				// because it triggers the timer.
				timer.BlockingAdvance(500 * time.Millisecond)
				close(finishedBlocking)
			}()
			eventually(t, chanReceives(timer.Chan(), time.Unix(1, 0)))
			eventually(t, chanClosed(finishedBlocking))
			assert.Equal(t, time.Unix(1, 0), clock.Now())
		})
		t.Run("does not block for AfterFunc", func(t *testing.T) {
			// It doesn't make sense for blocking advance to block
			// on AfterFunc because it's not writing to the signal
			// channel.
			funcCalled := uint32(0)

			clock := NewMockClockAt(time.Unix(0, 0))
			timer := clock.AfterFunc(1*time.Second, func() {
				atomic.AddUint32(&funcCalled, 1)
			}).(*MockTimer)

			finishedBlocking := make(chan struct{})
			go func() {
				// This should close immediately because it doesn't
				// trigger the timer.
				timer.BlockingAdvance(500 * time.Millisecond)
				close(finishedBlocking)
			}()
			consistently(t, chanDoesNotReceive(timer.Chan()))
			eventually(t, chanClosed(finishedBlocking))
			consistently(t, func() bool { return atomic.LoadUint32(&funcCalled) == 0 })
			assert.Equal(t, time.Unix(0, 500000000), clock.Now())

			finishedBlocking = make(chan struct{})
			go func() {
				timer.BlockingAdvance(500 * time.Millisecond)
				close(finishedBlocking)
			}()
			consistently(t, chanDoesNotReceive(timer.Chan()))
			eventually(t, chanClosed(finishedBlocking))
			eventually(t, func() bool { return atomic.LoadUint32(&funcCalled) == 1 })
			assert.Equal(t, time.Unix(1, 0), clock.Now())
		})
	})
	t.Run("afterfunc timer", func(t *testing.T) {
		t.Run("runs on trigger", func(t *testing.T) {
			funcCalled := uint32(0)

			clock := NewMockClockAt(time.Unix(0, 0))
			clock.AfterFunc(1*time.Second, func() {
				atomic.AddUint32(&funcCalled, 1)
			})
			consistently(t, func() bool { return atomic.LoadUint32(&funcCalled) == 0 })

			clock.Advance(500 * time.Millisecond)
			consistently(t, func() bool { return atomic.LoadUint32(&funcCalled) == 0 })

			clock.Advance(500 * time.Millisecond)
			eventually(t, func() bool { return atomic.LoadUint32(&funcCalled) == 1 })
		})
		t.Run("runs after expiration and reset", func(t *testing.T) {
			funcCalled := uint32(0)

			clock := NewMockClockAt(time.Unix(0, 0))
			timer := clock.AfterFunc(1*time.Second, func() {
				atomic.AddUint32(&funcCalled, 1)
			})

			clock.Advance(1 * time.Second)
			eventually(t, func() bool { return atomic.LoadUint32(&funcCalled) == 1 })

			assert.False(t, timer.Reset(2*time.Second))

			clock.Advance(1 * time.Second)
			consistently(t, func() bool { return atomic.LoadUint32(&funcCalled) == 1 })

			clock.Advance(1 * time.Second)
			eventually(t, func() bool { return atomic.LoadUint32(&funcCalled) == 2 })
		})
		t.Run("does not run when stopped", func(t *testing.T) {
			funcCalled := uint32(0)

			clock := NewMockClockAt(time.Unix(0, 0))
			timer := clock.AfterFunc(1*time.Second, func() {
				atomic.AddUint32(&funcCalled, 1)
			})

			clock.Advance(500 * time.Millisecond)
			consistently(t, func() bool { return atomic.LoadUint32(&funcCalled) == 0 })

			assert.True(t, timer.Stop())
			consistently(t, func() bool { return atomic.LoadUint32(&funcCalled) == 0 })

			clock.Advance(1 * time.Second)
			consistently(t, func() bool { return atomic.LoadUint32(&funcCalled) == 0 })
		})
		t.Run("extends timer when reset while running", func(t *testing.T) {
			funcCalled := uint32(0)

			clock := NewMockClockAt(time.Unix(0, 0))
			timer := clock.AfterFunc(1*time.Second, func() {
				atomic.AddUint32(&funcCalled, 1)
			})

			clock.Advance(500 * time.Millisecond)
			consistently(t, func() bool { return atomic.LoadUint32(&funcCalled) == 0 })

			assert.True(t, timer.Reset(1*time.Second))

			clock.Advance(500 * time.Millisecond)
			consistently(t, func() bool { return atomic.LoadUint32(&funcCalled) == 0 })

			clock.Advance(500 * time.Millisecond)
			eventually(t, func() bool { return atomic.LoadUint32(&funcCalled) == 1 })
		})
	})
	t.Run("firing", func(t *testing.T) {
		t.Run("on time", func(t *testing.T) {
			t.Run("advance directly to trigger", func(t *testing.T) {
				clock := NewMockClock()
				clock.SetCurrent(time.Unix(0, 0))

				timer := clock.NewTimer(1 * time.Second)
				consistently(t, chanDoesNotReceive(timer.Chan()))

				clock.Advance(1 * time.Second)
				eventually(t, chanReceives(timer.Chan(), time.Unix(1, 0)))
			})
			t.Run("advance once before triggering", func(t *testing.T) {
				clock := NewMockClock()
				clock.SetCurrent(time.Unix(0, 0))

				timer := clock.NewTimer(1 * time.Second)
				consistently(t, chanDoesNotReceive(timer.Chan()))

				clock.Advance(500 * time.Millisecond)
				consistently(t, chanDoesNotReceive(timer.Chan()))

				clock.Advance(500 * time.Millisecond)
				eventually(t, chanReceives(timer.Chan(), time.Unix(1, 0)))
			})
		})
		t.Run("after time", func(t *testing.T) {
			t.Run("advance directly after first trigger", func(t *testing.T) {
				clock := NewMockClock()
				clock.SetCurrent(time.Unix(0, 0))

				timer := clock.NewTimer(1 * time.Second)
				consistently(t, chanDoesNotReceive(timer.Chan()))

				clock.Advance(1*time.Second + 500*time.Millisecond)
				eventually(t, chanReceives(timer.Chan(), time.Unix(1, 500000000)))
			})
			t.Run("advance directly past second trigger", func(t *testing.T) {
				clock := NewMockClock()
				clock.SetCurrent(time.Unix(0, 0))

				timer := clock.NewTimer(1 * time.Second)
				consistently(t, chanDoesNotReceive(timer.Chan()))

				clock.Advance(2*time.Second + 500*time.Millisecond)
				eventually(t, chanReceives(timer.Chan(), time.Unix(2, 500000000)))
			})
			t.Run("advance once before triggering", func(t *testing.T) {
				clock := NewMockClock()
				clock.SetCurrent(time.Unix(0, 0))

				timer := clock.NewTimer(1 * time.Second)
				consistently(t, chanDoesNotReceive(timer.Chan()))

				clock.Advance(500 * time.Millisecond)
				consistently(t, chanDoesNotReceive(timer.Chan()))

				clock.Advance(1 * time.Second)
				eventually(t, chanReceives(timer.Chan(), time.Unix(1, 500000000)))
			})
		})
		t.Run("triggers once", func(t *testing.T) {
			clock := NewMockClock()
			clock.SetCurrent(time.Unix(0, 0))

			timer := clock.NewTimer(1 * time.Second)
			consistently(t, chanDoesNotReceive(timer.Chan()))

			clock.Advance(1 * time.Second)
			eventually(t, chanReceives(timer.Chan(), time.Unix(1, 0)))

			clock.Advance(1 * time.Second)
			consistently(t, chanDoesNotReceive(timer.Chan()))
		})
	})
	t.Run("stopping", func(t *testing.T) {
		t.Run("returns true when stopping timer", func(t *testing.T) {
			clock := NewMockClock()
			clock.SetCurrent(time.Unix(0, 0))

			timer := clock.NewTimer(1 * time.Second)
			clock.Advance(500 * time.Millisecond)

			assert.True(t, timer.Stop())
			consistently(t, chanDoesNotReceive(timer.Chan()))
		})
		t.Run("returns false when already expired", func(t *testing.T) {
			clock := NewMockClock()
			clock.SetCurrent(time.Unix(0, 0))

			timer := clock.NewTimer(1 * time.Second)

			clock.Advance(1 * time.Second)
			eventually(t, chanReceives(timer.Chan(), time.Unix(1, 0)))

			assert.False(t, timer.Stop())
		})
		t.Run("returns false when already stopped", func(t *testing.T) {
			clock := NewMockClock()
			clock.SetCurrent(time.Unix(0, 0))

			timer := clock.NewTimer(1 * time.Second)

			assert.True(t, timer.Stop())

			clock.Advance(500 * time.Millisecond)
			assert.False(t, timer.Stop())
		})
		t.Run("prevents timer from triggering", func(t *testing.T) {
			clock := NewMockClock()
			clock.SetCurrent(time.Unix(0, 0))

			timer := clock.NewTimer(1 * time.Second)
			consistently(t, chanDoesNotReceive(timer.Chan()))

			clock.Advance(500 * time.Millisecond)
			consistently(t, chanDoesNotReceive(timer.Chan()))

			assert.True(t, timer.Stop())
			consistently(t, chanDoesNotReceive(timer.Chan()))

			clock.Advance(500 * time.Millisecond)
			consistently(t, chanDoesNotReceive(timer.Chan()))
		})
	})
	t.Run("resetting", func(t *testing.T) {
		t.Run("returns true if timer was active", func(t *testing.T) {
			clock := NewMockClock()
			clock.SetCurrent(time.Unix(0, 0))

			timer := clock.NewTimer(1 * time.Second)
			clock.Advance(500 * time.Millisecond)

			assert.True(t, timer.Reset(1*time.Second))
		})
		t.Run("returns false if timer had expired", func(t *testing.T) {
			clock := NewMockClock()
			clock.SetCurrent(time.Unix(0, 0))

			timer := clock.NewTimer(1 * time.Second)

			clock.Advance(1 * time.Second)
			eventually(t, chanReceives(timer.Chan(), time.Unix(1, 0)))

			assert.False(t, timer.Reset(1*time.Second))
		})
		t.Run("returns false if timer had been stopped", func(t *testing.T) {
			clock := NewMockClock()
			clock.SetCurrent(time.Unix(0, 0))

			timer := clock.NewTimer(1 * time.Second)

			assert.True(t, timer.Stop())
			assert.False(t, timer.Reset(1*time.Second))
		})
		t.Run("runs again if timer had been stopped", func(t *testing.T) {
			clock := NewMockClock()
			clock.SetCurrent(time.Unix(0, 0))

			timer := clock.NewTimer(1 * time.Second)

			assert.True(t, timer.Stop())
			assert.False(t, timer.Reset(1*time.Second))

			clock.Advance(1 * time.Second)
			eventually(t, chanReceives(timer.Chan(), time.Unix(1, 0)))
		})
		t.Run("runs again if timer had expired", func(t *testing.T) {
			clock := NewMockClock()
			clock.SetCurrent(time.Unix(0, 0))

			timer := clock.NewTimer(1 * time.Second)

			clock.Advance(1 * time.Second)
			eventually(t, chanReceives(timer.Chan(), time.Unix(1, 0)))

			assert.False(t, timer.Reset(1*time.Second))

			clock.Advance(1 * time.Second)
			eventually(t, chanReceives(timer.Chan(), time.Unix(2, 0)))
		})
	})
}
