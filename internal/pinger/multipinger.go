package pinger

import (
	"context"
	"golang.org/x/net/icmp"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
)

type MultiPinger struct {
	targets map[string]*pinger
	conn    *icmpSocket
	logger  *slog.Logger
}

func NewMultiPinger(targets []Target, logger *slog.Logger) *MultiPinger {
	mp := MultiPinger{
		targets: make(map[string]*pinger, len(targets)),
		conn:    newICMPSocket(logger.With("module", "icmp")),
		logger:  logger,
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
		mp.targets[name] = newPinger(ip, mp.conn, logger.With("target", name, "transport", getTransport(ip).String()))
	}

	return &mp
}

func (mp *MultiPinger) OnReply(ip net.IP, echo *icmp.Echo) {
	for _, target := range mp.targets {
		if target.IP.String() == ip.String() {
			target.responses <- echo
			return
		}
	}
	mp.logger.Warn("failed to resolve target. skipping", "target", ip)
}

func (mp *MultiPinger) Run(ctx context.Context) error {
	mp.logger.Debug("multipinger starting")
	defer mp.logger.Debug("multipinger stopped")
	var g errgroup.Group
	g.Go(func() error { return mp.conn.listen(ctx) })
	for _, target := range mp.targets {
		g.Go(func() error { return target.Run(ctx) })
	}
	return g.Wait()
}

func (mp *MultiPinger) Statistics() map[string]Statistics {
	stats := make(map[string]Statistics, len(mp.targets))
	for label, target := range mp.targets {
		stats[label] = target.Statistics()
	}
	return stats
}
