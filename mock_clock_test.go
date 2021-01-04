package glock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMockClock(t *testing.T) {
	// Since we can't know the exact time that something gets created
	// at least make sure a new mock clock picks up a time AROUND now
	// instead of an exact time
	clock := NewMockClock()

	since := time.Since(clock.fakeTime)
	assert.Condition(t, func() bool { return since < 100*time.Millisecond })
}

func TestNow(t *testing.T) {
	clock := NewMockClock()
	assert.Equal(t, clock.fakeTime, clock.Now())
}

func TestSetCurrent(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(100, 0))
	assert.Equal(t, time.Unix(100, 0), clock.Now())
}

func TestAdvance(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(100, 0))
	assert.Equal(t, time.Unix(100, 0), clock.Now())
	clock.Advance(1 * time.Second)
	assert.Equal(t, time.Unix(101, 0), clock.Now())
	clock.Advance(1 * time.Hour)
	assert.Equal(t, time.Unix(3701, 0), clock.Now())
}

func TestAdvanceMultipleTriggers(t *testing.T) {
	clock := NewMockClock()
	results := make(chan time.Time)

	go func() {
		clock.Sleep(10 * time.Millisecond)
		results <- time.Unix(0, 0)
	}()

	go func() {
		clock.Sleep(5 * time.Millisecond)
		results <- time.Unix(0, 0)
	}()

	go func() {
		clock.Sleep(1 * time.Millisecond)
		results <- time.Unix(0, 0)
	}()

	// Allow goroutines to schedule
	<-time.After(time.Millisecond * 25)

	clock.Advance(6 * time.Millisecond)
	eventually(t, chanReceives(results, time.Unix(0, 0)))
	eventually(t, chanReceives(results, time.Unix(0, 0)))
	consistently(t, chanDoesNotReceive(results))

	clock.Advance(4 * time.Millisecond)
	eventually(t, chanReceives(results, time.Unix(0, 0)))
}

func TestBlockingAdvance(t *testing.T) {
	clock := NewMockClock()
	sync := make(chan struct{})
	done := make(chan time.Time)

	clock.SetCurrent(time.Unix(100, 0))
	assert.Equal(t, time.Unix(100, 0), clock.Now())

	go func() {
		<-sync
		clock.BlockingAdvance(time.Second)
		close(done)
	}()

	clock.After(time.Second)
	assert.Equal(t, time.Unix(100, 0), clock.Now())
	consistentlyNot(t, chanClosed(done))

	close(sync)
	eventually(t, chanClosed(done))
	assert.Equal(t, time.Unix(101, 0), clock.Now())
}

func TestGetAfterArgs(t *testing.T) {
	clock := NewMockClock()

	clock.After(3 * time.Second)
	clock.After(1 * time.Second)
	clock.After(2 * time.Second)

	args := clock.GetAfterArgs()
	assert.Len(t, args, 3)
	assert.Equal(t, []time.Duration{3 * time.Second, 1 * time.Second, 2 * time.Second}, args)

	clock.After(4 * time.Second)
	clock.After(5 * time.Second)
	clock.After(6 * time.Second)

	args = clock.GetAfterArgs()
	assert.Len(t, args, 3)
	assert.Equal(t, []time.Duration{4 * time.Second, 5 * time.Second, 6 * time.Second}, args)
}

func TestAfter(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	after := clock.After(1 * time.Second)
	consistently(t, chanDoesNotReceive(after))

	clock.Advance(500 * time.Millisecond)
	consistently(t, chanDoesNotReceive(after))

	clock.Advance(250 * time.Millisecond)
	consistently(t, chanDoesNotReceive(after))

	clock.Advance(250 * time.Millisecond)
	eventually(t, chanReceives(after, time.Unix(1, 0)))

	// With the real clock the channel will block after receiving
	// the "after" value
	consistently(t, chanDoesNotReceive(after))
}

func TestMultipleAfter(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	a1 := clock.After(1 * time.Second)
	a2 := clock.After(2 * time.Second)

	consistently(t, chanDoesNotReceive(a1))
	consistently(t, chanDoesNotReceive(a2))

	clock.Advance(500 * time.Millisecond)
	consistently(t, chanDoesNotReceive(a1))
	consistently(t, chanDoesNotReceive(a2))

	clock.Advance(500 * time.Millisecond)
	eventually(t, chanReceives(a1, time.Unix(1, 0)))
	consistently(t, chanDoesNotReceive(a2))

	clock.Advance(500 * time.Millisecond)
	consistently(t, chanDoesNotReceive(a1))
	consistently(t, chanDoesNotReceive(a2))

	clock.Advance(500 * time.Millisecond)
	consistently(t, chanDoesNotReceive(a1))
	eventually(t, chanReceives(a2, time.Unix(2, 0)))

	clock.Advance(500 * time.Millisecond)
	consistently(t, chanDoesNotReceive(a1))
	consistently(t, chanDoesNotReceive(a2))
}

