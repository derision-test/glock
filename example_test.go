package glock

import (
	"fmt"
	"time"
)

func ExampleMockClock() {
	clock := NewMockClock()

	fmt.Printf("Now: %s\n", clock.Now()) // prints time of creation
	fmt.Printf("Now: %s\n", clock.Now()) // prints time of creation
	clock.SetCurrent(time.Unix(603288000, 0))
	fmt.Printf("Now: %s\n", clock.Now()) // prints Feb 12, 1989
	clock.Advance(24 * time.Hour)
	fmt.Printf("Now: %s\n", clock.Now()) // prints Feb 13, 1989
	clock.SetCurrentToNow()
	fmt.Printf("Now: %s\n", clock.Now()) // prints time of previous call

	var clockInterface Clock = clock              // can be assigned to Clock
	fmt.Printf("Now: %s\n", clockInterface.Now()) // prints time from SetCurrentToNow()
}

func ExampleFakeClock() {
	clock := NewFakeClock()

	fmt.Printf("Now: %s\n", clock.Now()) // prints time of creation
	fmt.Printf("Now: %s\n", clock.Now()) // prints time of creation
	clock.SetCurrent(time.Unix(603288000, 0))
	fmt.Printf("Now: %s\n", clock.Now()) // prints Feb 12, 1989
	clock.Advance(24 * time.Hour)
	fmt.Printf("Now: %s\n", clock.Now()) // prints Feb 13, 1989

	var clockInterface Clock = clock              // can be assigned to Clock
	fmt.Printf("Now: %s\n", clockInterface.Now()) // prints Feb 13, 1989
}
