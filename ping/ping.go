// Package ping sends and receives icmp echo request/reply packets over a UDP socket.
// Both IPv4 and IPv6 are supported.
package ping

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	// defaultReadTimeout is the default timeout for reading icmp packets from the socket.
	defaultReadTimeout = 5 * time.Second
	// timeoutInterval determines how often we check for expired outstanding requests.
	timeoutInterval = 2 * time.Second
)

var (
	ErrTimeout = errors.New("timeout waiting for response")
	//errIncorrectID = errors.New("packet ignored: incorrect ID")
)

// errIncorrectID is an error returned when an icmp packet is received with an incorrect ID.
type errIncorrectID struct {
	id int
}

func (e errIncorrectID) Error() string {
	return fmt.Sprintf("incorrect ID: %d", e.id)
}

// The nextID variable is used to generate unique IDs for icmp packets sent by each Socket instance.
// This allows us to run multiple Socket instances in parallel without interfering with each other.
var nextID = uint32(os.Getpid())

// SequenceNumber represents the sequence number of an icmp packet.
type SequenceNumber uint16

var _ slog.LogValuer = Response{}

// Response represents an icmp packet received by the Socket.
type Response struct {
	From         net.IP
	Request      Request
	ResponseType ResponseType
	Latency      time.Duration
}

func (r Response) LogValue() slog.Value {
	attrs := []slog.Attr{slog.String("type", r.ResponseType.String())}
	if r.ResponseType != ResponseTimeout {
		attrs = append(attrs,
			slog.String("from", r.From.String()),
			slog.String("target", r.Request.Target.String()),
			slog.String("seq", fmt.Sprintf("%d", r.Request.Seq)),
			slog.String("ttl", fmt.Sprintf("%d", r.Request.TTL)),
		)
	}
	return slog.GroupValue(attrs...)
}

// Request represents an icmp packet sent by the Socket.
type Request struct {
	TimeSent time.Time
	Target   net.IP
	Seq      SequenceNumber
	TTL      uint8
}

const (
	ResponseEchoReply ResponseType = iota
	ResponseTimeExceeded
	ResponseTimeout
)

type ResponseType int

