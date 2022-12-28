package pinger

import (
	"context"
	"github.com/clambin/pinger/collector/pinger/socket"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type target struct {
	host    string
	addr    net.Addr
	network string
	s       *socket.Socket
	seqno   int
	packets map[int]time.Time
	lock    sync.Mutex
}

func newTarget(name string, s *socket.Socket) (*target, error) {
	addr, network, err := s.Resolve(name)
	if err != nil {
		return nil, err
	}

	log.Debugf("%s resolves to %s:%s", name, network, addr.String())

	return &target{
		host:    name,
		addr:    addr,
		network: network,
		s:       s,
		packets: make(map[int]time.Time),
	}, nil
}

func (t *target) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := t.Ping(); err != nil {
				log.WithError(err).WithField("target", t.host).Error("failed to send icmp echo request")
			}
		}
	}
}

func (t *target) Ping() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	err := t.s.Send(t.addr, t.network, t.seqno)
	if err == nil {
		t.packets[t.seqno] = time.Now()
		t.seqno = (t.seqno + 1) & 0xffff
	}
	return err
}

func (t *target) Pong(response socket.Response) (time.Time, bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	sent, found := t.packets[response.Seq]
	if !found {
		return time.Time{}, false
	}
	delete(t.packets, response.Seq)
	return sent, true
}
