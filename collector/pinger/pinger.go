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
	socket  *socket.Socket
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
	if p.socket, err = socket.New(); err != nil {
		return nil, err
	}

	for _, hostname := range hostnames {
		t, err := newTarget(hostname, p.socket)
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

// Run sends an icmp echo request to each target every second. All responses are sent back to the Response channel provided at creation.
func (p *Pinger) Run(ctx context.Context, interval time.Duration) {
	ch := make(chan socket.Response)
	go p.socket.Receive(ctx, ch)

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
			if r, ok := p.getResponse(response); ok {
				p.ch <- r
			}
		}
	}
}

func (p *Pinger) getResponse(response socket.Response) (r Response, ok bool) {
	t, found := p.targets[response.Addr.String()]
	if !found {
		log.Errorf("received packet for unknown target: %s", response.Addr.String())
		return
	}
	if timestamp, sent := t.Pong(response); sent {
		r = Response{
			Host:       t.host,
			SequenceNr: response.Seq,
			Latency:    time.Since(timestamp),
		}
		ok = true
	}
	return
}
