package glock

import "time"

type (
	mockClock struct {
		current    time.Time
		afterChan  <-chan time.Time
		ticker     *mockTicker
		afterArgs  []time.Duration
		tickerArgs []time.Duration
	}

	mockTicker struct {
		ch      <-chan time.Time
		stopped bool
	}
)

// NewMockClock return a mock clock that returns a nil after channel and ticke.r
func NewMockClock() *mockClock {
	return NewMockClockWithAfterChanAndTicker(nil, nil)
}

// NewMockClockWithAfterChan returns a mock clock that returns the given after channel
// and a nil ticker.
func NewMockClockWithAfterChan(afterChan <-chan time.Time) *mockClock {
	return NewMockClockWithAfterChanAndTicker(afterChan, nil)
}

// NewMockClockWithAfterChan returns a mock clock that returns a nil after channel and
// the given ticker.
func NewMockClockWithTicker(ticker *mockTicker) *mockClock {
	return NewMockClockWithAfterChanAndTicker(nil, ticker)
}

// NewMockClockWithAfterChanAndTicker returns a mock clock that returns the given after
// channel and ticker.
func NewMockClockWithAfterChanAndTicker(afterChan <-chan time.Time, ticker *mockTicker) *mockClock {
	return &mockClock{
		current:    time.Now(),
		afterChan:  afterChan,
		ticker:     ticker,
		afterArgs:  []time.Duration{},
		tickerArgs: []time.Duration{},
	}
}

//
// Now utilities

func (c *mockClock) Now() time.Time {
	return c.current
}

// SetCurrent sets the return value of calls to Now.
func (c *mockClock) SetCurrent(time time.Time) {
	c.current = time
}

// SetCurrentToNow sets the return value of calls to Now to the current time.
func (c *mockClock) SetCurrentToNow() {
	c.SetCurrent(time.Now())
}

// Advance advances the return value of calls to Now by the given duration.
func (c *mockClock) Advance(duration time.Duration) {
	c.SetCurrent(c.current.Add(duration))
}

//
//  After utilities

func (c *mockClock) After(duration time.Duration) <-chan time.Time {
	c.afterArgs = append(c.afterArgs, duration)
	return c.afterChan
}

// GetAfterArgs returns an ordered list of arguments to the After method.
// This method resets the list.
func (c *mockClock) GetAfterArgs() []time.Duration {
	args := c.afterArgs
	c.afterArgs = c.afterArgs[:0]
	return args
}

//
// Sleep utilities

func (c *mockClock) Sleep(duration time.Duration) {
	<-c.After(duration)
}

//
// Ticker utilities

func (c *mockClock) NewTicker(duration time.Duration) Ticker {
	c.tickerArgs = append(c.tickerArgs, duration)
	return c.ticker
}

// GetAfterArgs returns an ordered list of arguments to the NewTicker method.
// This method resets the list.
func (c *mockClock) GetTickerArgs() []time.Duration {
	args := c.tickerArgs
	c.tickerArgs = c.tickerArgs[:0]
	return args
}

func NewMockTicker(ch <-chan time.Time) *mockTicker {
	return &mockTicker{
		ch: ch,
	}
}

func (t *mockTicker) Chan() <-chan time.Time {
	return t.ch
}

func (t *mockTicker) Stop() {
	t.stopped = true
}

// IsStopped returns true if Stop has been called.
func (t *mockTicker) IsStopped() bool {
	return t.stopped
}
