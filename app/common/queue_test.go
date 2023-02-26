package common_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/pinebit/lognite/app/common"
	"github.com/stretchr/testify/assert"
)

func TestQueue(t *testing.T) {
	var discardedCount atomic.Int32
	received := make(chan int)
	q := common.NewQueue[int](3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer close(received)

	go q.RunDequeueLoop(ctx, func(ctx context.Context, item int, discarded bool) {
		if discarded {
			discardedCount.Add(1)
		} else {
			received <- item
		}
	})

	// Adding 5 items while capacity is 3.
	// Two items should be discarded.
	for i := 0; i < 5; i++ {
		q.Enqueue(i)
	}

	for i := 0; i < 3; i++ {
		assert.Equal(t, i, <-received)
	}

	q.Enqueue(7)
	assert.Equal(t, 7, <-received)

	assert.Equal(t, int32(2), discardedCount.Load())
}
