package pinger

import (
	"context"
	"github.com/clambin/pinger/pkg/ping"
	"iter"
	"log/slog"
	"maps"
	"slices"
	"time"
)

type TargetPinger struct {
	targets map[string]*ping.Target
	conn    ping.Socket
	logger  *slog.Logger
}

func New(targetList []Target, s ping.Socket, logger *slog.Logger) *TargetPinger {
	mp := TargetPinger{
		targets: make(map[string]*ping.Target, len(targetList)),
		conn:    s,
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
	go ping.Ping(ctx, slices.Collect(maps.Values(tp.targets)), tp.conn, time.Second, 5*time.Second, tp.logger)
	<-ctx.Done()
}

func (tp *TargetPinger) Statistics() iter.Seq2[string, ping.Statistics] {
	return func(yield func(string, ping.Statistics) bool) {
		for name, target := range tp.targets {
			stats := target.Statistics()
			target.ResetStatistics()
			if !yield(name, stats) {
				return
			}
		}
	}
}
