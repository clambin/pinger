// Package icmp sends and receives icmp echo request/reply packets over a UDP socket.  Both IPv4 and IPv6 are supported.
//
// A process using this package can only have one Socket instance.
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
	"sync"
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
	q       *responseQueue
	logger  *slog.Logger
	Timeout time.Duration
}

func New(tp Transport, l *slog.Logger) *Socket {
	s := Socket{
		q:       newResponseQueue(),
		logger:  l,
		Timeout: 5 * time.Second,
	}
	if tp&IPv4 != 0 {
		s.v4, _ = icmp.ListenPacket("udp4", "0.0.0.0")
	}
	if tp&IPv6 != 0 {
		s.v6, _ = icmp.ListenPacket("udp6", "::")
	}
	return &s
}

func (s *Socket) Resolve(host string) (net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", host, err)
	}

	s.logger.Debug("resolved host", "host", host, "ips", len(ips))

	for _, ip := range ips {
		tp := getTransport(ip)
		s.logger.Debug("examining IP", "ip", ip, "tp", tp)
		if (tp == IPv6 && s.v6 != nil) || tp == IPv4 && s.v4 != nil {
			s.logger.Debug("resolved IP", "ip", ip, "tp", tp)
			return ip, nil
		}
	}
	s.logger.Debug("no matching IP found")
	return nil, fmt.Errorf("no valid IP support for %s", host)
}

func (s *Socket) Serve(ctx context.Context) {
	if s.v4 != nil {
		go s.readResponses(ctx, s.v4, IPv4)
	}
	if s.v6 != nil {
		go s.readResponses(ctx, s.v6, IPv6)
	}
	<-ctx.Done()
}

func (s *Socket) readResponses(ctx context.Context, socket *icmp.PacketConn, tp Transport) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if response, err := readPacket(socket, tp, s.Timeout, s.logger.With("transport", tp)); err == nil {
				s.q.push(response)
			}
		}
	}
}

func readPacket(c *icmp.PacketConn, tp Transport, timeout time.Duration, l *slog.Logger) (Response, error) {
	if err := c.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		l.Warn("failed to set deadline", "err", err)
	}
	const maxPacketSize = 1500
	rb := make([]byte, maxPacketSize)
	count, from, err := c.ReadFrom(rb)
	if err != nil {
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
	l.Debug("packet received", "from", from.(*net.UDPAddr).IP, "packet", messageLogger(*msg))
	return Response{
		From:     from.(*net.UDPAddr).IP,
		MsgType:  msg.Type,
		Body:     msg.Body,
		Received: time.Now(),
	}, nil
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

func (s *Socket) Read(ctx context.Context) (Response, error) {
	subCtx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	for {
		r, err := s.q.popWait(subCtx)
		if err != nil {
			return Response{}, errors.New("timeout waiting for response")
		}

		if isPingResponse(r.MsgType) {
			return r, nil
		}
	}
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

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ slog.LogValuer = Response{}

type Response struct {
	Received time.Time
	MsgType  icmp.Type
	Body     icmp.MessageBody
	From     net.IP
}

func (r Response) SequenceNumber() SequenceNumber {
	if body, ok := r.Body.(*icmp.Echo); ok {
		return SequenceNumber(body.Seq)
	}
	return 0
}

func (r Response) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("from", r.From.String()),
		slog.Any("msgType", r.MsgType),
		slog.Any("seq", r.SequenceNumber()),
	)
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type responseQueue struct {
	notEmpty sync.Cond
	queue    []Response
	lock     sync.Mutex
}

func newResponseQueue() *responseQueue {
	q := &responseQueue{}
	q.notEmpty.L = &q.lock
	return q
}

func (q *responseQueue) push(r Response) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.queue = append(q.queue, r)
	q.notEmpty.Broadcast()
}

func (q *responseQueue) pop() (Response, bool) {
	q.lock.Lock()
	defer q.lock.Unlock()
	if len(q.queue) == 0 {
		return Response{}, false
	}
	r := q.queue[0]
	q.queue = q.queue[1:]
	return r, true
}

func (q *responseQueue) len() int {
	q.lock.Lock()
	defer q.lock.Unlock()
	return len(q.queue)
}

func (q *responseQueue) popWait(ctx context.Context) (Response, error) {
	for {
		if resp, ok := q.pop(); ok {
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
			return Response{}, ctx.Err()
		case <-notEmpty:
		}
	}
}
