package collector

import (
	"bytes"
	"github.com/clambin/pinger/pkg/ping"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"iter"
	"log/slog"
	"testing"
	"time"
)

func TestPinger_Collect(t *testing.T) {
	p := Collector{Pinger: fakeTracker{}, Logger: slog.Default()}

	err := testutil.CollectAndCompare(p, bytes.NewBufferString(`
# HELP pinger_latency_seconds Average latency in seconds
# TYPE pinger_latency_seconds gauge
pinger_latency_seconds{host="localhost"} 0.2

# HELP pinger_packets_sent_count Total packets sent
# TYPE pinger_packets_sent_count counter
pinger_packets_sent_count{host="localhost"} 20

# HELP pinger_packets_received_count Total packet received
# TYPE pinger_packets_received_count counter
pinger_packets_received_count{host="localhost"} 10
`))
	require.NoError(t, err)
}

var _ Pinger = fakeTracker{}

type fakeTracker struct{}

func (f fakeTracker) Statistics() iter.Seq2[string, ping.Statistics] {
	return func(yield func(string, ping.Statistics) bool) {
		yield("localhost", ping.Statistics{
			Sent:     20,
			Received: 10,
			Latency:  200 * time.Millisecond,
		})
	}
}
