package pinger

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNew_IPv4(t *testing.T) {
	ch := make(chan Response)
	c := MustNew(ch, "127.0.0.1")

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

func TestNew_IPv6(t *testing.T) {
	ch := make(chan Response)
	c := MustNew(ch, "::1")
	if !c.socket.HasIPv6() {
		t.Skip("build system does not have IPv6 enabled. skipping")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go c.Run(ctx, 10*time.Millisecond)

	p := <-ch
	assert.Equal(t, "::1", p.Host)
	assert.Equal(t, 0, p.SequenceNr)
	p = <-ch
	assert.Equal(t, "::1", p.Host)
	assert.Equal(t, 1, p.SequenceNr)

	cancel()
}

func TestMustNew_Panic(t *testing.T) {
	assert.Panics(t, func() {
		ch := make(chan Response)
		_ = MustNew(ch, "127.0.0.256")
	})
}

func TestWrap(t *testing.T) {
	ch := make(chan Response)
	c := MustNew(ch, "127.0.0.1")
	c.targets[0].seq = 0xfffe

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, time.Millisecond)

	p := <-ch
	assert.Equal(t, 0xfffe, p.SequenceNr)
	p = <-ch
	assert.Equal(t, 0xffff, p.SequenceNr)
	p = <-ch
	assert.Equal(t, 0x0000, p.SequenceNr)
	p = <-ch
	assert.Equal(t, 0x0001, p.SequenceNr)
}

func TestPinger_Multiple(t *testing.T) {
	ch := make(chan Response)
	c := MustNew(ch, "localhost", "::1")
	if !c.socket.HasIPv6() {
		t.Skip("build system does not have IPv6 enabled. skipping")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, 10*time.Millisecond)

	counts := make(map[string]int)
	for i := 0; i < 6; i++ {
		p := <-ch
		current := counts[p.Host]
		counts[p.Host] = current + 1
	}
	assert.Equal(t, 3, counts["localhost"])
	assert.Equal(t, 3, counts["::1"])
}
