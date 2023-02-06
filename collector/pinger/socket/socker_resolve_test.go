package socket

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSocket_Resolve_IPv4(t *testing.T) {
	s, err := New()
	require.NoError(t, err)
	delete(s.conn, "udp6")

	tests := []struct {
		name     string
		hostname string
		pass     bool
		addr     string
	}{
		{name: "hostname", hostname: "localhost", pass: true, addr: "127.0.0.1:0"},
		{name: "ip address", hostname: "127.0.0.1", pass: true, addr: "127.0.0.1:0"},
		{name: "ip v6 address", hostname: "::1", pass: false},
		{name: "invalid hostname", hostname: "not-a-hostname", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, network, err := s.Resolve(tt.hostname)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, "udp4", network)
			assert.Equal(t, tt.addr, addr.String())
		})
	}
}

func TestSocket_Resolve_IPv6(t *testing.T) {
	s, err := New()
	require.NoError(t, err)

	if !s.HasIPv6() {
		t.Skip("build system does not have IPv6 enabled. skipping")
	}

	tests := []struct {
		name     string
		hostname string
		pass     bool
		network  string
		addr     string
	}{
		{name: "hostname", hostname: "localhost", pass: true, network: "udp6", addr: "[::1]:0"},
		{name: "ip v6 address", hostname: "::1", pass: true, network: "udp6", addr: "[::1]:0"},
		{name: "ip v4 address", hostname: "127.0.0.1", pass: true, network: "udp4", addr: "127.0.0.1:0"},
		{name: "invalid hostname", hostname: "not-a-hostname", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, network, err := s.Resolve(tt.hostname)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, tt.network, network)
			assert.Equal(t, tt.addr, addr.String())
		})
	}
}
