package ping

import (
	"net"
	"sync"
	"time"
)

type Target struct {
	net.IP
	sent      int
	received  int
	latencies time.Duration
	lock      sync.RWMutex
}

func (t *Target) Sent() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.sent++
}

func (t *Target) Received(received bool, latency time.Duration) {
	if received {
		t.lock.Lock()
		defer t.lock.Unlock()
		t.received++
		t.latencies += latency
	}
}

func (t *Target) Statistics() (int, int, time.Duration) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	latency := t.latencies
	if t.received > 0 {
		latency /= time.Duration(t.received)
	}
	return t.sent, t.received, latency
}

func (t *Target) ResetStatistics() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.sent = 0
	t.received = 0
	t.latencies = 0
}
