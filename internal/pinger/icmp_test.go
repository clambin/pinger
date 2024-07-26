package pinger

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func Test_icmpSocket_v4(t *testing.T) {
	var count atomic.Int32
	s := newICMPSocket(slog.Default())
	s.v6 = nil
	s.OnReply = func(ip net.IP, echo *icmp.Echo) {
		// fmt.Println("OnReply echo:", ip, echo.Seq)
		count.Add(1)
	}
	s.Timeout = 5 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		require.NoError(t, s.listen(ctx))
	}()

	ips, err := net.LookupIP("127.0.0.1")
	//ips, err := net.LookupIP("::1")
	require.NoError(t, err)
	assert.Len(t, ips, 1)
	for seq := range 10 {
		assert.NoError(t, s.ping(ips[0], seq, []byte("hello world")))
		time.Sleep(time.Millisecond * 100)
	}
	assert.Eventually(t, func() bool { return count.Load() == 10 }, s.Timeout, time.Millisecond*100)
}

func Test_icmpSocket_v6(t *testing.T) {
	var count atomic.Int32
	s := newICMPSocket(slog.Default())
	if s.v6 == nil {
		t.Skip("IPv6 not available.  Skipping")
	}

	s.OnReply = func(ip net.IP, echo *icmp.Echo) {
		// fmt.Println("OnReply echo:", ip, echo.Seq)
		count.Add(1)
	}
	s.Timeout = 5 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		require.NoError(t, s.listen(ctx))
	}()

	ips, err := net.LookupIP("::1")
	require.NoError(t, err)
	assert.Len(t, ips, 1)
	for seq := range 10 {
		assert.NoError(t, s.ping(ips[0], seq, []byte("hello world")))
		time.Sleep(time.Millisecond * 100)
	}
	assert.Eventually(t, func() bool { return count.Load() == 10 }, s.Timeout, time.Millisecond*100)
}
