package icmp

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"
)

var logger = slog.New(slog.DiscardHandler)

func TestSocket_Ping_IPv4(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping ICMP test in GitHub Actions")
	}

	s, err := New(IPv4, logger)
	require.NoError(t, err)
	ip, err := s.Resolve("127.0.0.1")
	if err != nil {
		t.Skip(fmt.Errorf("IPv4 not supported: %w", err))
	}

	ctx := t.Context()
	go s.Serve(ctx)

	require.NoError(t, s.Ping(ip, 1, 255, []byte("payload")))

	response, err := s.Read(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", response.From.String())
	assert.Equal(t, ipv4.ICMPTypeEchoReply, response.MsgType)
	assert.Equal(t, SequenceNumber(1), response.SequenceNumber())
	assert.NotZero(t, response.Received)
}

func TestSocket_Ping_IPv6(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping ICMP test in GitHub Actions")
	}
	s, err := New(IPv6, logger)
	require.NoError(t, err)

	ip, err := s.Resolve("::1")
	if err != nil {
		t.Skip(fmt.Errorf("IPv6 not supported: %w", err))
	}

	ctx := t.Context()
	go s.Serve(ctx)

	require.NoError(t, s.Ping(ip, 1, 0, []byte("payload")))

	response, err := s.Read(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "::1", response.From.String())
	assert.Equal(t, ipv6.ICMPTypeEchoReply, response.MsgType)
	assert.Equal(t, SequenceNumber(1), response.SequenceNumber())
	assert.NotZero(t, response.Received)
}

func TestTransport_String(t *testing.T) {
	tests := []struct {
		name string
		tp   Transport
		want string
	}{
		{name: "IPv4", tp: IPv4, want: "ipv4"},
		{name: "IPv6", tp: IPv6, want: "ipv6"},
		{name: "unknown", tp: -1, want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tp.String())
		})
	}
}

func TestSocket_Resolve(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping ICMP test in GitHub Actions")
	}

	tests := []struct {
		name    string
		tp      Transport
		addr    string
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "IPv4",
			tp:      IPv4,
			addr:    "127.0.0.1",
			want:    "127.0.0.1",
			wantErr: assert.NoError,
		},
		{
			name:    "IPv6",
			tp:      IPv6,
			addr:    "::1",
			want:    "::1",
			wantErr: assert.NoError,
		},
		{
			name:    "IPv6 not supported",
			tp:      IPv4,
			addr:    "::1",
			want:    "<nil>",
			wantErr: assert.Error,
		},
		{
			name:    "invalid hostname",
			tp:      IPv4,
			addr:    "",
			want:    "<nil>",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(tt.tp, logger)
			require.NoError(t, err)
			addr, err := s.Resolve(tt.addr)
			assert.Equal(t, tt.want, addr.String())
			tt.wantErr(t, err)
		})
	}
}

func Test_responseQueue(t *testing.T) {
	q := newResponseQueue()

	_, ok := q.pop()
	require.False(t, ok)

	q.push(Response{})
	_, ok = q.pop()
	require.True(t, ok)
	_, ok = q.pop()
	require.False(t, ok)
	assert.Zero(t, q.len())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	t.Cleanup(cancel)
	_, err := q.popWait(ctx)
	assert.Error(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	t.Cleanup(cancel)
	_, err = q.popWait(ctx)
	assert.Error(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)
	errCh := make(chan error)
	go func() {
		_, err = q.popWait(ctx)
		errCh <- err
	}()
	time.Sleep(10 * time.Millisecond)
	go q.push(Response{})

	assert.NoError(t, <-errCh)
}

func TestResponse_LogValue(t *testing.T) {
	type fields struct {
		From    net.IP
		MsgType icmp.Type
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "IPv4",
			fields: fields{
				From:    net.ParseIP("127.0.0.1"),
				MsgType: ipv4.ICMPTypeEchoReply,
			},
			want: `[from=127.0.0.1 msgType=echo reply seq=10]`,
		},
		{
			name: "IPv6",
			fields: fields{
				From:    net.ParseIP("::1"),
				MsgType: ipv6.ICMPTypeEchoReply,
			},
			want: `[from=::1 msgType=echo reply seq=10]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Response{
				From:     tt.fields.From,
				MsgType:  tt.fields.MsgType,
				Body:     &icmp.Echo{Seq: 10},
				Received: time.Date(2024, time.August, 23, 15, 35, 0, 0, time.UTC),
			}
			assert.Equal(t, tt.want, r.LogValue().String())
		})
	}
}
