package pinger

import (
	"context"
	"golang.org/x/net/icmp"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"slices"
	"sync"
	"time"
)

type MultiPinger struct {
	conn   *icmpSocket
	target map[string]*targetPinger
	logger *slog.Logger
}

func NewMultiPinger(targets []Target, logger *slog.Logger) *MultiPinger {
	mp := MultiPinger{
		conn:   newICMPSocket(logger.With("module", "icmp")),
		target: make(map[string]*targetPinger, len(targets)),
		logger: logger,
	}
	mp.conn.OnReply = mp.OnReply

	for _, target := range targets {
		ip, err := mp.conn.resolve(target.Host)
		if err != nil {
			logger.Error("failed to resolve target. skipping", "target", target.Host, "err", err)
			continue
		}

		name := target.Name
		if name == "" {
			name = target.Host
		}
		mp.target[name] = newTargetPinger(ip, mp.conn, logger.With("target", name))
	}

	return &mp
}

func (mp *MultiPinger) OnReply(ip net.IP, echo *icmp.Echo) {
	for _, pinger := range mp.target {
		if pinger.IP.String() == ip.String() {
			pinger.responses <- echo
			return
		}
	}
	mp.logger.Warn("failed to resolve target. skipping", "target", ip)
}

func (mp *MultiPinger) Run(ctx context.Context) error {
	var g errgroup.Group
	g.Go(func() error { return mp.conn.listen(ctx) })
	for _, pinger := range mp.target {
		g.Go(func() error { return pinger.Run(ctx) })
	}
	return g.Wait()
}

func (mp *MultiPinger) Statistics() map[string]Statistics {
	stats := make(map[string]Statistics, len(mp.target))
	for label, pinger := range mp.target {
		stats[label] = pinger.Statistics()
	}
	return stats
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type targetPinger struct {
	Interval  time.Duration
	Timeout   time.Duration
	IP        net.IP
	conn      *icmpSocket
	logger    *slog.Logger
	stats     Statistics
	timings   timings
	responses chan *icmp.Echo
	lock      sync.Mutex
}

func newTargetPinger(ip net.IP, conn *icmpSocket, logger *slog.Logger) *targetPinger {
	return &targetPinger{
		Interval:  time.Second,
		Timeout:   30 * time.Second,
		IP:        ip,
		conn:      conn,
		logger:    logger,
		timings:   make(timings),
		responses: make(chan *icmp.Echo),
	}
}

func (p *targetPinger) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()
	var seq int
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			p.ping(seq)
			seq++
		case response := <-p.responses:
			p.pong(response)
		}
	}
}

func (p *targetPinger) ping(seq int) {
	if err := p.conn.ping(p.IP, seq, []byte("payload")); err != nil {
		p.logger.Warn("failed to send ping", "err", err)
		return
	}
	p.lock.Lock()
	p.lock.Unlock()
	p.timings[seq] = time.Now()
	p.stats.Sent++
	p.timings.cleanup(30 * time.Second)
}

func (p *targetPinger) pong(response *icmp.Echo) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if sent, ok := p.timings[response.Seq]; ok {
		p.stats.Rcvd++
		p.stats.Latencies = append(p.stats.Latencies, time.Since(sent))
		delete(p.timings, response.Seq)
	}
}

func (p *targetPinger) Statistics() Statistics {
	p.lock.Lock()
	defer p.lock.Unlock()
	stats := p.stats.Clone()
	p.stats.Reset()
	return stats
}

type timings map[int]time.Time

func (t timings) cleanup(timeout time.Duration) {
	for k, v := range t {
		if time.Since(v) > timeout {
			delete(t, k)
		}
	}
}

type Statistics struct {
	Sent      int
	Rcvd      int
	Latencies []time.Duration
}

func (s *Statistics) Latency() time.Duration {
	var total time.Duration
	for _, l := range s.Latencies {
		total += l
	}
	if len(s.Latencies) > 0 {
		total /= time.Duration(len(s.Latencies))
	}
	return total
}

func (s *Statistics) Loss() float64 {
	var loss float64
	if s.Sent > 0 {
		loss = 1 - float64(s.Rcvd)/float64(s.Sent)
	}
	return loss
}

func (s *Statistics) Clone() Statistics {
	return Statistics{
		Sent:      s.Sent,
		Rcvd:      s.Rcvd,
		Latencies: slices.Clone(s.Latencies),
	}
}

func (s *Statistics) Reset() {
	s.Sent = 0
	s.Rcvd = 0
	s.Latencies = s.Latencies[:0]
}
