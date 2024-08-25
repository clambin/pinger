package ping

import (
	"context"
	"errors"
	"github.com/clambin/pinger/pkg/ping/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"log/slog"
	"net"
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

func pingTarget(ctx context.Context, target *Target, s Socket, interval, timeout time.Duration, ch chan icmp.Response, l *slog.Logger) {
	sendTicker := time.NewTicker(interval)
	defer sendTicker.Stop()
	timeoutTicker := time.NewTicker(timeout)
	defer timeoutTicker.Stop()

	var seq icmp.SequenceNumber
	payload := make([]byte, 56)

	for {
		select {
		case <-sendTicker.C:
			// send a ping
			seq++
			if err := s.Ping(target.IP, seq, uint8(64), payload); err != nil {
				l.Warn("ping failed: %v", "err", err)
			}
			// record the outgoing packet
			target.Sent(seq)
			l.Debug("packet sent", "seq", seq)
		case <-timeoutTicker.C:
			// mark any old packets as timed out
			timedOut := target.timeout(timeout)
			l.Debug("packets timed out", "current", seq, "packets", timedOut)
		case resp := <-ch:
			// get latency for the received sequence nr. discard any old packets (we already count them during timeout)
			l.Debug("packet received", "packet", resp)
			// is the host up?
			up := resp.MsgType == ipv4.ICMPTypeEchoReply || resp.MsgType == ipv6.ICMPTypeEchoReply
			// measure the state & latency
			target.Received(up, resp.SequenceNumber())
			l.Debug("target measured", "up", up)

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
