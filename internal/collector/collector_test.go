package collector

import (
	"bytes"
	"context"
	"github.com/clambin/pinger/pkg/pinger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPinger_Collect(t *testing.T) {
	target := pinger.Target{Host: "127.0.0.1", Name: "localhost"}
	p := New([]pinger.Target{target})

	p.trackers[target].Track(0, 150*time.Millisecond)
	p.trackers[target].Track(1, 50*time.Millisecond)

	err := testutil.CollectAndCompare(p, bytes.NewBufferString(`# HELP pinger_latency_seconds Average latency in seconds
# TYPE pinger_latency_seconds gauge
pinger_latency_seconds{host="localhost"} 0.2
# HELP pinger_packet_count Total packet count
# TYPE pinger_packet_count gauge
pinger_packet_count{host="localhost"} 2
# HELP pinger_packet_loss_count Total measured packet loss
# TYPE pinger_packet_loss_count gauge
pinger_packet_loss_count{host="localhost"} 0
`))
	require.NoError(t, err)

	p.trackers[target].Track(3, 100*time.Millisecond)
	err = testutil.CollectAndCompare(p, bytes.NewBufferString(`# HELP pinger_latency_seconds Average latency in seconds
# TYPE pinger_latency_seconds gauge
pinger_latency_seconds{host="localhost"} 0.1
# HELP pinger_packet_count Total packet count
# TYPE pinger_packet_count gauge
pinger_packet_count{host="localhost"} 1
# HELP pinger_packet_loss_count Total measured packet loss
# TYPE pinger_packet_loss_count gauge
pinger_packet_loss_count{host="localhost"} 1
`))
	require.NoError(t, err)
}

func TestPinger_Run(t *testing.T) {
	p := New([]pinger.Target{
		{Host: "127.0.0.1", Name: "localhost1"},
		{Host: "localhost", Name: "localhost2"},
	})

	r := prometheus.NewPedanticRegistry()
	r.MustRegister(p)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Run(ctx)

	var m metrics
	var err error

	assert.Eventually(t, func() bool {
		m, err = getMetrics(r)
		if err != nil {
			return false
		}
		return m["pinger_packet_count"] != nil &&
			m["pinger_packet_count"]["localhost1"] > 0 &&
			m["pinger_packet_count"]["localhost2"] > 0
	}, time.Second, time.Millisecond)

	assert.Contains(t, m, "pinger_latency_seconds")
	assert.Contains(t, m, "pinger_packet_loss_count")
	assert.Contains(t, m, "pinger_packet_count")
}

type metrics map[string]map[string]float64

func getMetrics(r *prometheus.Registry) (metrics, error) {
	m, err := r.Gather()
	if err != nil {
		return nil, err
	}
	metrics := make(metrics)
	for i := range m {
		name := m[i].GetName()
		if !(name == "pinger_packet_count" || name == "pinger_packet_loss_count" || name == "pinger_latency_seconds") {
			continue
		}
		if _, ok := metrics[name]; !ok {
			metrics[name] = make(map[string]float64)
		}
		for j := range m[i].GetMetric() {
			for _, label := range m[i].GetMetric()[j].Label {
				if label.GetName() == "host" {
					metrics[name][label.GetValue()] = m[i].GetMetric()[j].Gauge.GetValue()
				}
			}
		}
	}
	return metrics, nil
}
