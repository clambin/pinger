package pinger

import (
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/nettest"
	"net"
	"os"
)

type icmpConnection struct {
	conn *icmp.PacketConn
	id   int
}

type packet struct {
	peer  net.Addr
	seqno int
}

func newConnection() (*icmpConnection, error) {
	c := icmpConnection{id: os.Getpid() & 0xffff}

	var err error
	if nettest.SupportsRawSocket() {
		log.Info("raw sockets supported")
		c.conn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	} else {
		c.conn, err = icmp.ListenPacket("udp4", "0.0.0.0")
	}
	return &c, err
}

func (c *icmpConnection) send(target net.Addr, seqno int) error {
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   c.id,
			Seq:  seqno,
			Data: []byte("hello"),
		},
	}

	wb, err := msg.Marshal(nil)
	if err == nil {
		_, err = c.conn.WriteTo(wb, target)
	}
	return err
}

func (c *icmpConnection) listen(ch chan<- packet) error {
	for {
		rb := make([]byte, 1500)
		n, peer, err := c.conn.ReadFrom(rb)
		if err != nil {
			return err
		}

		rm, err := icmp.ParseMessage(1, rb[:n])
		if err != nil {
			return err
		}

		reply := rm.Body.(*icmp.Echo)
		// FIXME: when running in a k8s container, received ID is not pid&0xffff???
		// use reply data instead
		//if reply.ID != c.id {
		if string(reply.Data) != "hello" {
			log.Debugf("dropping unexpected packet: %v", reply)
			continue
		}

		ch <- packet{peer: peer, seqno: reply.Seq}
	}
}

func (c *icmpConnection) resolve(name string) (net.Addr, error) {
	if nettest.SupportsRawSocket() {
		return net.ResolveIPAddr("ip4", name)
	}
	name += ":0"
	return net.ResolveUDPAddr("udp4", name)
}
