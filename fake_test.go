package glock

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

type FakeSuite struct{}

func (s *FakeSuite) TestNewFakeSuite(t *testing.T) {
	// Since we can't know the exact time that something gets created
	// at least make sure a new fake clock picks up a time AROUND now
	// instead of an exact time
	clock := NewFakeClock()

	since := time.Since(clock.fakeTime)
	Expect(since).To(BeNumerically("<", 100*time.Millisecond))
}

func (s *FakeSuite) TestNow(t *testing.T) {
	clock := NewFakeClock()

	Expect(clock.Now()).To(Equal(clock.fakeTime))
}

func (s *FakeSuite) TestSetCurrent(t *testing.T) {
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(100, 0))

	Expect(clock.Now()).To(Equal(time.Unix(100, 0)))
}

func (s *FakeSuite) TestAdvance(t *testing.T) {
	clock := NewFakeClock()

	clock.SetCurrent(time.Unix(100, 0))
	Expect(clock.Now()).To(Equal(time.Unix(100, 0)))

	clock.Advance(1 * time.Second)
	Expect(clock.Now()).To(Equal(time.Unix(101, 0)))

	clock.Advance(1 * time.Hour)
	Expect(clock.Now()).To(Equal(time.Unix(3701, 0)))
}

func (s *FakeSuite) TestAfter(t *testing.T) {
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(0, 0))

	after := clock.After(1 * time.Second)
	Consistently(after).ShouldNot(Receive())

	clock.Advance(500 * time.Millisecond)
	Consistently(after).ShouldNot(Receive())

	clock.Advance(250 * time.Millisecond)
	Consistently(after).ShouldNot(Receive())

	clock.Advance(250 * time.Millisecond)
	Eventually(after).Should(Receive(Equal(time.Unix(1, 0))))

	// With the real clock the channel will block after receiving
	// the "after" value
	Consistently(after).ShouldNot(Receive())
}

func (s *FakeSuite) TestMultipleAfter(t *testing.T) {
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(0, 0))

	a1 := clock.After(1 * time.Second)
	a2 := clock.After(2 * time.Second)

	Consistently(a1).ShouldNot(Receive())
	Consistently(a2).ShouldNot(Receive())

	clock.Advance(500 * time.Millisecond)
	Consistently(a1).ShouldNot(Receive())
	Consistently(a2).ShouldNot(Receive())

	clock.Advance(500 * time.Millisecond)
	Eventually(a1).Should(Receive(Equal(time.Unix(1, 0))))
	Consistently(a2).ShouldNot(Receive())

	clock.Advance(500 * time.Millisecond)
	Consistently(a1).ShouldNot(Receive())
	Consistently(a2).ShouldNot(Receive())

	clock.Advance(500 * time.Millisecond)
	Consistently(a1).ShouldNot(Receive())
	Eventually(a2).Should(Receive(Equal(time.Unix(2, 0))))

	clock.Advance(500 * time.Millisecond)
	Consistently(a1).ShouldNot(Receive())
	Consistently(a2).ShouldNot(Receive())
}

func (s *FakeSuite) TestAfterNotExact(t *testing.T) {
	// Make sure triggers are still fired even if the
	// time doesn't match up exactly with the trigger
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(0, 0))

	a1 := clock.After(1 * time.Second)
	a2 := clock.After(2 * time.Second)

	clock.Advance(3 * time.Second)
	Eventually(a1).Should(Receive(Equal(time.Unix(1, 0))))
	Eventually(a2).Should(Receive(Equal(time.Unix(2, 0))))
}

func (s *FakeSuite) TestAfterTriggersSorted(t *testing.T) {
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(0, 0))

	clock.After(3 * time.Second)
	Expect(clock.triggers).To(HaveLen(1))
	Expect(clock.triggers[0].trigger).To(Equal(time.Unix(3, 0)))

	clock.After(1 * time.Second)
	Expect(clock.triggers).To(HaveLen(2))
	Expect(clock.triggers[0].trigger).To(Equal(time.Unix(1, 0)))
	Expect(clock.triggers[1].trigger).To(Equal(time.Unix(3, 0)))

	clock.After(2 * time.Second)
	Expect(clock.triggers).To(HaveLen(3))
	Expect(clock.triggers[0].trigger).To(Equal(time.Unix(1, 0)))
	Expect(clock.triggers[1].trigger).To(Equal(time.Unix(2, 0)))
	Expect(clock.triggers[2].trigger).To(Equal(time.Unix(3, 0)))
}

func (s *FakeSuite) TestRemoveAfterTrigger(t *testing.T) {
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(0, 0))

	clock.After(1 * time.Second)
	clock.After(2 * time.Second)
	clock.After(3 * time.Second)
	Expect(clock.triggers).To(HaveLen(3))
	Expect(clock.triggers[0].trigger).To(Equal(time.Unix(1, 0)))
	Expect(clock.triggers[1].trigger).To(Equal(time.Unix(2, 0)))
	Expect(clock.triggers[2].trigger).To(Equal(time.Unix(3, 0)))

	clock.Advance(1 * time.Second)
	Expect(clock.triggers).To(HaveLen(2))
	Expect(clock.triggers[0].trigger).To(Equal(time.Unix(2, 0)))
	Expect(clock.triggers[1].trigger).To(Equal(time.Unix(3, 0)))

	clock.Advance(1 * time.Second)
	Expect(clock.triggers).To(HaveLen(1))
	Expect(clock.triggers[0].trigger).To(Equal(time.Unix(3, 0)))

	clock.Advance(1 * time.Second)
	Expect(clock.triggers).To(HaveLen(0))
}

