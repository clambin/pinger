package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	Init(8080)
	Measure("test", 10, 1, 50*time.Millisecond)

	_, err := packetsCounter.GetMetricWith(prometheus.Labels{"host": "test"})
	assert.Nil(t, err)
}


