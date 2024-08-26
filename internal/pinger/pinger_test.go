package pinger

import (
	"context"
	"github.com/clambin/pinger/pkg/ping"
	icmp2 "github.com/clambin/pinger/pkg/ping/icmp"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"log/slog"
	"net"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPinger(t *testing.T) {
	targets := Targets{
		{Name: "", Host: "127.0.0.1"},
	}

	p := New(targets, 0, slog.Default())

	s := fakeSocket{latency: 10 * time.Millisecond}
	p.conn = &s

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	go func() {
		p.Run(ctx)
		ch <- struct{}{}
	}()

	assert.Eventually(t, func() bool {
		return s.received.Load() > 1
	}, 5*time.Second, time.Second)

	stats := p.Statistics()
	//t.Log(stats)
	ipv4Stats, ok := stats["127.0.0.1"]
	assert.True(t, ok)
	assert.NotZero(t, ipv4Stats.Received)

	cancel()
	<-ch
}

var _ ping.Socket = &fakeSocket{}

type fakeSocket struct {
	packets  packets
	received atomic.Uint32
	latency  time.Duration
}

func (f *fakeSocket) Serve(ctx context.Context) {
	<-ctx.Done()
}

func (f *fakeSocket) Ping(ip net.IP, seq icmp2.SequenceNumber, _ uint8, _ []byte) error {
	f.packets.push(packet{ip: ip, seq: seq, receive: time.Now().Add(f.latency)})
	return nil
}

func (f *fakeSocket) Read(ctx context.Context) (icmp2.Response, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if pack, ok := f.packets.pop(); ok {
			f.received.Add(1)
			r := icmp2.Response{
				From:     pack.ip,
				Body:     &icmp.Echo{Seq: int(pack.seq)},
				Received: time.Now(),
			}
			if pack.ip.To4() == nil {
				// not an IPv4 address. must be IPv6
				r.MsgType = ipv6.ICMPTypeEchoReply
				return r, nil
			}
			r.MsgType = ipv4.ICMPTypeEchoReply
			return r, nil
		}
		select {
		case <-ctx.Done():
			return icmp2.Response{}, ctx.Err()
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
	seq     icmp2.SequenceNumber
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
