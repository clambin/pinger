package socket

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSocket_Resolve_IPv6(t *testing.T) {
	s, err := New()
	require.NoError(t, err)

	if !s.HasIPv6() {
		t.Skip("build system does not have IPv6 enabled. skipping")
	}

	addr, network, err := s.Resolve("localhost")
	require.NoError(t, err)
	assert.Equal(t, "udp6", network)
	assert.Equal(t, "[::1]:0", addr.String())
}

func TestSocket_Resolve_IPv4(t *testing.T) {
	s, err := New()
	require.NoError(t, err)

	delete(s.conn, "udp6")

	addr, network, err := s.Resolve("localhost")
	require.NoError(t, err)
	assert.Equal(t, "udp4", network)
	assert.Equal(t, "127.0.0.1:0", addr.String())
}
