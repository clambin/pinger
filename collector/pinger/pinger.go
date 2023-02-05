package pinger

import (
	"context"
	"fmt"
	"github.com/clambin/pinger/collector/pinger/socket"
	"github.com/clambin/pinger/configuration"
	"sync"
	"time"
)

// Pinger pings a list of targets and sends the response packets on a Response channel
type Pinger struct {
	socket  *socket.Socket
	ch      chan<- Response
	targets []*targetPinger
}

type Response struct {
	Target     configuration.Target
	SequenceNr int
	Latency    time.Duration
}

// New creates a Pinger for the specified hostnames
func New(ch chan<- Response, targets []configuration.Target) (*Pinger, error) {
	s, err := socket.New()
	if err != nil {
		return nil, err
	}

	var targetPingers []*targetPinger
	for _, target := range targets {
		var t *targetPinger
		if t, err = newTargetPinger(target, s); err != nil {
			return nil, fmt.Errorf("%s: %w", target.GetName(), err)
		}
		targetPingers = append(targetPingers, t)
	}

	return &Pinger{socket: s, ch: ch, targets: targetPingers}, nil
}

// MustNew creates a Pinger for the specified targets. Panics if a Pinger could not be created
func MustNew(ch chan<- Response, targets []configuration.Target) *Pinger {
	p, err := New(ch, targets)
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
		go func(t *targetPinger) {
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
					Target:     t.target,
					SequenceNr: response.Seq,
					Latency:    time.Since(timestamp),
				}
			}
		}
	}
}
