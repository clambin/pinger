package socket

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"net"
	"os"
	"strings"
)

type Socket struct {
	conn map[string]*icmp.PacketConn
	id   int
}

type Response struct {
	Addr net.Addr
	Seq  int
}

func New() (*Socket, error) {
	s := Socket{
		conn: make(map[string]*icmp.PacketConn),
		id:   os.Getpid() & 0xffff,
	}
	if conn, err := icmp.ListenPacket("udp4", "0.0.0.0"); err == nil {
		s.conn["udp4"] = conn
	} else {
		log.Warning("No IPv4 found")
	}
	if conn, err := icmp.ListenPacket("udp6", "::"); err == nil {
		s.conn["udp6"] = conn
	} else {
		log.Warning("No IPv6 found")
	}

	// TODO: privileged sockets

	return &s, nil
}

func (s *Socket) Resolve(name string) (net.Addr, string, error) {
	ips, err := net.LookupIP(name)
	if err != nil {
		return nil, "", fmt.Errorf("ip lookup: %w", err)
	}

	for _, ip := range ips {
		isV6 := strings.Count(ip.String(), ":") >= 2

		var network string
		if isV6 && s.HasIPv6() {
			network = "udp6"
		}
		if !isV6 && s.HasIPv4() {
			network = "udp4"
		}

		if network != "" {
			return &net.UDPAddr{IP: ip}, network, nil
		}
	}
	return nil, "", fmt.Errorf("no supported IP address found")
}

func (s *Socket) HasIPv4() bool {
	_, ok := s.conn["udp4"]
	return ok
}

func (s *Socket) HasIPv6() bool {
	_, ok := s.conn["udp6"]
	return ok
}

func (s *Socket) Send(addr net.Addr, network string, seq int) error {
	c, ok := s.conn[network]
	if !ok {
		return fmt.Errorf("invalid network: %s", network)
	}

	var msgType icmp.Type
	switch network {
	case "udp4":
		msgType = ipv4.ICMPTypeEcho
	case "udp6":
		msgType = ipv6.ICMPTypeEchoRequest
	}

	msg := icmp.Message{
		Type: msgType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   1,
			Seq:  seq,
			Data: []byte("hello"),
		},
	}

	wb, _ := msg.Marshal(nil)
	_, err := c.WriteTo(wb, addr)
	return err
}

func (s *Socket) Receive(ctx context.Context, ch chan<- Response) {
	for network, conn := range s.conn {
		go func(network string, conn *icmp.PacketConn) {
			err := s.receiveFromConn(ctx, network, conn, ch)
			if err != nil {
				panic(fmt.Errorf("%s: %w", network, err))
			}
		}(network, conn)
	}

	<-ctx.Done()
}

var ianaProtocols = map[string]int{
	"udp4": 1,
	"udp6": 58,
}

func (s *Socket) receiveFromConn(_ context.Context, network string, conn *icmp.PacketConn, ch chan<- Response) error {
	for {
		// TODO: exit when context.Done()
		// needs a timeout on read: conn.SetReadDeadline(time.Now().Add(time.Second))
		// use errors.Is(err, os.ErrDeadlineExceeded) to see if there was a timeout (ie no data)

		rb := make([]byte, 1500)
		n, peer, err := conn.ReadFrom(rb)
		if err != nil {
			return err
		}

		rm, err := icmp.ParseMessage(ianaProtocols[network], rb[:n])
		if err != nil {
			return fmt.Errorf("recv parse: %w", err)
		}

		if rm.Type != ipv4.ICMPTypeEchoReply && rm.Type != ipv6.ICMPTypeEchoReply {
			continue
		}

		reply := rm.Body.(*icmp.Echo)

		// FIXME: when running in a k8s container, received ID is not pid&0xffff???
		// use reply data instead
		//if reply.ID != c.id {
		if string(reply.Data) != "hello" {
			log.Debugf("dropping unexpected packet: %v", reply)
			continue
		}

		ch <- Response{Addr: peer, Seq: reply.Seq}
	}
}
