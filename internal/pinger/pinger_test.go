package pinger

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
	"log/slog"
	"net"
	"testing"
	"time"
)

func TestMultiPinger(t *testing.T) {
	p := NewMultiPinger([]Target{{Host: "127.0.0.1"}, {Host: "::1"}}, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		require.NoError(t, p.Run(ctx))
	}()

	assert.Eventually(t, func() bool {
		stats := p.Statistics()
		ipv4, ok := stats["127.0.0.1"]
		if !ok {
			return false
		}
		ipv6, ok := stats["::1"]
		if !ok {
			return false
		}
		return ipv4.Rcvd > 0 && ipv6.Rcvd > 0
	}, 2*time.Second, 500*time.Millisecond)
}

func Test_targetPinger_Ping(t *testing.T) {
	s := newICMPSocket(slog.Default())
	p := newTargetPinger(
		net.ParseIP("127.0.0.1"),
		s,
		slog.Default(),
	)
	s.OnReply = func(ip net.IP, echo *icmp.Echo) {
		p.responses <- echo
	}
	ctx, cancel := context.WithCancel(context.Background())
	go s.listen(ctx)
	go p.Run(ctx)

	var stats Statistics
	assert.Eventually(t, func() bool {
		stats = p.Statistics()
		return stats.Rcvd > 0

	}, 2*time.Second, 10*time.Millisecond)
	cancel()

	t.Log(stats)
}
