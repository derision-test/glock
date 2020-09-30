package glock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTickerArgs(t *testing.T) {
	clock := NewMockClock()

	clock.NewTicker(3 * time.Second)
	clock.NewTicker(1 * time.Second)
	clock.NewTicker(2 * time.Second)

	args := clock.GetTickerArgs()
	assert.Len(t, args, 3)
	assert.Equal(t, []time.Duration{3 * time.Second, 1 * time.Second, 2 * time.Second}, args)

	clock.NewTicker(4 * time.Second)
	clock.NewTicker(5 * time.Second)
	clock.NewTicker(6 * time.Second)

	args = clock.GetTickerArgs()
	assert.Len(t, args, 3)
	assert.Equal(t, []time.Duration{4 * time.Second, 5 * time.Second, 6 * time.Second}, args)
}

func TestNewTickerNoDuration(t *testing.T) {
	clock := NewMockClock()
	assert.Panics(t, func() { clock.NewTicker(0) })
}

func TestTickerOnTime(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	ticker := clock.NewTicker(2 * time.Second)
	consistently(t, chanDoesNotReceive(ticker.Chan()))

	clock.Advance(1 * time.Second)
	consistently(t, chanDoesNotReceive(ticker.Chan()))

	clock.Advance(1 * time.Second)
	eventually(t, chanReceives(ticker.Chan(), time.Unix(2, 0)))

	clock.Advance(1 * time.Second)
	consistently(t, chanDoesNotReceive(ticker.Chan()))

	clock.Advance(1 * time.Second)
	eventually(t, chanReceives(ticker.Chan(), time.Unix(4, 0)))

	clock.Advance(2 * time.Second)
	eventually(t, chanReceives(ticker.Chan(), time.Unix(6, 0)))
}

// This test makes sure the ticker works similar to the actual go implementation
// as seen with the code pasted below:
/*
package main

import (
	"fmt"
	"time"
)

func main() {
	ticker := time.NewTicker(time.Second * 2)

	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(3 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(1 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(7 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(7 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(7 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(1 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(1 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(6 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
}
*/
func TestTickerOffset2Second(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	ticker := clock.NewTicker(2 * time.Second)
	consistently(t, chanDoesNotReceive(ticker.Chan()))

	clock.Advance(2 * time.Second) // 2s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(2, 0)))
	clock.Advance(3 * time.Second) // 5s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(4, 0)))
	clock.Advance(1 * time.Second) // 6s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(6, 0)))
	clock.Advance(7 * time.Second) // 13s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(8, 0)))
	clock.Advance(7 * time.Second) // 20s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(14, 0)))
	clock.Advance(7 * time.Second) // 27s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(22, 0)))
	clock.Advance(1 * time.Second) // 28s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(28, 0)))
	clock.Advance(1 * time.Second) // 29s
	consistently(t, chanDoesNotReceive(ticker.Chan()))
	clock.Advance(1 * time.Second) // 30s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(30, 0)))
	clock.Advance(6 * time.Second) // 36s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(32, 0)))
}

// This test makes sure the ticker works similar to the actual go implementation
// as seen with the code pasted below:
/*
package main

import (
	"fmt"
	"time"
)

func main() {
	ticker := time.NewTicker(time.Second * 3)

	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(3 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(1 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(7 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(7 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(7 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(1 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(1 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
	time.Sleep(6 * time.Second)
	fmt.Printf("%s %s\n", time.Now(), <-ticker.C)
}
*/
func TestTickerOffset3Second(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	ticker := clock.NewTicker(3 * time.Second)
	consistently(t, chanDoesNotReceive(ticker.Chan()))

	clock.Advance(3 * time.Second) // 3s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(3, 0)))
	clock.Advance(3 * time.Second) // 6s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(6, 0)))
	clock.Advance(1 * time.Second) // 7s
	consistently(t, chanDoesNotReceive(ticker.Chan()))
	clock.Advance(2 * time.Second) // 9s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(9, 0)))
	clock.Advance(7 * time.Second) // 16s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(12, 0)))
	clock.Advance(7 * time.Second) // 23s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(18, 0)))
	clock.Advance(7 * time.Second) // 30s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(24, 0)))
	clock.Advance(1 * time.Second) // 31s
	consistently(t, chanDoesNotReceive(ticker.Chan()))
	clock.Advance(2 * time.Second) // 33s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(33, 0)))
	clock.Advance(1 * time.Second) // 34s
	consistently(t, chanDoesNotReceive(ticker.Chan()))
	clock.Advance(1 * time.Second) // 35s
	consistently(t, chanDoesNotReceive(ticker.Chan()))
	clock.Advance(1 * time.Second) // 36s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(36, 0)))
	clock.Advance(6 * time.Second) // 42s
	eventually(t, chanReceives(ticker.Chan(), time.Unix(39, 0)))
}

func TestMultipleTickers(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	t2 := clock.NewTicker(2 * time.Second)
	t3 := clock.NewTicker(3 * time.Second)

	clock.Advance(1 * time.Second)
	consistently(t, chanDoesNotReceive(t2.Chan()))
	consistently(t, chanDoesNotReceive(t3.Chan()))

	clock.Advance(1 * time.Second)
	eventually(t, chanReceives(t2.Chan(), time.Unix(2, 0)))
	consistently(t, chanDoesNotReceive(t3.Chan()))

	clock.Advance(1 * time.Second)
	consistently(t, chanDoesNotReceive(t2.Chan()))
	eventually(t, chanReceives(t3.Chan(), time.Unix(3, 0)))

	clock.Advance(1 * time.Second)
	eventually(t, chanReceives(t2.Chan(), time.Unix(4, 0)))
	consistently(t, chanDoesNotReceive(t3.Chan()))

	clock.Advance(1 * time.Second)
	consistently(t, chanDoesNotReceive(t2.Chan()))
	consistently(t, chanDoesNotReceive(t3.Chan()))

	clock.Advance(1 * time.Second)
	eventually(t, chanReceives(t2.Chan(), time.Unix(6, 0)))
	eventually(t, chanReceives(t3.Chan(), time.Unix(6, 0)))
}

func TestTickerStopped(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(0, 0))

	ticker := clock.NewTicker(2 * time.Second)

	clock.Advance(2 * time.Second)
	eventually(t, chanReceives(ticker.Chan(), time.Unix(2, 0)))
	clock.Advance(2 * time.Second)
	eventually(t, chanReceives(ticker.Chan(), time.Unix(4, 0)))

	ticker.Stop()

	clock.Advance(2 * time.Second)
	consistently(t, chanDoesNotReceive(ticker.Chan()))
	clock.Advance(2 * time.Second)
	consistently(t, chanDoesNotReceive(ticker.Chan()))
}
