package icmp

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"log/slog"
	"net"
	"os"
	"time"
)

type Transport int

type SequenceNumber uint16

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

type Socket struct {
	v4      *icmp.PacketConn
	v6      *icmp.PacketConn
	logger  *slog.Logger
	Timeout time.Duration
}

func New(tp Transport, l *slog.Logger) *Socket {
	s := Socket{
		Timeout: 5 * time.Second,
		logger:  l,
	}
	if tp&IPv4 != 0 {
		s.v4, _ = icmp.ListenPacket("udp4", "0.0.0.0")
	}
	if tp&IPv6 != 0 {
		s.v6, _ = icmp.ListenPacket("udp6", "::")
	}
	return &s
}

func (s *Socket) Ping(ip net.IP, seq SequenceNumber, ttl uint8, payload []byte) error {
	socket, tp, err := s.socket(ip)
	if err != nil {
		return err
	}
	msg := echoRequest(tp, seq, payload)
	data, _ := msg.Marshal(nil)
	if ttl != 0 {
		if err := s.setTTL(ttl); err != nil {
			return fmt.Errorf("icmp socket failed to set ttl: %w", err)
		}
	}
	s.logger.Debug("sending packet", "addr", ip, "ttl", ttl, "packet", messageLogger(msg))
	_, err = socket.WriteTo(data, &net.UDPAddr{IP: ip})
	return err
}

func (s *Socket) socket(ip net.IP) (*icmp.PacketConn, Transport, error) {
	tp := getTransport(ip)
	switch tp {
	case IPv4:
		//s.logger.Debug("selecting IPv4 socket")
		return s.v4, tp, nil
	case IPv6:
		//s.logger.Debug("selecting IPv6 socket")
		return s.v6, tp, nil
	default:
		return nil, 0, fmt.Errorf("icmp socket does not support %s", tp)
	}
}

func (s *Socket) setTTL(ttl uint8) (err error) {
	if s.v4 != nil {
		err = s.v4.IPv4PacketConn().SetTTL(int(ttl))
	}
	if s.v6 != nil {
		err = errors.Join(err, s.v6.IPv6PacketConn().SetHopLimit(int(ttl)))
	}
	return err
}

type Response struct {
	from    net.IP
	msgType icmp.Type
	body    icmp.MessageBody
}

func (s *Socket) Read(ctx context.Context) (net.IP, icmp.Type, SequenceNumber, error) {
	subCtx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	ch := make(chan Response)
	for {
		// FIXME: this leaks goroutines (& channels). Once one goroutine returns a packet, we return from this function.
		// If the other goroutine receives a packet too, it will be blocked on sending on the channel.
		if s.v4 != nil {
			go func() {
				if resp, err := s.readPacket(s.v4, IPv4); err == nil {
					ch <- resp
				}
			}()
		}
		if s.v6 != nil {
			go func() {
				if resp, err := s.readPacket(s.v6, IPv6); err == nil {
					ch <- resp
				}
			}()
		}
		select {
		case resp := <-ch:
			var seq SequenceNumber
			if body, ok := resp.body.(*icmp.Echo); ok {
				seq = SequenceNumber(body.Seq)
			}
			if isPingResponse(resp.msgType) {
				return resp.from, resp.msgType, seq, nil
			}
		case <-subCtx.Done():
			if s.v4 != nil {
				return nil, ipv4.ICMPTypeTimeExceeded, 0, subCtx.Err()
			}
			return nil, ipv6.ICMPTypeTimeExceeded, 0, subCtx.Err()
		}
	}
}

func (s *Socket) readPacket(c *icmp.PacketConn, tp Transport) (Response, error) {
	if err := c.SetReadDeadline(time.Now().Add(s.Timeout)); err != nil {
		s.logger.Warn("failed to set deadline", "err", err)
	}
	const maxPacketSize = 1500
	rb := make([]byte, maxPacketSize)
	count, from, err := c.ReadFrom(rb)
	if err != nil {
		var terr net.Error
		if errors.As(err, &terr) && terr.Timeout() {
			err = nil
		}
		return Response{}, err
	}
	msg, err := echoReply(rb[:count], tp)
	if err != nil {
		return Response{}, fmt.Errorf("parse: %w", err)
	}
	/*
		// TODO: this does not work inside a container: packet ID's seem to get overwritten
		if echo, ok := msg.Body.(*icmp.Echo); ok && echo.ID != id() {
			s.logger.Warn("discarding packet with invalid ID", "from", from, "id", echo.ID)
			return response{}, errors.New("not my packet")
		}
	*/
	s.logger.Debug("packet received", "from", from.(*net.UDPAddr).IP, "packet", messageLogger(*msg))
	return Response{
		from:    from.(*net.UDPAddr).IP,
		msgType: msg.Type,
		body:    msg.Body,
	}, nil
}

func (s *Socket) Resolve(host string) (net.IP, error) {
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
	if ip.To4() != nil {
		return IPv4
	}
	if ip.To16() != nil {
		return IPv6
	}
	return 0
}

var echoRequestTypes = map[Transport]icmp.Type{
	IPv4: ipv4.ICMPTypeEcho,
	IPv6: ipv6.ICMPTypeEchoRequest,
}

func echoRequest(tp Transport, seq SequenceNumber, payload []byte) icmp.Message {
	return icmp.Message{
		Type: echoRequestTypes[tp],
		Code: 0,
		Body: &icmp.Echo{
			ID:   id(),
			Seq:  int(seq),
			Data: payload,
		},
	}
}

func echoReply(data []byte, tp Transport) (*icmp.Message, error) {
	switch tp {
	case IPv4:
		return icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), data)
	case IPv6:
		return icmp.ParseMessage(ipv6.ICMPTypeEchoReply.Protocol(), data)
	default:
		return nil, fmt.Errorf("unknown protocol: %d", tp)
	}
}

func isPingResponse(msgType icmp.Type) bool {
	return msgType == ipv4.ICMPTypeEchoReply || msgType == ipv6.ICMPTypeEchoReply ||
		msgType == ipv4.ICMPTypeTimeExceeded || msgType == ipv6.ICMPTypeTimeExceeded
}

func id() int {
	return os.Getpid() & 0xffff
}
