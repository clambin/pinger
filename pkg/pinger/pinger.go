package pinger

import (
	"context"
	"fmt"
	"github.com/clambin/pinger/pkg/pinger/socket"
	"log/slog"
	"maps"
	"net"
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
	Target     Target
	SequenceNr int
	Latency    time.Duration
}

// New creates a Pinger for the specified hostnames
func New(ch chan<- Response, targets []Target) (*Pinger, error) {
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
func MustNew(ch chan<- Response, targets []Target) *Pinger {
	p, err := New(ch, targets)
	if err != nil {
		panic(fmt.Errorf("pinger: %w", err))
	}
	return p
}

// Run sends an icmp echo request to each target every second. All responses are sent back to the Response channel provided at creation.
func (p *Pinger) Run(ctx context.Context, interval time.Duration) {
	responses := make(chan socket.Response)
	go p.socket.Receive(ctx, responses)

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
		case response := <-responses:
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

type targetPinger struct {
	target       Target
	addr         net.Addr
	addrAsString string
	network      string
	socket       *socket.Socket
	seq          int
	packets      map[int]time.Time
	lock         sync.Mutex
}

func newTargetPinger(target Target, s *socket.Socket) (*targetPinger, error) {
	addr, network, err := s.Resolve(target.Host)
	if err != nil {
		return nil, err
	}

	slog.Debug("adding target", "name", target.GetName(), "network", network, "addr", addr.String())

	return &targetPinger{
		target:       target,
		addr:         addr,
		addrAsString: addr.String(),
		network:      network,
		socket:       s,
		packets:      make(map[int]time.Time),
	}, nil
}

const retentionPeriod = time.Minute

func (t *targetPinger) run(ctx context.Context, interval time.Duration) {
	cleanup := time.NewTicker(retentionPeriod)
	defer cleanup.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	slog.Debug("pinger started", "target", t.target.GetName())
	defer slog.Debug("pinger stopped", "target", t.target.GetName())

	for {
		if err := t.ping(); err != nil {
			slog.Error("failed to send icmp echo request", err, "target", t.target.GetName())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-cleanup.C:
			t.cleanup()
		}
	}
}

func (t *targetPinger) ping() (err error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if err = t.socket.Send(t.addr, t.network, t.seq); err == nil {
		t.packets[t.seq] = time.Now()
		t.seq = (t.seq + 1) & 0xffff
	}
	return err
}

func (t *targetPinger) pong(response socket.Response) (sent time.Time, found bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if sent, found = t.packets[response.Seq]; found {
		delete(t.packets, response.Seq)
	}
	return sent, found
}

func (t *targetPinger) cleanup() {
	t.lock.Lock()
	defer t.lock.Unlock()
	maps.DeleteFunc(t.packets, func(i int, t time.Time) bool {
		return time.Since(t) > retentionPeriod
	})
}
