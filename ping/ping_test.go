package ping_test

import (
	"errors"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/clambin/pinger/ping"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSocket(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ping.SocketOption
		network string
		address string
		target  string
	}{
		{"IPv4", []ping.SocketOption{ping.WithIPv4(), ping.WithTimeout(10 * time.Second)}, "udp4", "8.8.8.8:53", "127.0.0.1"},
		{"IPv6", []ping.SocketOption{ping.WithIPv6()}, "udp6", "[2001:4860:4860::8888]:53", "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// check if we have the required IP version
			if _, err := net.Dial(tt.network, tt.address); err != nil {
				t.Skip("IP version not supported")
			}

			opts := append(tt.opts, ping.WithLogger(slog.New(slog.DiscardHandler)))
			socket, err := ping.New(opts...)
			if errors.Is(err, os.ErrPermission) {
				t.Skip("IP version not supported")
			}
			require.NoError(t, err)

			ctx := t.Context()
			go socket.Serve(ctx)

			target, err := socket.Resolve(tt.target)
			require.NoError(t, err)

			err = socket.Send(target, 1, 255, []byte("payload"))
			require.NoError(t, err)

			resp, err := socket.Read(ctx)
			require.NoError(t, err)
			assert.Equal(t, ping.ResponseEchoReply, resp.ResponseType)
			assert.Equal(t, tt.target, resp.From.String())
			assert.Equal(t, ping.SequenceNumber(1), resp.Request.Seq)
		})
	}
}

func TestResponse_LogValue(t *testing.T) {
	tests := []struct {
		name string
		resp ping.Response
		want string
	}{
		{
			name: "timeout",
			resp: ping.Response{ResponseType: ping.ResponseTimeout},
			want: `[type=timeout]`,
		},
		{
			name: "time exceeded",
			resp: ping.Response{ResponseType: ping.ResponseTimeExceeded, From: net.ParseIP("192.168.0.1"), Request: ping.Request{Target: net.ParseIP("1.1.1.1"), Seq: 10, TTL: 1}},
			want: `[type=time exceeded from=192.168.0.1 target=1.1.1.1 seq=10 ttl=1]`,
		},
		{
			name: "echo reply",
			resp: ping.Response{ResponseType: ping.ResponseEchoReply, From: net.ParseIP("192.168.0.1"), Request: ping.Request{Target: net.ParseIP("192.168.0.1"), Seq: 2, TTL: 64}},
			want: `[type=echo reply from=192.168.0.1 target=192.168.0.1 seq=2 ttl=64]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.resp.LogValue().String())
		})
	}
}

func TestSocket_Timeout(t *testing.T) {
	s, err := ping.New(
		ping.WithIPv4(),
		ping.WithTimeout(time.Second),
		ping.WithLogger(slog.New(slog.DiscardHandler)),
	)
	if errors.Is(err, os.ErrPermission) {
		t.Skip("IPv4 not supported")
	}
	require.NoError(t, err)

	go s.Serve(t.Context())

	target := net.ParseIP("127.0.0.2")
	require.NoError(t, s.Send(target, 10, 64, []byte("payload")))

	// wait for the timeout response
	var resp ping.Response
	require.Eventually(t, func() bool {
		resp, err = s.Read(t.Context())
		return err == nil
	}, 5*time.Second, 500*time.Millisecond)

	want := ping.Response{
		ResponseType: ping.ResponseTimeout,
		Request:      ping.Request{Target: target, Seq: 10, TTL: 64},
	}
	// clear TimeSent so we can compare but check it's set in the response
	assert.NotZero(t, resp.Request.TimeSent)
	resp.Request.TimeSent = time.Time{}
	assert.Equal(t, want, resp)
}
