package pinger

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/nettest"
	"net"
	"os"
	"sync"
	"time"
)

// ICMPPingers spawns a collector process and reports to a specified Tracker
func ICMPPingers(ch chan Response, hosts ...string) error {
	p, err := New(ch, hosts...)
	if err == nil {
		p.Run(context.Background(), time.Second)
	}
	return err
}

type Pinger struct {
	conn    *icmpConnection
	ch      chan<- Response
	targets map[string]*target
}

type PingResponse struct {
	Host       string
	SequenceNr int
	Latency    time.Duration
}

type target struct {
	host    string
	addr    net.Addr
	seqno   int
	packets map[int]time.Time
	lock    sync.Mutex
}

func New(ch chan<- Response, targets ...string) (p *Pinger, err error) {
	p = &Pinger{
		ch:      ch,
		targets: make(map[string]*target),
	}
	if p.conn, err = newConnection(); err != nil {
		return nil, err
	}

	for _, t := range targets {
		if err = p.addTarget(t); err != nil {
			return nil, fmt.Errorf("%s: %w", t, err)
		}
	}

	return p, nil
}

func (p *Pinger) addTarget(name string) error {
	addr, err := p.conn.resolve(name)
	if err == nil {
		p.targets[addr.String()] = &target{
			host:    name,
			addr:    addr,
			packets: make(map[int]time.Time),
		}
	}
	return err
}

func (p *Pinger) Run(ctx context.Context, interval time.Duration) {
	ch := make(chan packet)
	go func() {
		if err := p.conn.listen(ch); err != nil {
			log.WithError(err).Fatal("could not create icmp socket")
		}
	}()

	var wg sync.WaitGroup
	for _, t := range p.targets {
		wg.Add(1)
		go func(t *target) {
			defer wg.Done()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					_ = p.ping(t)
				}
			}
		}(t)
	}

	defer wg.Wait()
	for {
		select {
		case <-ctx.Done():
			return
		case response := <-ch:
			p.pong(response)
		}
	}
}

func (p *Pinger) ping(t *target) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if err := p.conn.send(t.addr, t.seqno); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	t.packets[t.seqno] = time.Now()
	t.seqno++
	return nil
}

func (p *Pinger) pong(response packet) {
	t := p.targets[response.peer.String()]
	t.lock.Lock()
	defer t.lock.Unlock()

	if sent, found := t.packets[response.seqno]; found {
		p.ch <- Response{
			Host:       t.host,
			SequenceNr: response.seqno,
			Latency:    time.Since(sent),
		}
		delete(t.packets, response.seqno)
	}
}

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
	log.Debugf("icmpConnection id: %d", c.id)

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
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   c.id,
			Seq:  seqno,
			Data: []byte("hello"),
		},
	}

	wb, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	if _, err = c.conn.WriteTo(wb, target); err != nil {
		return fmt.Errorf("%s: %w", target, err)
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
		if reply.ID != c.id {
			// FIXME: when running in a github action, received ID is always 1???
			if reply.ID != 1 {
				log.Infof("dropping unexpected packet. id=%d, seq=%d, data=%s", reply.ID, reply.Seq, string(reply.Data))
				continue
			}
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
