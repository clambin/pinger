package collector_test

import (
	"bytes"
	"context"
	"github.com/clambin/pinger/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"os"
	"testing"
	"time"
)

func TestPinger_Collect(t *testing.T) {
	p := collector.New([]string{"localhost"})

	p.Trackers["localhost"].Track(0, 150*time.Millisecond)
	p.Trackers["localhost"].Track(1, 50*time.Millisecond)

	r := prometheus.NewPedanticRegistry()
	r.MustRegister(p)

	err := testutil.GatherAndCompare(r, bytes.NewBufferString(`# HELP pinger_latency_seconds Average latency in seconds
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

	p.Trackers["localhost"].Track(3, 100*time.Millisecond)
	err = testutil.GatherAndCompare(r, bytes.NewBufferString(`# HELP pinger_latency_seconds Average latency in seconds
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
	ops := slog.HandlerOptions{Level: slog.LevelDebug}
	slog.SetDefault(slog.New(ops.NewTextHandler(os.Stdout)))
	p := collector.New([]string{"127.0.0.1"})
	r := prometheus.NewPedanticRegistry()
	r.MustRegister(p)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Run(ctx)

	var metrics []*io_prometheus_client.MetricFamily
	var err error

	// wait for 1 packet to arrive
	assert.Eventually(t, func() bool {
		metrics, err = r.Gather()
		require.NoError(t, err)
		for _, metric := range metrics {
			if metric.GetName() == "pinger_packet_count" {
				return metric.Metric[0].GetGauge().GetValue() > 0
			}
		}
		return false
	}, 5*time.Second, 10*time.Millisecond)

	for _, metric := range metrics {
		switch metric.GetName() {
		case "pinger_packet_count":
			assert.NotZero(t, metric.Metric[0].GetGauge().GetValue())
		case "pinger_latency_seconds":
			assert.NotZero(t, metric.Metric[0].GetGauge().GetValue())
		case "pinger_packet_loss_count":
			assert.Zero(t, metric.Metric[0].GetGauge().GetValue())
		default:
			t.Fatal(metric.GetName())
		}
	}
}
