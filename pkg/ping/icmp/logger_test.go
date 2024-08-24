package icmp

import (
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"testing"
)

func TestMessageLogger(t *testing.T) {
	tests := []struct {
		name string
		msg  icmp.Message
		want string
	}{
		{
			name: "ipv4 - request",
			msg:  icmp.Message{Type: ipv4.ICMPTypeEcho, Code: 0, Body: &icmp.Echo{ID: 1, Seq: 1, Data: []byte("hello world")}},
			want: `[type=echo seq=1]`,
		},
		{
			name: "ipv4 - response",
			msg:  icmp.Message{Type: ipv4.ICMPTypeEchoReply, Code: 0, Body: &icmp.Echo{ID: 1, Seq: 1, Data: []byte("hello world")}},
			want: `[type=echo reply seq=1]`,
		},
		{
			name: "ipv4 - time exceeded",
			msg:  icmp.Message{Type: ipv4.ICMPTypeTimeExceeded, Code: 0, Body: &icmp.TimeExceeded{Data: []byte("hello world")}},
			want: `[type=time exceeded]`,
		},
		{
			name: "ipv6 - request",
			msg:  icmp.Message{Type: ipv6.ICMPTypeEchoRequest, Code: 0, Body: &icmp.Echo{ID: 1, Seq: 1, Data: []byte("hello world")}},
			want: `[type=echo request seq=1]`,
		},
		{
			name: "ipv6 - response",
			msg:  icmp.Message{Type: ipv6.ICMPTypeEchoReply, Code: 0, Body: &icmp.Echo{ID: 1, Seq: 1, Data: []byte("hello world")}},
			want: `[type=echo reply seq=1]`,
		},
		{
			name: "ipv6 - time exceeded",
			msg:  icmp.Message{Type: ipv6.ICMPTypeTimeExceeded, Code: 0, Body: &icmp.TimeExceeded{Data: []byte("hello world")}},
			want: `[type=time exceeded]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, messageLogger(tt.msg).LogValue().String())
		})
	}

}
