package pinger

import (
	"context"
	"github.com/clambin/pinger/pkg/pinger/socket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNew_IPv4(t *testing.T) {
	ch := make(chan Response)
	c := MustNew(ch, []Target{{Host: "127.0.0.1"}})

	ctx, cancel := context.WithCancel(context.Background())
	go c.Run(ctx, 10*time.Millisecond)

	p := <-ch
	assert.Equal(t, "127.0.0.1", p.Target.Host)
	assert.Equal(t, 0, p.SequenceNr)
	p = <-ch
	assert.Equal(t, "127.0.0.1", p.Target.Host)
	assert.Equal(t, 1, p.SequenceNr)

	cancel()
}

func TestNew_IPv6(t *testing.T) {
	ch := make(chan Response)
	c := MustNew(ch, []Target{{Host: "::1"}})
	if !c.socket.HasIPv6() {
		t.Skip("build system does not have IPv6 enabled. skipping")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go c.Run(ctx, 10*time.Millisecond)

	p := <-ch
	assert.Equal(t, "::1", p.Target.Host)
	assert.Equal(t, 0, p.SequenceNr)
	p = <-ch
	assert.Equal(t, "::1", p.Target.Host)
	assert.Equal(t, 1, p.SequenceNr)

	cancel()
}

func TestMustNew_Panic(t *testing.T) {
	assert.Panics(t, func() {
		ch := make(chan Response)
		_ = MustNew(ch, []Target{{Host: "127.0.0.256"}})
	})
}

func TestWrap(t *testing.T) {
	ch := make(chan Response)
	c := MustNew(ch, []Target{{Host: "127.0.0.1"}})
	c.targets[0].seq = 0xfffe

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, time.Millisecond)

	p := <-ch
	assert.Equal(t, 0xfffe, p.SequenceNr)
	p = <-ch
	assert.Equal(t, 0xffff, p.SequenceNr)
	p = <-ch
	assert.Equal(t, 0x0000, p.SequenceNr)
	p = <-ch
	assert.Equal(t, 0x0001, p.SequenceNr)
}

func TestPinger_Multiple(t *testing.T) {
	ch := make(chan Response)
	c := MustNew(ch, []Target{{Host: "localhost"}, {Host: "::1"}})
	if !c.socket.HasIPv6() {
		t.Skip("build system does not have IPv6 enabled. skipping")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, 10*time.Millisecond)

	counts := make(map[string]int)
	for range 6 {
		p := <-ch
		current := counts[p.Target.Host]
		counts[p.Target.Host] = current + 1
	}
	assert.Equal(t, 3, counts["localhost"])
	assert.Equal(t, 3, counts["::1"])
}

func TestTargetNew(t *testing.T) {
	s, err := socket.New()
	require.NoError(t, err)

	endpoint, err := newTargetPinger(Target{Host: "127.0.0.1"}, s)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:0", endpoint.addr.String())
	assert.Equal(t, "udp", endpoint.addr.Network())

	if !s.HasIPv6() {
		t.Skip("build system does not have IPv6 enabled. skipping")
	}

	endpoint, err = newTargetPinger(Target{Host: "::1"}, s)
	require.NoError(t, err)
	assert.Equal(t, "[::1]:0", endpoint.addr.String())
	assert.Equal(t, "udp", endpoint.addr.Network())
}

func TestTargetSend_V4(t *testing.T) {
	s, err := socket.New()
	require.NoError(t, err)

	endpoint, err := newTargetPinger(Target{Host: "127.0.0.1"}, s)
	require.NoError(t, err)

	for i := range 10 {
		err = endpoint.ping()
		require.NoError(t, err)

		time.Sleep(time.Millisecond)
		timestamp, found := endpoint.pong(socket.Response{
			Addr: endpoint.addr,
			Seq:  i,
		})
		assert.True(t, found)
		assert.NotZero(t, timestamp)
	}
}

func TestTargetSend_V6(t *testing.T) {
	s, err := socket.New()
	require.NoError(t, err)
	if !s.HasIPv6() {
		t.Skip("build system does not have IPv6 enabled. skipping")
	}

	endpoint, err := newTargetPinger(Target{Host: "::1"}, s)
	require.NoError(t, err)

	for i := range 10 {
		err = endpoint.ping()
		require.NoError(t, err)

		time.Sleep(time.Millisecond)
		timestamp, found := endpoint.pong(socket.Response{
			Addr: endpoint.addr,
			Seq:  i,
		})
		assert.True(t, found)
		assert.NotZero(t, timestamp)
	}
}

func TestTarget_cleanup(t *testing.T) {
	s, _ := socket.New()
	endpoint, _ := newTargetPinger(Target{Host: "::1"}, s)

	timestamp := time.Now()
	endpoint.packets[1] = timestamp.Add(-time.Minute)
	endpoint.packets[2] = timestamp.Add(-time.Second)

	endpoint.cleanup()

	assert.Len(t, endpoint.packets, 1)
	assert.NotZero(t, endpoint.packets[2])
}
