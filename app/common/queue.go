package common

import (
	"context"
	"sync/atomic"
)

// User handler to process a dequeued or discarded item.
// Implementaion must respect context and be thread-safe.
type HandleFunc[T any] func(ctx context.Context, item T, discarded bool)

// Queue with capacity.
// Designed for multiple producers single consumer.
type Queue[T any] interface {
	// Thread-safe non-blocking function.
	// Adds item T to the queue. Discards item if the queue is full.
	// Would panic if RunDequeueLoop exited.
	Enqueue(t T)
	// Starts dequeue loop. Blocking call.
	RunDequeueLoop(ctx context.Context, handler HandleFunc[T])
}

type queue[T any] struct {
	receivingCh chan T
	capacity    uint
}

func NewQueue[T any](capacity uint) Queue[T] {
	return &queue[T]{
		receivingCh: make(chan T),
		capacity:    capacity,
	}
}

func (q queue[T]) Enqueue(t T) {
	q.receivingCh <- t
}

func (q *queue[T]) RunDequeueLoop(ctx context.Context, handler HandleFunc[T]) {
	var count atomic.Int32
	queueCh := make(chan T, q.capacity)

	defer close(q.receivingCh)
	defer close(queueCh)

	go func() {
		for item := range queueCh {
			handler(ctx, item, false)
			count.Add(-1)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-q.receivingCh:
			if !ok {
				return
			}
			if uint(count.Load()) < q.capacity {
				count.Add(1)
				queueCh <- item
			} else {
				handler(ctx, item, true)
			}
		}
	}
}
