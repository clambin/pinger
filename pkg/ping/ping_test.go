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
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	addr := net.ParseIP("127.0.0.1")
	h := Target{IP: addr}
	s := mocks.NewSocket(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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

	l := slog.Default() // slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	go Ping(ctx, []*Target{&h}, s, time.Second, 5*time.Second, l)

	assert.Eventually(t, func() bool {
		_, received, _ := h.GetStatistics()
		return received > 0
	}, 5*time.Second, 10*time.Millisecond)
	_, _, latency := h.GetStatistics()
	assert.LessOrEqual(t, latency, 2*delay)
}

func Test_outstandingPackets_timeout(t *testing.T) {
	var p outstandingPackets
	p.add(1)
	p.add(2)
	time.Sleep(time.Second)
	p.add(3)
	timedOut := p.timeout(500 * time.Millisecond)
	slices.Sort(timedOut)
	assert.Equal(t, []icmp2.SequenceNumber{1, 2}, timedOut)
	assert.Len(t, p.packets, 1)
	_, ok := p.packets[3]
	assert.True(t, ok)
}
