package ping

import (
	"context"
	icmp2 "github.com/clambin/pinger/pkg/ping/icmp"
	"github.com/clambin/pinger/pkg/ping/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	addr := net.ParseIP("127.0.0.1")
	target := Target{IP: addr}
	s := mocks.NewSocket(t)
	ctx := t.Context()
	var lastSeq atomic.Int32
	s.EXPECT().
		Ping(addr, mock.Anything, uint8(0x40), mock.Anything).
		RunAndReturn(func(_ net.IP, seq icmp2.SequenceNumber, _ uint8, _ []byte) error {
			lastSeq.Store(int32(seq))
			return nil
		})
	const delay = 500 * time.Millisecond
	s.EXPECT().Read(ctx).RunAndReturn(func(ctx context.Context) (icmp2.Response, error) {
		time.Sleep(delay)
		r := icmp2.Response{
			From:    addr,
			MsgType: ipv4.ICMPTypeEchoReply,
			Body: &icmp.Echo{
				Seq:  int(lastSeq.Load()),
				Data: []byte{},
			},
			Received: time.Now(),
		}
		return r, nil
	})

	l := slog.New(slog.DiscardHandler) // slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	go Ping(ctx, []*Target{&target}, s, time.Second, 5*time.Second, l)

	assert.Eventually(t, func() bool {
		statistics := target.Statistics()
		return statistics.Received > 0
	}, 5*time.Second, 10*time.Millisecond)
	statistics := target.Statistics()
	assert.LessOrEqual(t, statistics.Latency, 2*delay)
}
