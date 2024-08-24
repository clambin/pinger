package icmp

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"log/slog"
)

var _ slog.LogValuer = messageLogger{}

type messageLogger icmp.Message

func (m messageLogger) LogValue() slog.Value {
	attrs := []slog.Attr{slog.Any("type", m.Type)}
	switch m.Type {
	case ipv4.ICMPTypeEcho, ipv6.ICMPTypeEchoRequest, ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
		b := m.Body.(*icmp.Echo)
		attrs = append(attrs, slog.Int("seq", b.Seq))
	case ipv4.ICMPTypeTimeExceeded, ipv6.ICMPTypeTimeExceeded:
		//b := m.Body.(*icmp.TimeExceeded)
		//attrs = append(attrs, slog.String("data", string(b.Data)))
	}
	return slog.GroupValue(attrs...)
}
