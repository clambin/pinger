package pinger

import (
	"context"
	"github.com/clambin/pinger/collector/pinger/socket"
	"golang.org/x/exp/slog"
	"net"
	"sync"
	"time"
)

type target struct {
	host         string
	addr         net.Addr
	addrAsString string
	network      string
	socket       *socket.Socket
	seq          int
	packets      map[int]time.Time
	lock         sync.Mutex
}

func newTarget(name string, s *socket.Socket) (*target, error) {
	addr, network, err := s.Resolve(name)
	if err != nil {
		return nil, err
	}

	slog.Debug("adding target", "name", name, "network", network, "addr", addr.String())

	return &target{
		host:         name,
		addr:         addr,
		addrAsString: addr.String(),
		network:      network,
		socket:       s,
		packets:      make(map[int]time.Time),
	}, nil
}

const retentionPeriod = time.Minute

func (t *target) run(ctx context.Context, interval time.Duration) {
	cleanup := time.NewTicker(retentionPeriod)
	defer cleanup.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := t.ping(); err != nil {
				slog.Error("failed to send icmp echo request", err, "target", t.host)
			}
		case <-cleanup.C:
			t.cleanup()
		}
	}
}

func (t *target) ping() (err error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if err = t.socket.Send(t.addr, t.network, t.seq); err == nil {
		t.packets[t.seq] = time.Now()
		t.seq = (t.seq + 1) & 0xffff
	}
	return err
}

func (t *target) pong(response socket.Response) (sent time.Time, found bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if sent, found = t.packets[response.Seq]; found {
		delete(t.packets, response.Seq)
	}
	return sent, found
}

func (t *target) cleanup() {
	t.lock.Lock()
	defer t.lock.Unlock()
	for seq, timestamp := range t.packets {
		if time.Since(timestamp) > retentionPeriod {
			delete(t.packets, seq)
		}
	}
}
