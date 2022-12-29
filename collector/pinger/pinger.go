package pinger

import (
	"context"
	"fmt"
	"github.com/clambin/pinger/collector/pinger/socket"
	"sync"
	"time"
)

// Pinger pings a list of targets and sends the response packets on a Response channel
type Pinger struct {
	socket  *socket.Socket
	ch      chan<- Response
	targets []*target
}

type Response struct {
	Host       string
	SequenceNr int
	Latency    time.Duration
}

// New creates a Pinger for the specified hostnames
func New(ch chan<- Response, hostnames ...string) (*Pinger, error) {
	s, err := socket.New()
	if err != nil {
		return nil, err
	}

	var targets []*target
	for _, hostname := range hostnames {
		var t *target
		if t, err = newTarget(hostname, s); err != nil {
			return nil, fmt.Errorf("%s: %w", hostname, err)
		}
		targets = append(targets, t)
	}

	return &Pinger{socket: s, ch: ch, targets: targets}, nil
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
			t.run(ctx, interval)
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
	responseAddr := response.Addr.String()
	for _, t := range p.targets {
		if t.addrAsString == responseAddr {
			if timestamp, sent := t.pong(response); sent {
				p.ch <- Response{
					Host:       t.host,
					SequenceNr: response.Seq,
					Latency:    time.Since(timestamp),
				}
			}
		}
	}
}
