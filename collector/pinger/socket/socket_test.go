package socket_test

import (
	"context"
	"github.com/clambin/pinger/collector/pinger/socket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

func TestSocket_Send(t *testing.T) {
	s, err := socket.New()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan socket.Response)
	go s.Receive(ctx, ch)

	tests := []struct {
		name    string
		network string
		address string
	}{
		{name: "upd4", network: "udp4", address: "127.0.0.1:0"},
		{name: "upd6", network: "udp6", address: "[::1]:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := net.ResolveUDPAddr(tt.network, tt.address)
			require.NoError(t, err)

			for i := 0; i < 10; i++ {
				err = s.Send(addr, tt.network, i)
				require.NoError(t, err)

				response := <-ch
				assert.Equal(t, tt.address, response.Addr.String())
				assert.Equal(t, "udp", response.Addr.Network())
				assert.Equal(t, i, response.Seq)

				time.Sleep(time.Millisecond)
			}
		})
	}
}
