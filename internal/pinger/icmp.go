package pinger

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"time"
)

type Transport int

const (
	IPv4 Transport = 0x01
	IPv6 Transport = 0x02
)

func (tp Transport) String() string {
	switch tp {
	case IPv4:
		return "ipv4"
	case IPv6:
		return "ipv6"
	default:
		return "unknown"
	}
}

type icmpSocket struct {
	Timeout time.Duration
	OnReply func(net.IP, *icmp.Echo)
	v4      *icmp.PacketConn
	v6      *icmp.PacketConn
	logger  *slog.Logger
}

func newICMPSocket(l *slog.Logger) *icmpSocket {
	s := icmpSocket{
		Timeout: 5 * time.Second,
		logger:  l,
	}
	s.v4, _ = icmp.ListenPacket("udp4", "0.0.0.0")
	s.v6, _ = icmp.ListenPacket("udp6", "::")
	return &s
}

func (s *icmpSocket) ping(ip net.IP, seq int, payload []byte) error {
	var socket *icmp.PacketConn
	tp := getTransport(ip)
	switch tp {
	case IPv4:
		socket = s.v4
	case IPv6:
		socket = s.v6
	default:
		return fmt.Errorf("icmp socket does not support %s", tp)
	}
	msg := echoRequest(tp, seq, payload)
	data, _ := msg.Marshal(nil)
	_, err := socket.WriteTo(data, &net.UDPAddr{IP: ip})
	return err
}

func (s *icmpSocket) listen(ctx context.Context) error {
	s.logger.Debug("socket listening")
	var g errgroup.Group
	if s.v4 != nil {
		g.Go(func() error { return s.serve(ctx, s.v4, IPv4) })
	}
	if s.v6 != nil {
		g.Go(func() error { return s.serve(ctx, s.v6, IPv6) })
	}
	<-ctx.Done()
	s.logger.Debug("socket stopping")
	defer s.logger.Debug("socket stopped")
	return g.Wait()
}

func (s *icmpSocket) serve(ctx context.Context, c *icmp.PacketConn, tp Transport) error {
	s.logger.Debug("starting ICMP packet listener", "transport", tp.String())
	defer s.logger.Debug("stopping ICMP packet listener", "transport", tp.String())

	ch := make(chan response)
	go s.read(ctx, c, tp, ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case resp := <-ch:
			s.OnReply(resp.from, resp.echo)
		}
	}
}

type response struct {
	from net.IP
	echo *icmp.Echo
}

func (s *icmpSocket) read(ctx context.Context, c *icmp.PacketConn, tp Transport, ch chan<- response) {
	for {
		resp, err := s.readPacket(c, tp)
		if err == nil {
			ch <- resp
		} else {
			var terr net.Error
			if !errors.Is(err, errNotICMPTypeEchoReply) && (!errors.As(err, &terr) || terr.Timeout()) {
				s.logger.Error("failed to read icmp packet", "err", err, "transport", tp.String())
			}
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

var errNotICMPTypeEchoReply = errors.New("non-relevant ICMP packet")

func (s *icmpSocket) readPacket(c *icmp.PacketConn, tp Transport) (response, error) {
	if err := c.SetReadDeadline(time.Now().Add(s.Timeout)); err != nil {
		s.logger.Warn("failed to set deadline", "err", err)
	}
	rb := make([]byte, 1500)
	count, from, err := c.ReadFrom(rb)
	if err != nil {
		return response{}, fmt.Errorf("read: %w", err)
	}
	msg, err := echoReply(rb[:count], tp)
	if err != nil {
		return response{}, fmt.Errorf("parse: %w", err)
	}
	if msg.Type != ipv4.ICMPTypeEchoReply && msg.Type != ipv6.ICMPTypeEchoReply {
		return response{}, errNotICMPTypeEchoReply
	}
	return response{from: from.(*net.UDPAddr).IP, echo: msg.Body.(*icmp.Echo)}, nil
}

func (s *icmpSocket) resolve(host string) (net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", host, err)
	}

	for _, ip := range ips {
		if tp := getTransport(ip); (tp == IPv6 && s.v6 != nil) || tp == IPv4 && s.v4 != nil {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no valid IP support for %s", host)
}

func getTransport(ip net.IP) Transport {
	if len(ip.To4()) == 0 {
		return IPv6
	}
	return IPv4
}

var echoRequestTypes = map[Transport]icmp.Type{
	IPv4: ipv4.ICMPTypeEcho,
	IPv6: ipv6.ICMPTypeEchoRequest,
}

func echoRequest(tp Transport, seq int, payload []byte) icmp.Message {
	return icmp.Message{
		Type: echoRequestTypes[tp],
		Code: 0,
		Body: &icmp.Echo{
			ID:   1,
			Seq:  seq,
			Data: payload,
		},
	}
}

func echoReply(data []byte, tp Transport) (*icmp.Message, error) {
	var protocol int
	if tp&IPv4 != 0 {
		protocol = 1
	} else if tp&IPv6 != 0 {
		protocol = 58
	} else {
		return nil, fmt.Errorf("unknown protocol: %d", tp)
	}
	return icmp.ParseMessage(protocol, data)
}
