package pinger_test

import (
	"context"
	"github.com/clambin/pinger/collector/pinger"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestICMPPingers(t *testing.T) {
	ch := make(chan pinger.Response)
	c := pinger.MustNew(ch, "127.0.0.1")

	ctx, cancel := context.WithCancel(context.Background())
	go c.Run(ctx, 10*time.Millisecond)

	p := <-ch
	assert.Equal(t, "127.0.0.1", p.Host)
	assert.Equal(t, 0, p.SequenceNr)
	p = <-ch
	assert.Equal(t, "127.0.0.1", p.Host)
	assert.Equal(t, 1, p.SequenceNr)

	cancel()
}
