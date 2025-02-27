package ping

import (
	"github.com/clambin/pinger/pkg/ping/icmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTarget(t *testing.T) {
	var target Target

	// empty statistics
	assert.Zero(t, target.Statistics())

	// two outstanding requests
	target.Sent(1)
	target.Sent(2)
	assert.Equal(t, 2, target.Statistics().Sent)

	// one response received
	target.Received(true, 1)
	statistics := target.Statistics()
	assert.Equal(t, 2, statistics.Sent)
	assert.Equal(t, 1, statistics.Received)
	assert.NotZero(t, statistics.Latency)

	// second response times out
	assert.Equal(t, []icmp.SequenceNumber{2}, target.timeout(0))
	statistics = target.Statistics()
	assert.Equal(t, 2, statistics.Sent)
	assert.Equal(t, 1, statistics.Received)
	assert.NotZero(t, statistics.Latency)

	// reset zeroes the statistics
	target.ResetStatistics()
	assert.Zero(t, target.Statistics())
}
