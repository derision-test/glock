# Glock

[![GoDoc](https://godoc.org/github.com/efritz/glock?status.svg)](https://godoc.org/github.com/efritz/glock)
[![Build Status](https://secure.travis-ci.org/efritz/glock.png)](http://travis-ci.org/efritz/glock)
[![codecov.io](http://codecov.io/github/efritz/glock/coverage.svg?branch=master)](http://codecov.io/github/efritz/glock?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/efritz/glock)](https://goreportcard.com/report/github.com/efritz/glock)

Small go library for mocking parts of the [time package](https://golang.org/pkg/time/).

## Example

The package contains a `Clock` and `Ticker` interface which wrap the `time.Now`, `time.After`,
adnd `time.Sleep` functions and the `Ticker` struct, respectively.

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
time with `SetCurrent` and `Advance` methods.

```go
clock := glock.NewMockClock()

clock.Now() // returns time of creation
clock.Now() // returns time of creation
clock.SetCurrent(time.Unix(603288000, 0))
clock.Now() // returns Feb 12, 1989
clock.Advance(time.Day)
clock.Now() // returns Feb 13, 1989
```

The `Advance` method will also trigger a value on the channels created by the `After` and
`Ticker` functions.

```go
clock := glock.NewMockClockAt(time.Unix(603288000, 0))

c1 := clock.After(time.Second)
c2 := clock.After(time.Minute)
clock.GetAfterArgs()            // returns {time.Second, time.Minute}
clock.GetAfterArgs()            // returns {}
clock.Advance(time.Second * 30) // Fires c2
clock.Advance(time.Second * 30) // Fires c1
```

```go
clock := glock.NewMockClock()
ticker := clock.NewTicker(time.Minute)

ch := ticker.Chan()
clock.Advance(time.Second * 30)
clock.Advance(time.Second * 30) // Fires ch
clock.Advance(time.Second * 30)
clock.Advance(time.Second * 30) // Fires ch
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
