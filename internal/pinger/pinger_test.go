package pinger_test

import (
	"testing"
	"time"

	"github.com/clambin/gotools/metrics"
	"github.com/stretchr/testify/assert"

	"pinger/internal/pinger"
)

func TestPinger(t *testing.T) {
	hosts := []string{"127.0.0.1"}

	go pinger.Run(hosts, 1*time.Second)

	time.Sleep(5 * time.Second)

	value, err := metrics.LoadValue("pinger_packet_count", "127.0.0.1")
	assert.Nil(t, err)
	assert.GreaterOrEqual(t, value, 4.0)

	value, err = metrics.LoadValue("pinger_packet_loss_count", "127.0.0.1")
	assert.Nil(t, err)
	assert.LessOrEqual(t, value, 3.0)

	value, err = metrics.LoadValue("pinger_latency_seconds", "127.0.0.1")
	assert.Nil(t, err)
	assert.Greater(t, value, 0.0)
}
