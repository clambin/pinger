package pinger

import (
	"context"
	"github.com/clambin/pinger/pkg/ping"
	"github.com/clambin/pinger/pkg/ping/icmp"
	"golang.org/x/exp/maps"
	"log/slog"
	"time"
)

type TargetPinger struct {
	targets map[string]*ping.Target
	conn    ping.Socket
	logger  *slog.Logger
}

func New(targetList []Target, tp icmp.Transport, logger *slog.Logger) *TargetPinger {
	if tp == 0 {
		tp = icmp.IPv4 | icmp.IPv6
	}
	mp := TargetPinger{
		targets: make(map[string]*ping.Target, len(targetList)),
		conn:    icmp.New(tp, logger.With("module", "icmp")),
		logger:  logger,
	}

	for _, target := range targetList {
		ip, err := mp.conn.Resolve(target.Host)
		if err != nil {
			logger.Error("failed to resolve target. skipping", "target", target.Host, "err", err)
			continue
		}

		name := target.Name
		if name == "" {
			name = target.Host
		}
		mp.targets[name] = &ping.Target{IP: ip}
	}

	return &mp
}

func (tp *TargetPinger) Run(ctx context.Context) {
	go tp.conn.Serve(ctx)
	go ping.Ping(ctx, maps.Values(tp.targets), tp.conn, time.Second, 5*time.Second, tp.logger)
	<-ctx.Done()
}

func (tp *TargetPinger) Statistics() map[string]ping.Statistics {
	stats := make(map[string]ping.Statistics, len(tp.targets))
	for name, target := range tp.targets {
		stats[name] = target.Statistics()
		target.ResetStatistics()
	}
	return stats
}
