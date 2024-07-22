package socket_test

import (
	"context"
	"github.com/clambin/pinger/pkg/pinger/socket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

func TestSocket_Send(t *testing.T) {
	s, _ := socket.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan socket.Response)
	go s.Receive(ctx, ch)

	tests := []struct {
		name    string
		network string
		address string
		want    assert.ErrorAssertionFunc
	}{
		{name: "upd4", network: "udp4", address: "127.0.0.1:0", want: assert.NoError},
		{name: "upd6", network: "udp6", address: "[::1]:0", want: assert.NoError},
		{name: "invalid", network: "bad", address: "[::1]:0", want: assert.Error},
	}

	//FIXME: flaky test
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := net.ResolveUDPAddr(tt.network, tt.address)
			tt.want(t, err)

			if err != nil {
				return
			}

			for i := range 10 {
				require.NoError(t, s.Send(addr, tt.network, i))
				select {
				case response := <-ch:
					assert.Equal(t, tt.address, response.Addr.String())
					assert.Equal(t, "udp", response.Addr.Network())
					assert.Equal(t, i, response.Seq)
				case <-time.After(time.Second):
					t.Logf("%s: packet %d lost", t.Name(), i)
				}

				//time.Sleep(10 * time.Millisecond)
			}
		})
	}
}
