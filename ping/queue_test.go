package ping

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	const n = 100_000
	q := newQueue[int32]()
	responses := make(map[int32]struct{}, n)
	var seq int32

	for range 1_000 {
		go func() {
			for {
				val := atomic.AddInt32(&seq, 1)
				if val > n {
					return
				}
				q.Push(val)
				time.Sleep(time.Millisecond)
			}
		}()
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)

	for range n {
		value, err := q.PopWait(ctx)
		if err != nil {
			t.Fatalf("failed to pop: %v", err)
		}
		_, ok := responses[value]
		if ok {
			t.Fatalf("received duplicate response: %d", value)
		}
		responses[value] = struct{}{}
	}
	if len(responses) != n {
		t.Fatalf("received %d responses, expected %d", len(responses), n)
	}
}
