package collector

import (
	"bytes"
	"github.com/clambin/pinger/internal/pinger"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestPinger_Collect(t *testing.T) {
	p := Collector{Trackers: fakeTracker{}, Logger: slog.Default()}

	err := testutil.CollectAndCompare(p, bytes.NewBufferString(`# HELP pinger_latency_seconds Average latency in seconds
# TYPE pinger_latency_seconds gauge
pinger_latency_seconds{host="localhost"} 0.2
# HELP pinger_packet_count Total packet count
# TYPE pinger_packet_count gauge
pinger_packet_count{host="localhost"} 10
# HELP pinger_packet_loss_count Total measured packet loss
# TYPE pinger_packet_loss_count gauge
pinger_packet_loss_count{host="localhost"} 0.5
`))
	require.NoError(t, err)
}

var _ Trackers = fakeTracker{}

type fakeTracker struct{}

func (f fakeTracker) Statistics() map[string]pinger.Statistics {
	return map[string]pinger.Statistics{
		"localhost": {
			Sent:      20,
			Rcvd:      10,
			Latencies: []time.Duration{200 * time.Millisecond},
		},
	}
}
