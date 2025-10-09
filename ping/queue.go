package ping

import (
	"context"
	"sync"
)

type queue[T any] struct {
	notEmpty sync.Cond
	queue    []T
	lock     sync.Mutex
}

func newQueue[T any]() *queue[T] {
	var q queue[T]
	q.notEmpty.L = &q.lock
	return &q
}

func (q *queue[T]) Push(val T) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.queue = append(q.queue, val)
	q.notEmpty.Broadcast()
}

func (q *queue[T]) Pop() (value T, ok bool) {
	q.lock.Lock()
	defer q.lock.Unlock()
	if len(q.queue) == 0 {
		return value, false
	}
	value = q.queue[0]
	q.queue = q.queue[1:]
	return value, true
}

func (q *queue[T]) PopWait(ctx context.Context) (value T, err error) {
	for {
		if resp, ok := q.Pop(); ok {
			return resp, nil
		}
		notEmpty := make(chan struct{})
		go func() {
			q.lock.Lock()
			q.notEmpty.Wait()
			q.lock.Unlock()
			notEmpty <- struct{}{}
		}()
		select {
		case <-ctx.Done():
			return value, ctx.Err()
		case <-notEmpty:
		}
	}
}
