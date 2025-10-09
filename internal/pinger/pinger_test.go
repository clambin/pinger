package pinger

import (
	"context"
	"log/slog"
	"net"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clambin/pinger/ping"
	"github.com/stretchr/testify/assert"
)

func TestPinger(t *testing.T) {
	targets := Targets{
		&Target{Name: "localhost", Host: "127.0.0.1"},
	}

	s := fakeSocket{latency: 10 * time.Millisecond}
	p := New(targets, &s, slog.New(slog.DiscardHandler))
	go p.Run(t.Context())

	assert.Eventually(t, func() bool {
		return s.received.Load() > 1
	}, 5*time.Second, 500*time.Millisecond)
	for name, stats := range targets.Statistics() {
		assert.Equal(t, "localhost", name)
		assert.NotZero(t, stats.Received)
		assert.NotZero(t, stats.Latency)
	}
}

var _ Socket = &fakeSocket{}

type fakeSocket struct {
	packets  packets
	received atomic.Uint32
	latency  time.Duration
}

func (f *fakeSocket) Serve(ctx context.Context) {
	<-ctx.Done()
}

func (f *fakeSocket) Send(ip net.IP, seq ping.SequenceNumber, _ uint8, _ []byte) error {
	f.packets.push(packet{ip: ip, seq: seq, receive: time.Now().Add(f.latency)})
	return nil
}

func (f *fakeSocket) Read(ctx context.Context) (ping.Response, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if pack, ok := f.packets.pop(); ok {
			f.received.Add(1)
			r := ping.Response{
				ResponseType: ping.ResponseEchoReply,
				Request:      ping.Request{Target: pack.ip, Seq: pack.seq, TTL: 64},
				From:         pack.ip,
				Latency:      f.latency,
			}
			return r, nil
		}
		select {
		case <-ctx.Done():
			return ping.Response{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (f *fakeSocket) Resolve(s string) (net.IP, error) {
	return net.ParseIP(s), nil
}

type packet struct {
	receive time.Time
	ip      net.IP
	seq     ping.SequenceNumber
}

type packets struct {
	queue []packet
	lock  sync.Mutex
}

func (p *packets) push(pack packet) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.queue = append(p.queue, pack)
	slices.SortFunc(p.queue, func(a, b packet) int {
		if a.receive.Before(b.receive) {
			return -1
		}
		if a.receive.After(b.receive) {
			return 1
		}
		return 0
	})
}

func (p *packets) pop() (packet, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.queue) == 0 {
		return packet{}, false
	}
	if time.Now().Before(p.queue[0].receive) {
		return packet{}, false
	}
	pack := p.queue[0]
	p.queue = p.queue[1:]
	return pack, true
}
