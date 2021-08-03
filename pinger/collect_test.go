package pinger_test

import (
	"github.com/clambin/pinger/pinger"
	"github.com/prometheus/client_golang/prometheus"
	pcg "github.com/prometheus/client_model/go"
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
	assert.Equal(t, "foo", metricLabel(m, "host"))
	assert.Equal(t, 2.0, getMetric(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", metricLabel(m, "host"))
	assert.Equal(t, 0.0, getMetric(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", metricLabel(m, "host"))
	assert.Equal(t, 0.2, getMetric(m).GetGauge().GetValue())

	p.Trackers["foo"].Track(3, 100*time.Millisecond)
	go p.Collect(ch)

	m = <-ch
	assert.Equal(t, "foo", metricLabel(m, "host"))
	assert.Equal(t, 1.0, getMetric(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", metricLabel(m, "host"))
	assert.Equal(t, 1.0, getMetric(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", metricLabel(m, "host"))
	assert.Equal(t, 0.1, getMetric(m).GetGauge().GetValue())

}

// metricLabel returns the value for a specified label
func metricLabel(metric prometheus.Metric, labelName string) string {
	var m pcg.Metric

	if metric.Write(&m) != nil {
		panic("failed to parse metric")
	}

	for _, label := range m.GetLabel() {
		if label.GetName() == labelName {
			return label.GetValue()
		}
	}

	return ""
}
