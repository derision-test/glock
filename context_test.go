package glock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContextValues(t *testing.T) {
	t.Run("value can be added to and retrieved from context", func(t *testing.T) {
		t.Parallel()

		clock := NewMockClock()

		ctx := context.Background()
		ctx = WithContext(ctx, clock)

		ctxClock := FromContext(ctx)
		assert.Same(t, clock, ctxClock)
	})
	t.Run("default to returning real clock", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		ctxClock := FromContext(ctx)
		assert.NotNil(t, ctxClock)
		assert.IsType(t, &realClock{}, ctxClock)
	})
}

func TestContext(t *testing.T) {
	t.Parallel()

	testContextWithTimeout(t, func(state *contextTestState) {})
}

func TestContextCancelParent(t *testing.T) {
	t.Parallel()

	testContextWithTimeout(t, func(state *contextTestState) {
		state.err1 = context.Canceled
		state.err2 = context.Canceled
		state.err3 = context.Canceled
		state.cancel1()
	})
}

func TestContextCancel(t *testing.T) {
	t.Parallel()

	testContextWithTimeout(t, func(state *contextTestState) {
		state.err2 = context.Canceled
		state.err3 = context.Canceled
		state.cancel2()
	})
}

func TestContextCancelChild(t *testing.T) {
	t.Parallel()

	testContextWithTimeout(t, func(state *contextTestState) {
		state.err3 = context.Canceled
		state.cancel3()
	})
}

func TestContextDeadline(t *testing.T) {
	t.Parallel()

	testContextWithTimeout(t, func(state *contextTestState) {
		state.err2 = context.DeadlineExceeded
		state.err3 = context.DeadlineExceeded
		state.clock.BlockingAdvance(time.Second)
	})
}

type contextTestState struct {
	clock   *MockClock
	ctx1    context.Context
	ctx2    context.Context
	ctx3    context.Context
	cancel1 context.CancelFunc
	cancel2 context.CancelFunc
	cancel3 context.CancelFunc
	err1    error
	err2    error
	err3    error
}

func testContextWithTimeout(t *testing.T, f func(state *contextTestState)) {
	clock := NewMockClock()

	ctx := context.Background()
	ctx1, cancel1 := context.WithCancel(ctx)
	ctx2, cancel2 := ContextWithTimeout(ctx1, clock, time.Second)
	ctx3, cancel3 := context.WithCancel(ctx2)
	defer cancel1()
	defer cancel2()
	defer cancel3()

	state := &contextTestState{
		clock:   clock,
		ctx1:    ctx1,
		ctx2:    ctx2,
		ctx3:    ctx3,
		cancel1: cancel1,
		cancel2: cancel2,
		cancel3: cancel3,
	}

	assertDoneAndErr(t, ctx1, nil)
	assertDoneAndErr(t, ctx2, nil)
	assertDoneAndErr(t, ctx3, nil)

	f(state)

	assertDoneAndErr(t, ctx1, state.err1)
	assertDoneAndErr(t, ctx2, state.err2)
	assertDoneAndErr(t, ctx3, state.err3)
}

func assertDoneAndErr(t *testing.T, ctx context.Context, err error) {
	if err != nil {
		eventually(t, structChanReceives(ctx.Done()))
		assert.Equal(t, ctx.Err(), err)
		return
	}

	consistently(t, structChanDoesNotReceive(ctx.Done()))
}
