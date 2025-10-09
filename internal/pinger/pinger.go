package pinger

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/clambin/pinger/ping"
)

type Socket interface {
	Send(target net.IP, seq ping.SequenceNumber, ttl uint8, payload []byte) error
	Serve(ctx context.Context)
	Read(ctx context.Context) (ping.Response, error)
	Resolve(name string) (net.IP, error)
}

var _ Socket = &ping.Socket{}

type TargetPinger struct {
	targets map[string]*Target
	socket  Socket
	logger  *slog.Logger
}

func New(targets Targets, s Socket, logger *slog.Logger) *TargetPinger {
	mp := TargetPinger{
		targets: make(map[string]*Target, len(targets)),
		socket:  s,
		logger:  logger,
	}

	for _, target := range targets {
		var err error
		target.addr, err = s.Resolve(target.Host)
		if err != nil {
			logger.Error("failed to resolve target. omitting from target list", "target", target.Host, "err", err)
			continue
		}
		mp.targets[target.addr.String()] = target
	}

	return &mp
}

func (tp *TargetPinger) Run(ctx context.Context) {
	go tp.socket.Serve(ctx)
	for _, target := range tp.targets {
		go tp.pingTarget(ctx, target)
	}
	go tp.readResponses(ctx)
	<-ctx.Done()
}

func (tp *TargetPinger) pingTarget(ctx context.Context, target *Target) {
	logger := tp.logger.With("target", target.Name)
	var seq ping.SequenceNumber
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := tp.socket.Send(target.addr, seq, 64, []byte("payload")); err != nil {
				logger.Error("ping failed", "err", err)
			}
			target.markRequests(seq)
			seq++
		}
	}
}

func (tp *TargetPinger) readResponses(ctx context.Context) {
	for {
		response, err := tp.socket.Read(ctx)
		if errors.Is(err, context.Canceled) || errors.Is(err, ping.ErrTimeout) {
			return
		}
		if err != nil {
			tp.logger.Error("read failed", "err", err)
			continue
		}
		if response.ResponseType != ping.ResponseEchoReply {
			tp.logger.Debug("ignoring non-echo response", "response", response)
			continue
		}
		target, ok := tp.targets[response.From.String()]
		if !ok {
			tp.logger.Debug("no target found for response", "response", response)
			continue
		}
		target.markResponse(response)
	}
}
