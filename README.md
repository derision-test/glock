# Glock

[![GoDoc](https://godoc.org/github.com/efritz/glock?status.svg)](https://godoc.org/github.com/efritz/glock)

Small go library for mocking parts of the [time package](https://golang.org/pkg/time/).

## Example

The package contains a `Clock` and `Ticker` interface which wrap the `time.Now` and `time.After`
functions and the `Ticker` struct, respectively.

A *real* clock can be created for general (non-test) use. This implementation simply falls back
to the functions provided in the time package. 

```go
clock := glock.NewRealClock()
clock.Now()                       // calls time.Now
clock.After(time.Second)          // calls time.After(time.Second)

t := clock.NewTicker(time.Second) // wraps time.NewTicker(time.Second)
t.Chan()                          // returns ticker's C field
t.Stop()                          // stops the ticker
```

In order to make unit tests that depend on time deterministic (and free of sleep calls), a *mock*
clock can be used in place of the real clock. The mock clock allows you to control the current
time with `SetCurrent`, `SetCurrentToNow`, and `Advance` methods.

```go
clock := glock.NewMockClock()

clock.Now() // returns time of creation
clock.Now() // returns time of creation
clock.SetCurrent(time.Unix(603288000, 0))
clock.Now() // returns Feb 12, 1989
clock.Advance(time.Day)
clock.Now() // returns Feb 13, 1989
clock.SetCurrentToNow()
clock.Now() // returns time of previous call
```

The mock clock also allows you to control the point at which the `After` and `Ticker` channels
yield a value, and also stores the arguments of the methods for later inspection. It is up to
the user to close the associated channels in unit tests (they are not closed automatically). Be
careful not to close a channel too early, as a read from a closed channel will immediately yield.

```go
ch := make(chan time.Time)
clock := glock.NewMockClockWithAfterChan(ch)

clock.After(time.Minute) // returns ch
clock.After(time.Second) // returns ch
clock.GetAfterArgs()     // returns {time.Minute, time.Second}
clock.After(time.Hour)   // returns ch
clock.GetAfterArgs()     // returns {time.Hour}
```

```go
ch := make(chan time.Time)
ticker := glock.NewMockTicker(ch)

clock := glock.NewMockClockWithTicker(ticker)
clock.NewTicker(time.Second) // returns ticker
clock.GetTickerArgs()        // returns {time.Second}

ticker.Chan()      // returns ch
ticker.IsStopped() // returns false
ticker.Stop()
ticker.IsStopped() // returns true
```

## License

Copyright (c) 2017 Eric Fritz

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
