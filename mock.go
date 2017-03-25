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

func NewMockClock() *mockClock {
	return NewMockClockWithAfterChan(nil)
}

func NewMockClockWithAfterChan(afterChan <-chan time.Time) *mockClock {
	return NewMockClockWithAfterChanAndTicker(afterChan, nil)
}

func NewMockClockWithTicker(ticker *mockTicker) *mockClock {
	return NewMockClockWithAfterChanAndTicker(nil, ticker)
}

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

func (c *mockClock) SetCurrent(time time.Time) {
	c.current = time
}

func (c *mockClock) SetCurrentToNow() {
	c.SetCurrent(time.Now())
}

func (c *mockClock) Advance(duration time.Duration) {
	c.SetCurrent(c.current.Add(duration))
}

//
//  After utilities

func (c *mockClock) After(duration time.Duration) <-chan time.Time {
	c.afterArgs = append(c.afterArgs, duration)
	return c.afterChan
}

func (c *mockClock) GetAfterArgs() []time.Duration {
	args := c.afterArgs
	c.afterArgs = c.afterArgs[:0]
	return args
}

//
// Ticker utilities

func (c *mockClock) NewTicker(duration time.Duration) Ticker {
	c.tickerArgs = append(c.tickerArgs, duration)
	return c.ticker
}

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

func (t *mockTicker) IsStopped() bool {
	return t.stopped
}
