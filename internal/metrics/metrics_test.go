package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"pinger/internal/metrics"
)

func TestMetrics(t *testing.T) {
	var err error
	metrics.Init(8080)
	metrics.Measure("test", 10, 1, 50*time.Millisecond)

	_, err = metrics.LoadValue("pinger_packet_count", "test")
	assert.Nil(t, err)
	_, err = metrics.LoadValue("pinger_packet_loss_count", "test")
	assert.Nil(t, err)
	_, err = metrics.LoadValue("pinger_latency_seconds", "test")
	assert.Nil(t, err)

	assert.Panics(t, func() { metrics.Init(8080) })

}
