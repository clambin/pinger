package pinger

import (
	"context"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestMultiPinger(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	p := NewMultiPinger([]Target{{Host: "127.0.0.1"}, {Host: "::1"}}, IPv4|IPv6, l)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		errCh <- p.Run(ctx)
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
	}, 5*time.Second, 500*time.Millisecond)

	l.Debug("shutting down")
	cancel()
	assert.NoError(t, <-errCh)
}
