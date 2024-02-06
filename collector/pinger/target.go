package pinger

import (
	"context"
	"github.com/clambin/pinger/collector/pinger/socket"
	"github.com/clambin/pinger/configuration"
	"log/slog"
	"maps"
	"net"
	"sync"
	"time"
)

type targetPinger struct {
	target       configuration.Target
	addr         net.Addr
	addrAsString string
	network      string
	socket       *socket.Socket
	seq          int
	packets      map[int]time.Time
	lock         sync.Mutex
}

func newTargetPinger(target configuration.Target, s *socket.Socket) (*targetPinger, error) {
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
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := t.ping(); err != nil {
				slog.Error("failed to send icmp echo request", err, "target", t.target.GetName())
			}
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
