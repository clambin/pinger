package pinger

import (
	"context"
	"github.com/clambin/pinger/pkg/ping"
	icmp2 "github.com/clambin/pinger/pkg/ping/icmp"
	"log/slog"
	"time"
)

type TargetPinger struct {
	targets []*ping.Target
	labels  map[string]string
	conn    ping.Socket
	logger  *slog.Logger
}

func New(targetList []Target, tp icmp2.Transport, logger *slog.Logger) *TargetPinger {
	if tp == 0 {
		tp = icmp2.IPv4 | icmp2.IPv6
	}
	mp := TargetPinger{
		targets: make([]*ping.Target, 0, len(targetList)),
		labels:  make(map[string]string, len(targetList)),
		conn:    icmp2.New(tp, logger.With("module", "icmp")),
		logger:  logger,
	}

	for _, target := range targetList {
		ip, err := mp.conn.Resolve(target.Host)
		if err != nil {
			logger.Error("failed to resolve target. skipping", "target", target.Host, "err", err)
			continue
		}
		mp.targets = append(mp.targets, &ping.Target{IP: ip})

		name := target.Name
		if name == "" {
			name = target.Host
		}
		mp.labels[ip.String()] = name
	}

	return &mp
}

func (tp *TargetPinger) Run(ctx context.Context) {
	go ping.Ping(ctx, tp.targets, tp.conn, time.Second, 5*time.Second, tp.logger)
	<-ctx.Done()
}

type Statistics struct {
	Sent     int
	Received int
	Latency  time.Duration
}

func (tp *TargetPinger) Statistics() map[string]Statistics {
	stats := make(map[string]Statistics, len(tp.targets))
	for _, target := range tp.targets {
		label, ok := tp.labels[target.String()]
		if !ok {
			label = "(unknown)"
		}
		sent, received, latency := target.GetStatistics()

		stats[label] = Statistics{
			Sent:     sent,
			Received: received,
			Latency:  latency,
		}
	}
	return stats
}