func TestAfterNotExact(t *testing.T) {
	// Make sure triggers are still fired even if the
	// time doesn't match up exactly with the trigger
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	a1 := clock.After(1 * time.Second)
	a2 := clock.After(2 * time.Second)

	clock.Advance(3 * time.Second)
	eventually(t, chanReceives(a1, time.Unix(1, 0)))
	eventually(t, chanReceives(a2, time.Unix(2, 0)))
}

func TestAfterTriggersSorted(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	clock.After(3 * time.Second)
	assert.Len(t, clock.triggers, 1)
	assert.Equal(t, time.Unix(3, 0), clock.triggers[0].trigger)

	clock.After(1 * time.Second)
	assert.Len(t, clock.triggers, 2)
	assert.Equal(t, time.Unix(1, 0), clock.triggers[0].trigger)
	assert.Equal(t, time.Unix(3, 0), clock.triggers[1].trigger)

	clock.After(2 * time.Second)
	assert.Len(t, clock.triggers, 3)
	assert.Equal(t, time.Unix(1, 0), clock.triggers[0].trigger)
	assert.Equal(t, time.Unix(2, 0), clock.triggers[1].trigger)
	assert.Equal(t, time.Unix(3, 0), clock.triggers[2].trigger)
}

func TestRemoveAfterTrigger(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	clock.After(1 * time.Second)
	clock.After(2 * time.Second)
	clock.After(3 * time.Second)
	assert.Len(t, clock.triggers, 3)
	assert.Equal(t, time.Unix(1, 0), clock.triggers[0].trigger)
	assert.Equal(t, time.Unix(2, 0), clock.triggers[1].trigger)
	assert.Equal(t, time.Unix(3, 0), clock.triggers[2].trigger)

	clock.Advance(1 * time.Second)
	assert.Len(t, clock.triggers, 2)
	assert.Equal(t, time.Unix(2, 0), clock.triggers[0].trigger)
	assert.Equal(t, time.Unix(3, 0), clock.triggers[1].trigger)

	clock.Advance(1 * time.Second)
	assert.Len(t, clock.triggers, 1)
	assert.Equal(t, time.Unix(3, 0), clock.triggers[0].trigger)

	clock.Advance(1 * time.Second)
	assert.Len(t, clock.triggers, 0)
}

func TestBlockedOnAfter(t *testing.T) {
	clock := NewMockClock()

	assert.Equal(t, 0, clock.BlockedOnAfter())

	// Set up the blocking calls
	clock.After(5 * time.Second)
	assert.Equal(t, 1, clock.BlockedOnAfter())

	clock.After(10 * time.Second)
	assert.Equal(t, 2, clock.BlockedOnAfter())

	clock.After(15 * time.Second)
	assert.Equal(t, 3, clock.BlockedOnAfter())

	// Start advancing the clock
	clock.Advance(6 * time.Second)
	assert.Equal(t, 2, clock.BlockedOnAfter())

	clock.Advance(3 * time.Second)
	assert.Equal(t, 2, clock.BlockedOnAfter())

	clock.Advance(1 * time.Second)
	assert.Equal(t, 1, clock.BlockedOnAfter())

	clock.Advance(5 * time.Second)
	assert.Equal(t, 0, clock.BlockedOnAfter())
}

func TestSleep(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	sync := make(chan struct{})
	finished := make(chan time.Time, 1)
	go func() {
		close(sync)
		clock.Sleep(1 * time.Second)
		finished <- clock.Now()
	}()

	// Make sure the goroutine starts up and sleep is starting before we
	// start advancing time.  There is still a slight race here because the
	// trigger doesn't get added until Sleep() itself is called but this at
	// least slows down until the goroutine is executing.  There's probably
	// a better way to do this, so fix it if you think of it!
	<-sync

	consistently(t, chanDoesNotReceive(finished))

	clock.Advance(500 * time.Millisecond)
	consistently(t, chanDoesNotReceive(finished))

	clock.Advance(500 * time.Millisecond)
	eventually(t, chanReceives(finished, time.Unix(1, 0)))
}

func TestSince(t *testing.T) {
	clock := NewMockClockAt(time.Unix(10, 0))

	assert.Equal(t, time.Duration(0), clock.Since(time.Unix(10, 0)))
	assert.Equal(t, 5*time.Second, clock.Since(time.Unix(5, 0)))
	assert.Equal(t, -5*time.Second, clock.Since(time.Unix(15, 0)))
}

func TestUntil(t *testing.T) {
	clock := NewMockClockAt(time.Unix(10, 0))

	assert.Equal(t, time.Duration(0), clock.Until(time.Unix(10, 0)))
	assert.Equal(t, -5*time.Second, clock.Until(time.Unix(5, 0)))
	assert.Equal(t, 5*time.Second, clock.Until(time.Unix(15, 0)))
}
