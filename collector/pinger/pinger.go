package pinger

import (
	"context"
	"fmt"
	"github.com/clambin/pinger/collector/pinger/socket"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Pinger pings a list of targets and sends the response packets on a Response channel
type Pinger struct {
	s       *socket.Socket
	ch      chan<- Response
	targets map[string]*target
}

type Response struct {
	Host       string
	SequenceNr int
	Latency    time.Duration
}

// New creates a Pinger for the specified hostnames
func New(ch chan<- Response, hostnames ...string) (p *Pinger, err error) {
	p = &Pinger{
		ch:      ch,
		targets: make(map[string]*target),
	}
	if p.s, err = socket.New(); err != nil {
		return nil, err
	}

	for _, hostname := range hostnames {
		t, err := newTarget(hostname, p.s)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", hostname, err)
		}
		p.targets[t.addr.String()] = t
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

func (p *Pinger) Run(ctx context.Context, interval time.Duration) {
	ch := make(chan socket.Response)
	go p.s.Receive(ctx, ch)

	var wg sync.WaitGroup
	wg.Add(len(p.targets))
	for _, t := range p.targets {
		go func(t *target) {
			defer wg.Done()
			t.Run(ctx, interval)
		}(t)
	}
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		case response := <-ch:
			p.processResponse(response)
		}
	}
}

func (p *Pinger) processResponse(response socket.Response) {
	t, found := p.targets[response.Addr.String()]
	if !found {
		log.Errorf("received packet for unknown target: %s", response.Addr.String())
		return
	}
	if sent, found := t.Pong(response); found {
		p.ch <- Response{
			Host:       t.host,
			SequenceNr: response.Seq,
			Latency:    time.Since(sent),
		}
	}
}