func (rt ResponseType) String() string {
	switch rt {
	case ResponseEchoReply:
		return "echo reply"
	case ResponseTimeExceeded:
		return "time exceeded"
	case ResponseTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

type Socket struct {
	v4                  *icmp.PacketConn
	v6                  *icmp.PacketConn
	q                   *queue[Response]
	logger              *slog.Logger
	outstandingRequests map[SequenceNumber]Request
	Timeout             time.Duration
	lock                sync.Mutex
	id                  uint16
	checkID             bool
}

// New creates a new Socket instance.
func New(opts ...SocketOption) (*Socket, error) {
	s := Socket{
		q:                   newQueue[Response](),
		logger:              slog.Default(),
		Timeout:             defaultReadTimeout,
		id:                  uint16(atomic.AddUint32(&nextID, 1) & 0xffff),
		outstandingRequests: make(map[SequenceNumber]Request),
		checkID:             true,
	}
	var errs error
	for _, opt := range opts {
		if err := opt(&s); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return &s, errs
}

type SocketOption func(*Socket) error

func WithIPv4() SocketOption {
	return func(s *Socket) error {
		var err error
		s.v4, err = icmp.ListenPacket("udp4", "0.0.0.0")
		return err
	}
}

func WithIPv6() SocketOption {
	return func(s *Socket) error {
		var err error
		s.v6, err = icmp.ListenPacket("udp6", "::")
		return err
	}
}

func WithLogger(l *slog.Logger) SocketOption {
	return func(s *Socket) error {
		s.logger = l
		return nil
	}
}

func WithTimeout(d time.Duration) SocketOption {
	return func(s *Socket) error {
		s.Timeout = d
		return nil
	}
}

func WithoutCheckID() SocketOption {
	return func(s *Socket) error {
		s.checkID = false
		return nil
	}
}

// Resolve resolves the provided host to an IP address and returns it.
// Resolve returns an error if the host does not have a valid IP address of a type supported by the socket
// (e.g., if the socket only supports IPv6, but the host doesn't have an IPv4 address).
func (s *Socket) Resolve(host string) (net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", host, err)
	}

	for _, ip := range ips {
		s.logger.Debug("examining IP", "ip", ip, "s.v4", s.v4 != nil, "s.v6", s.v6 != nil)
		switch {
		// order is important here: ip.To16 returns an IPv4 address if ip is an IPv4 address
		case ip.To4() != nil:
			if s.v4 != nil {
				return ip, nil
			}
		case ip.To16() != nil:
			if s.v6 != nil {
				return ip, nil
			}
		}
	}
	return nil, fmt.Errorf("no IP support for %s", host)
}

// Send creates an icmp packet with the provided seq, ttl and payload and sends it to the specified target.
func (s *Socket) Send(target net.IP, seq SequenceNumber, ttl uint8, payload []byte) error {
	// we're setting socket options, so only send one packet at a time
	s.lock.Lock()
	defer s.lock.Unlock()

	// get the right socket & request type for the target's IP type (ipv4 or ipv6)
	var socket *icmp.PacketConn
	var requestType icmp.Type
	switch {
	case target.To4() != nil:
		socket = s.v4
		requestType = ipv4.ICMPTypeEcho
	case target.To16() != nil:
		socket = s.v6
		requestType = ipv6.ICMPTypeEchoRequest
	default:
		return fmt.Errorf("unable to determine IP version for %q", target)
	}

	// create the ICMP echo Request message
	msg := icmp.Message{
		Type: requestType,
		Body: &icmp.Echo{
			ID:   int(s.id),
			Seq:  int(seq),
			Data: payload,
		},
	}
	data, _ := msg.Marshal(nil)

	// if ttl is specified, set it on the socket
	if ttl != 0 {
		if err := s.setTTL(ttl); err != nil {
			return fmt.Errorf("icmp socket failed to set ttl: %w", err)
		}
	}

	// send the packet
	s.logger.Debug("sending packet", "addr", target, "ttl", ttl)
	if _, err := socket.WriteTo(data, &net.UDPAddr{IP: target}); err != nil {
		return err
	}

	// mark an outstanding packet for seq & time sent
	s.outstandingRequests[seq] = Request{
		Target:   target,
		TTL:      ttl,
		Seq:      seq,
		TimeSent: time.Now(),
	}
	return nil
}

// Read reads the next icmp packet from the socket.
// It blocks until a packet is received or the context is canceled.
func (s *Socket) Read(ctx context.Context) (Response, error) {
	subCtx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	r, err := s.q.PopWait(subCtx)
	if err != nil {
		return Response{}, ErrTimeout
	}
	return r, nil
}

// Serve listens for icmp packets on the socket and dispatches them to the appropriate handler.
// It's the responsibility of the caller to call Serve before sending or receiving packets.
// Serve blocks until the context is canceled.
func (s *Socket) Serve(ctx context.Context) {
	ch := make(chan Response)
	if s.v4 != nil {
		go s.readPackets(ctx, s.v4, "IPv4", ch)
	}
	if s.v6 != nil {
		go s.readPackets(ctx, s.v6, "IPv6", ch)
	}
	timeoutTicker := time.NewTicker(timeoutInterval)
	defer timeoutTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeoutTicker.C:
			s.timeout()
		case resp := <-ch:
			s.lock.Lock()
			// process the response:
			// if not an outstanding packet, drop it
			if _, ok := s.outstandingRequests[resp.Request.Seq]; !ok {
				s.logger.Debug("ignoring packet", "seq", resp.Request.Seq)
			} else {
				// queue for delivery by Receive and remove the outstanding packet
				s.q.Push(resp)
			}
			s.lock.Unlock()
		}
	}
}

// readPackets reads packets from the provided socket and parses the ICMP response.
func (s *Socket) readPackets(ctx context.Context, socket *icmp.PacketConn, tp string, ch chan Response) {
	logger := s.logger.With("transport", tp)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			response, err := s.readPacket(socket)
			var err2 errIncorrectID
			if errors.As(err, &err2) {
				logger.Debug("ignoring received packet", "err", err2, "id", s.id)
				continue
			}
			if err != nil {
				logger.Warn("failed to read packet", "err", err)
				break
			}
			ch <- response
		}
	}
}

