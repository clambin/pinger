package icmp

import (
	"bytes"
	"github.com/clambin/go-common/testutils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"log/slog"
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
			want: `level=INFO msg=msg msg.type=echo msg.seq=1
`,
		},
		{
			name: "ipv4 - response",
			msg:  icmp.Message{Type: ipv4.ICMPTypeEchoReply, Code: 0, Body: &icmp.Echo{ID: 1, Seq: 1, Data: []byte("hello world")}},
			want: `level=INFO msg=msg msg.type="echo reply" msg.seq=1
`,
		},
		{
			name: "ipv4 - time exceeded",
			msg:  icmp.Message{Type: ipv4.ICMPTypeTimeExceeded, Code: 0, Body: &icmp.TimeExceeded{Data: []byte("hello world")}},
			want: `level=INFO msg=msg msg.type="time exceeded"
`,
		},
		{
			name: "ipv6 - request",
			msg:  icmp.Message{Type: ipv6.ICMPTypeEchoRequest, Code: 0, Body: &icmp.Echo{ID: 1, Seq: 1, Data: []byte("hello world")}},
			want: `level=INFO msg=msg msg.type="echo request" msg.seq=1
`,
		},
		{
			name: "ipv6 - response",
			msg:  icmp.Message{Type: ipv6.ICMPTypeEchoReply, Code: 0, Body: &icmp.Echo{ID: 1, Seq: 1, Data: []byte("hello world")}},
			want: `level=INFO msg=msg msg.type="echo reply" msg.seq=1
`,
		},
		{
			name: "ipv6 - time exceeded",
			msg:  icmp.Message{Type: ipv6.ICMPTypeTimeExceeded, Code: 0, Body: &icmp.TimeExceeded{Data: []byte("hello world")}},
			want: `level=INFO msg=msg msg.type="time exceeded"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var log bytes.Buffer
			l := testutils.NewTextLogger(&log, slog.LevelInfo)
			l.Info("msg", "msg", messageLogger(tt.msg))
			assert.Equal(t, tt.want, log.String())
		})
	}

}
