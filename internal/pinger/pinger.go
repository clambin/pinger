package pinger

import (
	"context"
	"golang.org/x/net/icmp"
	"log/slog"
	"net"
	"slices"
	"sync"
	"time"
)

type pinger struct {
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

func newPinger(ip net.IP, conn *icmpSocket, logger *slog.Logger) *pinger {
	return &pinger{
		Interval:  time.Second,
		Timeout:   30 * time.Second,
		IP:        ip,
		conn:      conn,
		logger:    logger,
		timings:   make(timings),
		responses: make(chan *icmp.Echo),
	}
}

func (p *pinger) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()
	p.logger.Debug("pinger started")
	defer p.logger.Debug("pinger stopped")
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

func (p *pinger) ping(seq int) {
	if err := p.conn.ping(p.IP, seq, []byte("payload")); err != nil {
		p.logger.Warn("failed to send ping", "err", err)
		return
	}
	p.logger.Debug("ping succeeded", "seq", seq)
	p.lock.Lock()
	defer p.lock.Unlock()
	p.timings.cleanup(p.Timeout)
	p.timings[seq] = time.Now()
	p.stats.Sent++
}

func (p *pinger) pong(response *icmp.Echo) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if sent, ok := p.timings[response.Seq]; ok {
		p.logger.Debug("response received", "seq", response.Seq)
		p.stats.Rcvd++
		p.stats.Latencies = append(p.stats.Latencies, time.Since(sent))
		delete(p.timings, response.Seq)
	}
}

func (p *pinger) Statistics() Statistics {
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
	if loss < 0 {
		loss = 0
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
