package pinger

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/icmp"
	"log/slog"
	"net"
	"testing"
	"time"
)

func TestStatistics(t *testing.T) {
	tests := []struct {
		name        string
		stats       Statistics
		wantLoss    float64
		wantLatency time.Duration
	}{
		{
			name: "statistics",
			stats: Statistics{
				Sent:      10,
				Rcvd:      5,
				Latencies: []time.Duration{time.Second, 1500 * time.Millisecond, 500 * time.Millisecond},
			},
			wantLoss:    .5,
			wantLatency: time.Second,
		},
		{
			name:        "no statistics",
			stats:       Statistics{},
			wantLoss:    0,
			wantLatency: 0,
		},
		{
			name:        "late arrival",
			stats:       Statistics{Sent: 1, Rcvd: 2},
			wantLoss:    0,
			wantLatency: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantLoss, tt.stats.Loss())
			assert.Equal(t, tt.wantLatency, tt.stats.Latency())
		})
	}
}

func Test_timings_cleanup(t *testing.T) {
	now := time.Now()
	tm := timings{
		1: now.Add(-5 * time.Second),
		2: now.Add(-4 * time.Second),
		3: now,
	}
	assert.Equal(t, 1, tm.cleanup(5*time.Second))
	assert.Len(t, tm, 2)
	_, ok := tm[1]
	assert.False(t, ok)
}

func Test_icmpSeq_next(t *testing.T) {
	var s icmpSeq
	s = s.next()
	assert.Equal(t, 1, int(s))
	s = s.next()
	assert.Equal(t, 2, int(s))
	s = icmpSeq(0xffff)
	s = s.next()
	assert.Equal(t, 0, int(s))
}

func Test_pinger_Run(t *testing.T) {
	var l fakePinger
	p := newPinger(net.ParseIP("::1"), nil, slog.Default())
	p.conn = &l
	p.Interval = 100 * time.Millisecond
	l.pinger = p

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- p.run(ctx) }()

	assert.Eventually(t, func() bool { return p.Statistics().Rcvd >= 2 }, time.Second, p.Interval)

	cancel()
	stats := p.Statistics()
	assert.GreaterOrEqual(t, stats.Sent, 2)
	assert.GreaterOrEqual(t, stats.Rcvd, 2)
}

func Test_pinger_Run_Fail(t *testing.T) {
	var l fakePinger
	p := newPinger(net.ParseIP("::1"), nil, slog.Default())
	p.conn = &l
	p.Interval = 100 * time.Millisecond
	l.pinger = p
	l.err = errors.New("ping failed")

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- p.run(ctx) }()

	time.Sleep(3 * p.Interval)

	cancel()
	stats := p.Statistics()
	assert.Zero(t, stats.Sent)
	assert.Zero(t, stats.Rcvd)
}

var _ icmpConn = &fakePinger{}

type fakePinger struct {
	pinger *pinger
	err    error
}

func (f fakePinger) ping(_ net.IP, seq int, payload []byte) error {
	if f.err != nil {
		return f.err
	}
	go func() {
		f.pinger.responses <- &icmp.Echo{
			ID:   0,
			Seq:  seq,
			Data: payload,
		}
	}()
	return nil
}
