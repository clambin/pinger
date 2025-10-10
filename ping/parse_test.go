package ping

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func TestParseTimeExceeded(t *testing.T) {
	type want struct {
		err require.ErrorAssertionFunc
		id  int
		seq SequenceNumber
	}
	tests := []struct {
		name  string
		build func() ([]byte, net.IP)
		want  want
	}{
		{
			name: "ipv4 success",
			build: func() ([]byte, net.IP) {
				// Build ICMPv4 Echo request
				echo := &icmp.Echo{ID: 1, Seq: 2}
				msg := &icmp.Message{Type: ipv4.ICMPTypeEcho, Body: echo}
				raw, _ := msg.Marshal(nil)
				// Fake IPv4 header (20 bytes, IHL=5)
				ipHeader := make([]byte, ipv4.HeaderLen)
				ipHeader[0] = (4 << 4) | 5 // Version 4, IHL=5 (20 bytes)
				return append(ipHeader, raw...), net.IPv4(127, 0, 0, 1)
			},
			want: want{require.NoError, 1, 2},
		},
		{
			name: "ipv4 too short",
			build: func() ([]byte, net.IP) {
				return make([]byte, ipv4.HeaderLen+7), net.IPv4(127, 0, 0, 1)
			},
			want: want{require.Error, 0, 0},
		},
		{
			name: "ipv6 success",
			build: func() ([]byte, net.IP) {
				// Build ICMPv6 Echo request
				echo := &icmp.Echo{ID: 1, Seq: 2}
				msg := &icmp.Message{Type: ipv6.ICMPTypeEchoRequest, Body: echo}
				raw, _ := msg.Marshal(nil)
				// Prepend IPv6 header
				return append(make([]byte, ipv6.HeaderLen), raw...), net.IPv6loopback
			},
			want: want{require.NoError, 1, 2},
		},
		{
			name: "ipv6 fallback to raw bytes",
			build: func() ([]byte, net.IP) {
				// inner payload that isn't a valid ICMP message, but is long enough
				inner := make([]byte, 8)
				binary.BigEndian.PutUint16(inner[4:], uint16(1))
				binary.BigEndian.PutUint16(inner[6:], uint16(2))
				// Prepend IPv6 header
				return append(make([]byte, ipv6.HeaderLen), inner...), net.IPv6loopback
			},
			want: want{require.NoError, 1, 2},
		},
		{
			name: "ipv6 too short",
			build: func() ([]byte, net.IP) {
				return make([]byte, ipv6.HeaderLen+7), net.IPv6loopback
			},
			want: want{require.Error, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, src := tt.build()
			gotID, gotSeq, err := parseTimeExceeded(data, src)
			tt.want.err(t, err)
			assert.Equal(t, tt.want.id, gotID)
			assert.Equal(t, tt.want.seq, gotSeq)
		})
	}
}
