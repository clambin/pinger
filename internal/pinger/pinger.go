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
	conn      icmpConn
	logger    *slog.Logger
	timings   timings
	responses chan *icmp.Echo
	IP        net.IP
	payload   []byte
	stats     Statistics
	Interval  time.Duration
	Timeout   time.Duration
	lock      sync.Mutex
}

const payloadSize = 64
const timeout = 30 * time.Second

func newPinger(ip net.IP, conn icmpConn, logger *slog.Logger) *pinger {
	return &pinger{
		Interval:  time.Second,
		Timeout:   timeout,
		IP:        ip,
		conn:      conn,
		logger:    logger,
		timings:   make(timings),
		responses: make(chan *icmp.Echo),
		payload:   make([]byte, payloadSize),
	}
}

type icmpConn interface {
	ping(net.IP, int, []byte) error
}

func (p *pinger) run(ctx context.Context) error {
	ticker := time.NewTicker(p.Interval)
	p.logger.Debug("pinger started")
	var seq uint16
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			p.logger.Debug("pinger stopped")
			return nil
		case <-ticker.C:
			p.ping(seq)
			seq++
		case resp := <-p.responses:
			p.pong(resp)
		}
	}
}

func (p *pinger) ping(seq uint16) {
	if err := p.conn.ping(p.IP, int(seq), p.payload); err != nil {
		p.logger.Warn("failed to send ping", "err", err)
		return
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	p.stats.Sent += p.timings.cleanup(p.Timeout)
	p.timings[seq] = time.Now()
}

func (p *pinger) pong(response *icmp.Echo) {
	p.lock.Lock()
	defer p.lock.Unlock()
	seq := uint16(response.Seq)
	if sent, ok := p.timings[seq]; ok {
		latency := time.Since(sent)
		p.logger.Debug("pong", "id", response.ID, "seq", response.Seq, "latency", latency)
		p.stats.Sent++
		p.stats.Rcvd++
		p.stats.Latencies = append(p.stats.Latencies, latency)
		delete(p.timings, seq)
	}
}

func (p *pinger) Statistics() Statistics {
	p.lock.Lock()
	defer p.lock.Unlock()
	stats := p.stats.Clone()
	p.stats.Reset()
	return stats
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type timings map[uint16]time.Time

func (t timings) cleanup(timeout time.Duration) int {
	var timedOut int
	for k, v := range t {
		if time.Since(v) > timeout {
			delete(t, k)
			timedOut++
		}
	}
	return timedOut
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type Statistics struct {
	Latencies []time.Duration
	Sent      int
	Rcvd      int
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
	if s.Sent > 0 && s.Sent >= s.Rcvd {
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
	s.Latencies = s.Latencies[:0]
}
