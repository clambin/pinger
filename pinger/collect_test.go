package pinger_test

import (
	"github.com/clambin/go-metrics"
	"github.com/clambin/pinger/pinger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPinger_Describe(t *testing.T) {
	p := pinger.New([]string{"foo", "bar"})

	ch := make(chan *prometheus.Desc)
	go p.Describe(ch)

	for _, name := range []string{
		"pinger_packet_count",
		"pinger_packet_loss_count",
		"pinger_latency_seconds",
	} {
		metric := <-ch
		assert.Contains(t, metric.String(), "\""+name+"\"")
	}
}

func TestPinger_Collect(t *testing.T) {
	p := pinger.New([]string{"foo"})

	p.Trackers["foo"].Track(0, 150*time.Millisecond)
	p.Trackers["foo"].Track(1, 50*time.Millisecond)

	ch := make(chan prometheus.Metric)
	go p.Collect(ch)

	m := <-ch
	assert.Equal(t, "foo", metrics.MetricLabel(m, "host"))
	assert.Equal(t, 2.0, metrics.MetricValue(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", metrics.MetricLabel(m, "host"))
	assert.Equal(t, 0.0, metrics.MetricValue(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", metrics.MetricLabel(m, "host"))
	assert.Equal(t, 0.2, metrics.MetricValue(m).GetGauge().GetValue())

	p.Trackers["foo"].Track(3, 100*time.Millisecond)
	go p.Collect(ch)

	m = <-ch
	assert.Equal(t, "foo", metrics.MetricLabel(m, "host"))
	assert.Equal(t, 1.0, metrics.MetricValue(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", metrics.MetricLabel(m, "host"))
	assert.Equal(t, 1.0, metrics.MetricValue(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", metrics.MetricLabel(m, "host"))
	assert.Equal(t, 0.1, metrics.MetricValue(m).GetGauge().GetValue())

}
