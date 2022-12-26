package pinger

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

// Pinger pings a list of targets and sends the response packets on a Response channel
type Pinger struct {
	conn    *icmpConnection
	ch      chan<- Response
	targets map[string]*target
}

type Response struct {
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

// New creates a Pinger for the specified targets
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

// MustNew creates a Pinger for the specified targets. Panics if a Pinger could not be created
func MustNew(ch chan<- Response, targets ...string) *Pinger {
	p, err := New(ch, targets...)
	if err != nil {
		panic(fmt.Errorf("pinger: %w", err))
	}
	return p
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
	wg.Add(len(p.targets))
	for _, t := range p.targets {
		go func(t *target) {
			defer wg.Done()
			p.runPing(ctx, t, interval)
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

func (p *Pinger) runPing(ctx context.Context, t *target, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.ping(t); err != nil {
				log.WithError(err).WithField("target", t.host).Error("failed to send icmp echo request")
			}
		}
	}
}

func (p *Pinger) ping(t *target) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	err := p.conn.send(t.addr, t.seqno)
	if err == nil {
		t.packets[t.seqno] = time.Now()
		t.seqno++
	}
	return err
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
