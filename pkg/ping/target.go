package ping

import (
	"github.com/clambin/pinger/pkg/ping/icmp"
	"net"
	"sync"
	"time"
)

type Target struct {
	outstandingPackets map[icmp.SequenceNumber]time.Time
	net.IP
	sent      int
	received  int
	latencies time.Duration
	lock      sync.RWMutex
}

func (t *Target) Sent(seq icmp.SequenceNumber) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.sent++
	if t.outstandingPackets == nil {
		t.outstandingPackets = make(map[icmp.SequenceNumber]time.Time)
	}
	t.outstandingPackets[seq] = time.Now()
}

func (t *Target) Received(received bool, seq icmp.SequenceNumber) {
	if received {
		t.lock.Lock()
		defer t.lock.Unlock()
		if timeSent, ok := t.outstandingPackets[seq]; ok {
			t.received++
			t.latencies += time.Since(timeSent)
			delete(t.outstandingPackets, seq)
		}
	}
}

func (t *Target) timeout(timeout time.Duration) []icmp.SequenceNumber {
	t.lock.Lock()
	defer t.lock.Unlock()
	timedOut := make([]icmp.SequenceNumber, 0, len(t.outstandingPackets))
	for seq, timeSent := range t.outstandingPackets {
		if time.Now().After(timeSent.Add(timeout)) {
			timedOut = append(timedOut, seq)
			delete(t.outstandingPackets, seq)
		}
	}
	return timedOut
}

type Statistics struct {
	Sent     int
	Received int
	Latency  time.Duration
}

func (t *Target) Statistics() Statistics {
	t.lock.RLock()
	defer t.lock.RUnlock()
	sent, received, latency := t.sent, t.received, t.latencies
	if t.received > 0 {
		latency /= time.Duration(received)
	}
	if received > sent {
		received = sent
	}
	return Statistics{Sent: sent, Received: received, Latency: latency}
}

func (t *Target) ResetStatistics() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.sent = 0
	t.received = 0
	t.latencies = 0
}
