package ping

import (
	"context"
	"errors"
	"github.com/clambin/pinger/pkg/ping/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"log/slog"
	"net"
	"sync"
	"time"
)

type Socket interface {
	Ping(net.IP, icmp.SequenceNumber, uint8, []byte) error
	Read(context.Context) (icmp.Response, error)
	Resolve(string) (net.IP, error)
	Serve(context.Context)
}

func Ping(ctx context.Context, targets []*Target, s Socket, interval, timeout time.Duration, l *slog.Logger) {
	responses := make(map[string]chan icmp.Response)
	for _, target := range targets {
		if target != nil && target.String() != "" {
			responses[target.String()] = make(chan icmp.Response, 1)
		}
	}
	go receiveResponses(ctx, s, responses, l)
	for _, target := range targets {
		if target != nil {
			if ch, ok := responses[target.String()]; ok {
				go pingTarget(ctx, target, s, interval, timeout, ch, l.With("addr", target.String()))
			}
		}
	}
	<-ctx.Done()
}

func pingTarget(ctx context.Context, hop *Target, s Socket, interval, timeout time.Duration, ch chan icmp.Response, l *slog.Logger) {
	sendTicker := time.NewTicker(interval)
	defer sendTicker.Stop()
	timeoutTicker := time.NewTicker(timeout)
	defer timeoutTicker.Stop()

	var packets outstandingPackets
	var seq icmp.SequenceNumber
	payload := make([]byte, 56)

	for {
		select {
		case <-sendTicker.C:
			// send a ping
			seq++
			if err := s.Ping(hop.IP, seq, uint8(64), payload); err != nil {
				l.Warn("ping failed: %v", "err", err)
			}
			// record the outgoing packet
			packets.add(seq)
			hop.Sent()
			l.Debug("packet sent", "seq", seq)
		case <-timeoutTicker.C:
			// mark any old packets as timed out
			if timedOut := packets.timeout(timeout); len(timedOut) > 0 {
				for range timedOut {
					hop.Received(false, 0)
				}
				l.Debug("packets timed out", "packets", timedOut, "current", seq)
			}
		case resp := <-ch:
			l.Debug("packet received", "packet", resp)
			// get latency for the received sequence nr. discard any old packets (we already count them during timeout)
			if latency, ok := packets.latency(resp); ok {
				// is the host up?
				up := ok && (resp.MsgType == ipv4.ICMPTypeEchoReply || resp.MsgType == ipv6.ICMPTypeEchoReply)
				// measure the state & latency
				hop.Received(up, latency)
				l.Debug("hop measured", "up", up, "latency", latency, "ok", ok)
			}
		case <-ctx.Done():
			return
		}
	}
}

func receiveResponses(ctx context.Context, s Socket, responses map[string]chan icmp.Response, l *slog.Logger) {
	for {
		response, err := s.Read(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			l.Warn("read failed", "err", err)
			continue
		}
		l.Debug("received packet", "packet", response)
		ch, ok := responses[response.From.String()]
		if !ok {
			l.Warn("no channel found for address", "packet", response)
			continue
		}
		ch <- response

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////

type outstandingPackets struct {
	packets map[icmp.SequenceNumber]time.Time
	lock    sync.Mutex
}

func (o *outstandingPackets) add(seq icmp.SequenceNumber) {
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.packets == nil {
		o.packets = make(map[icmp.SequenceNumber]time.Time)
	}
	o.packets[seq] = time.Now()
}

func (o *outstandingPackets) latency(response icmp.Response) (time.Duration, bool) {
	o.lock.Lock()
	defer o.lock.Unlock()
	seq := response.SequenceNumber()
	sent, ok := o.packets[seq]
	if ok {
		delete(o.packets, seq)
		return response.Received.Sub(sent), true
	}
	return 0, false
}

func (o *outstandingPackets) timeout(timeout time.Duration) []icmp.SequenceNumber {
	o.lock.Lock()
	defer o.lock.Unlock()
	var timedOut []icmp.SequenceNumber
	for seq, sent := range o.packets {
		if time.Since(sent) > timeout {
			delete(o.packets, seq)
			timedOut = append(timedOut, seq)
		}
	}
	return timedOut
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////

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

func (t *Target) GetStatistics() (int, int, time.Duration) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	latency := t.latencies
	if t.received > 0 {
		latency /= time.Duration(t.received)
	}
	return t.sent, t.received, latency
}
