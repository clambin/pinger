package pinger

import (
	"github.com/clambin/pinger/collector/pinger/socket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTargetNew(t *testing.T) {
	s, err := socket.New()
	require.NoError(t, err)

	endpoint, err := newTarget("127.0.0.1", s)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:0", endpoint.addr.String())
	assert.Equal(t, "udp", endpoint.addr.Network())

	if !s.HasIPv6() {
		t.Skip("build system does not have IPv6 enabled. skipping")
	}

	endpoint, err = newTarget("::1", s)
	require.NoError(t, err)
	assert.Equal(t, "[::1]:0", endpoint.addr.String())
	assert.Equal(t, "udp", endpoint.addr.Network())
}

func TestTargetSend_V4(t *testing.T) {
	s, err := socket.New()
	require.NoError(t, err)

	endpoint, err := newTarget("127.0.0.1", s)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		err = endpoint.Ping()
		require.NoError(t, err)

		time.Sleep(time.Millisecond)
		timestamp, found := endpoint.Pong(socket.Response{
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

	endpoint, err := newTarget("::1", s)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		err = endpoint.Ping()
		require.NoError(t, err)

		time.Sleep(time.Millisecond)
		timestamp, found := endpoint.Pong(socket.Response{
			Addr: endpoint.addr,
			Seq:  i,
		})
		assert.True(t, found)
		assert.NotZero(t, timestamp)
	}
}