func (s *FakeSuite) TestSleep(t *testing.T) {
	clock := NewFakeClock()
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

	Consistently(finished).ShouldNot(Receive())

	clock.Advance(500 * time.Millisecond)
	Consistently(finished).ShouldNot(Receive())

	clock.Advance(500 * time.Millisecond)
	Eventually(finished).Should(Receive(Equal(time.Unix(1, 0))))
}

func (s *FakeSuite) TestNewTickerNoDuration(t *testing.T) {
	clock := NewFakeClock()

	Expect(func() {
		clock.NewTicker(0)
	}).To(Panic())
}

func (s *FakeSuite) TestTickerOnTime(t *testing.T) {
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(0, 0))

	ticker := clock.NewTicker(2 * time.Second)
	Consistently(ticker.Chan()).ShouldNot(Receive())

	clock.Advance(1 * time.Second)
	Consistently(ticker.Chan()).ShouldNot(Receive())

	clock.Advance(1 * time.Second)
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(2, 0))))

	clock.Advance(1 * time.Second)
	Consistently(ticker.Chan()).ShouldNot(Receive())

	clock.Advance(1 * time.Second)
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(4, 0))))

	clock.Advance(2 * time.Second)
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(6, 0))))
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
func (s *FakeSuite) TestTickerOffset2Second(t *testing.T) {
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(0, 0))

	ticker := clock.NewTicker(2 * time.Second)
	Consistently(ticker.Chan()).ShouldNot(Receive())

	clock.Advance(2 * time.Second) // 2s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(2, 0))))
	clock.Advance(3 * time.Second) // 5s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(4, 0))))
	clock.Advance(1 * time.Second) // 6s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(6, 0))))
	clock.Advance(7 * time.Second) // 13s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(8, 0))))
	clock.Advance(7 * time.Second) // 20s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(14, 0))))
	clock.Advance(7 * time.Second) // 27s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(22, 0))))
	clock.Advance(1 * time.Second) // 28s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(28, 0))))
	clock.Advance(1 * time.Second) // 29s
	Consistently(ticker.Chan()).ShouldNot(Receive())
	clock.Advance(1 * time.Second) // 30s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(30, 0))))
	clock.Advance(6 * time.Second) // 36s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(32, 0))))
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
func (s *FakeSuite) TestTickerOffset3Second(t *testing.T) {
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(0, 0))

	ticker := clock.NewTicker(3 * time.Second)
	Consistently(ticker.Chan()).ShouldNot(Receive())

	clock.Advance(3 * time.Second) // 3s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(3, 0))))
	clock.Advance(3 * time.Second) // 6s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(6, 0))))
	clock.Advance(1 * time.Second) // 7s
	Consistently(ticker.Chan()).ShouldNot(Receive())
	clock.Advance(2 * time.Second) // 9s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(9, 0))))
	clock.Advance(7 * time.Second) // 16s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(12, 0))))
	clock.Advance(7 * time.Second) // 23s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(18, 0))))
	clock.Advance(7 * time.Second) // 30s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(24, 0))))
	clock.Advance(1 * time.Second) // 31s
	Consistently(ticker.Chan()).ShouldNot(Receive())
	clock.Advance(2 * time.Second) // 33s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(33, 0))))
	clock.Advance(1 * time.Second) // 34s
	Consistently(ticker.Chan()).ShouldNot(Receive())
	clock.Advance(1 * time.Second) // 35s
	Consistently(ticker.Chan()).ShouldNot(Receive())
	clock.Advance(1 * time.Second) // 36s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(36, 0))))
	clock.Advance(6 * time.Second) // 42s
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(39, 0))))
}

func (s *FakeSuite) TestMultipleTickers(t *testing.T) {
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(0, 0))

	t2 := clock.NewTicker(2 * time.Second)
	t3 := clock.NewTicker(3 * time.Second)

	clock.Advance(1 * time.Second)
	Consistently(t2.Chan()).ShouldNot(Receive())
	Consistently(t3.Chan()).ShouldNot(Receive())

	clock.Advance(1 * time.Second)
	Eventually(t2.Chan()).Should(Receive(Equal(time.Unix(2, 0))))
	Consistently(t3.Chan()).ShouldNot(Receive())

	clock.Advance(1 * time.Second)
	Consistently(t2.Chan()).ShouldNot(Receive())
	Eventually(t3.Chan()).Should(Receive(Equal(time.Unix(3, 0))))

	clock.Advance(1 * time.Second)
	Eventually(t2.Chan()).Should(Receive(Equal(time.Unix(4, 0))))
	Consistently(t3.Chan()).ShouldNot(Receive())

	clock.Advance(1 * time.Second)
	Consistently(t2.Chan()).ShouldNot(Receive())
	Consistently(t3.Chan()).ShouldNot(Receive())

	clock.Advance(1 * time.Second)
	Eventually(t2.Chan()).Should(Receive(Equal(time.Unix(6, 0))))
	Eventually(t3.Chan()).Should(Receive(Equal(time.Unix(6, 0))))
}

func (s *FakeSuite) TestTickerStopped(t *testing.T) {
	clock := NewFakeClock()
	clock.SetCurrent(time.Unix(0, 0))

	ticker := clock.NewTicker(2 * time.Second)

	clock.Advance(2 * time.Second)
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(2, 0))))
	clock.Advance(2 * time.Second)
	Eventually(ticker.Chan()).Should(Receive(Equal(time.Unix(4, 0))))

	ticker.Stop()

	clock.Advance(2 * time.Second)
	Consistently(ticker.Chan()).ShouldNot(Receive())
	clock.Advance(2 * time.Second)
	Consistently(ticker.Chan()).ShouldNot(Receive())
}
