package handler_test

import (
	"testing"

	. "github.com/simimpact/srsim/pkg/engine/event/handler"
	"github.com/stretchr/testify/assert"
)

type testCancelEvent struct {
}

func (e testCancelEvent) Cancelled() {}

func TestCancelableEmitNoSubscription(t *testing.T) {
	var handler CancelableEventHandler[testCancelEvent]
	assert.False(t, handler.Emit(testCancelEvent{}))
}

func TestCancelableListeners(t *testing.T) {
	var handler CancelableEventHandler[testCancelEvent]

	handler.Subscribe(func(event testCancelEvent) bool {
		assert.Fail(t, "the 2nd priority listener should never have been called")
		return false
	}, 2)

	callCount := 0

	handler.Subscribe(func(event testCancelEvent) bool {
		assert.Equal(t, 1, callCount)
		callCount += 1
		return true
	}, 1)

	handler.Subscribe(func(event testCancelEvent) bool {
		callCount += 1
		return false
	}, 0)

	handler.Emit(testCancelEvent{})
	assert.Equal(t, 2, callCount)
}