func (s *Socket) readPacket(socket *icmp.PacketConn) (Response, error) {
	if err := socket.SetReadDeadline(time.Now().Add(s.Timeout)); err != nil {
		return Response{}, fmt.Errorf("failed to set deadline: %w", err)
	}
	const maxPacketSize = 1500
	buff := make([]byte, maxPacketSize)
	n, from, err := socket.ReadFrom(buff)
	if err != nil {
		return Response{}, fmt.Errorf("read: %w", err)
	}

	var protocol int
	switch {
	case socket.IPv6PacketConn() != nil:
		protocol = 58
	case socket.IPv4PacketConn() != nil:
		protocol = 1
	default:
		return Response{}, fmt.Errorf("unknown IP version")
	}

	var msgID int
	var respType ResponseType
	var seq SequenceNumber

	resp, err := icmp.ParseMessage(protocol, buff[:n])
	if err != nil {
		return Response{}, fmt.Errorf("parse: %w", err)
	}
	switch body := resp.Body.(type) {
	case *icmp.Echo:
		respType = ResponseEchoReply
		msgID = body.ID
		seq = SequenceNumber(body.Seq)
	case *icmp.TimeExceeded:
		respType = ResponseTimeExceeded
		msgID, seq, err = parseTimeExceeded(body.Data, from.(*net.UDPAddr).IP)
		if err != nil {
			return Response{}, fmt.Errorf("parse time exceeded payload: %w", err)
		}
	case *icmp.RawBody:
		// drop these silently
	default:
		return Response{}, fmt.Errorf("unknown response type: %T", body)
	}

	// if the packet is not for our id, drop it
	if s.checkID && msgID != int(s.id) {
		return Response{}, errIncorrectID{id: msgID}
	}

	// find back the original request
	s.lock.Lock()
	defer s.lock.Unlock()
	req, ok := s.outstandingRequests[seq]
	if !ok {
		return Response{}, fmt.Errorf("no request found for seq %d", seq)
	}

	return Response{
		ResponseType: respType,
		From:         from.(*net.UDPAddr).IP,
		Latency:      time.Since(s.outstandingRequests[seq].TimeSent),
		Request:      req,
	}, nil
}

// timeout removes any outstanding packets that have timed out and queue a timeout response for each of them.
func (s *Socket) timeout() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for seq, req := range s.outstandingRequests {
		if time.Since(req.TimeSent) > s.Timeout {
			s.logger.Debug("timeout expired", "seq", seq)
			s.q.Push(Response{
				ResponseType: ResponseTimeout,
				Request:      req,
			})
			delete(s.outstandingRequests, seq)
		}
	}
}

// setTTL sets the ttl on the socket to the provided value.
func (s *Socket) setTTL(ttl uint8) (err error) {
	if s.v4 != nil {
		err = s.v4.IPv4PacketConn().SetTTL(int(ttl))
	}
	if s.v6 != nil {
		err = errors.Join(err, s.v6.IPv6PacketConn().SetHopLimit(int(ttl)))
	}
	return err
}

// parseTimeExceeded extracts Echo ID and Seq from the inner ICMP packet
// Supports both IPv4 and IPv6 TimeExceeded messages
func parseTimeExceeded(data []byte, src net.IP) (id int, seq SequenceNumber, err error) {
	if src.To4() != nil {
		return parseTimeExceededV4(data)
	}
	return parseTimeExceededV6(data)
}

func parseTimeExceededV4(data []byte) (id int, seq SequenceNumber, err error) {
	if len(data) < ipv4.HeaderLen+8 {
		return 0, 0, errors.New("IPv4 payload too short")
	}
	hlen := int(data[0]&0x0f) * 4
	if len(data) < hlen+8 {
		return 0, 0, errors.New("IPv4 inner payload too short")
	}
	inner := data[hlen : hlen+8]
	id = int(binary.BigEndian.Uint16(inner[4:6]))
	seq = SequenceNumber(binary.BigEndian.Uint16(inner[6:8]))
	return id, seq, nil
}

func parseTimeExceededV6(data []byte) (id int, seq SequenceNumber, err error) {
	if len(data) < ipv6.HeaderLen {
		return 0, 0, errors.New("IPv6 payload too short")
	}
	inner := data[ipv6.HeaderLen:]
	m, err := icmp.ParseMessage(58, inner)
	if err != nil {
		return 0, 0, err
	}
	switch b := m.Body.(type) {
	case *icmp.Echo:
		return b.ID, SequenceNumber(b.Seq), nil
	default:
		if len(inner) >= 8 {
			id = int(binary.BigEndian.Uint16(inner[4:6]))
			seq = SequenceNumber(binary.BigEndian.Uint16(inner[6:8]))
			return id, seq, nil
		}
		return 0, 0, errors.New("inner ICMPv6 not Echo and too short")
	}
}
