package collector_test

import (
	"bytes"
	"context"
	"github.com/clambin/pinger/internal/collector"
	"github.com/clambin/pinger/pkg/pinger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPinger_Collect(t *testing.T) {
	target := pinger.Target{Host: "127.0.0.1", Name: "localhost"}
	p := collector.New([]pinger.Target{target})

	p.Trackers[target].Track(0, 150*time.Millisecond)
	p.Trackers[target].Track(1, 50*time.Millisecond)

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

	p.Trackers[target].Track(3, 100*time.Millisecond)
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
	//ops := slog.HandlerOptions{Level: slog.LevelDebug}
	//slog.SetDefault(slog.New(ops.NewTextHandler(os.Stdout)))

	p := collector.New([]pinger.Target{
		{Host: "127.0.0.1", Name: "localhost1"},
		{Host: "localhost", Name: "localhost2"},
	})

	r := prometheus.NewPedanticRegistry()
	r.MustRegister(p)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Run(ctx)

	var metrics []*io_prometheus_client.MetricFamily
	// wait for 1 packet to arrive for each target
	assert.Eventually(t, func() bool {
		var err error
		metrics, err = r.Gather()
		require.NoError(t, err)
		var packetsRcvd int
		for _, metric := range metrics {
			if metric.GetName() == "pinger_packet_count" {
				for _, m := range metric.Metric {
					if m.GetGauge().GetValue() > 0 {
						packetsRcvd++
					}
				}
			}
		}
		return packetsRcvd >= 2
	}, 5*time.Second, 10*time.Millisecond)

	var entries int
	for _, metric := range metrics {
		for _, m := range metric.Metric {
			entries++
			switch metric.GetName() {
			case "pinger_packet_count":
				assert.NotZero(t, m.GetGauge().GetValue())
			case "pinger_latency_seconds":
				assert.NotZero(t, m.GetGauge().GetValue())
			case "pinger_packet_loss_count":
				assert.Zero(t, m.GetGauge().GetValue())
			default:
				t.Fatal(metric.GetName())
			}
		}
	}
	// check two targets for same IP address are both counted: 3 metrics for 2 targets = 6 entries
	assert.Equal(t, 6, entries)
}
