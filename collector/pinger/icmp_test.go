package pinger

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"
	"testing"
	"time"
)

func TestICMPConnection(t *testing.T) {
	c, err := newConnection()
	require.NoError(t, err)

	addr, err := c.resolve("127.0.0.1")
	require.NoError(t, err)

	ch := make(chan packet)
	go func() {
		err2 := c.listen(ch)
		require.NoError(t, err2)
	}()

	for i := 0; i < 10; i++ {
		err = c.send(addr, i)
		require.NoError(t, err)
		p := <-ch
		assert.Equal(t, i, p.seqno)
		if nettest.SupportsRawSocket() {
			assert.Equal(t, "127.0.0.1", p.peer.String())
		} else {
			assert.Equal(t, "127.0.0.1:0", p.peer.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
