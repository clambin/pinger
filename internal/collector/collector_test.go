package collector

import (
	"bytes"
	"github.com/clambin/pinger/internal/pinger"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (f fakeTracker) Statistics() map[string]pinger.Statistics {
	return map[string]pinger.Statistics{
		"localhost": {
			Sent:     20,
			Received: 10,
			Latency:  200 * time.Millisecond,
		},
	}
}

func Test_adjustedSentReceived(t *testing.T) {
	tests := []struct {
		name         string
		args         pinger.Statistics
		wantSent     int
		wantReceived int
	}{
		{
			name:         "equal",
			args:         pinger.Statistics{Sent: 20, Received: 20},
			wantSent:     20,
			wantReceived: 20,
		},
		{
			name:         "sent off by one",
			args:         pinger.Statistics{Sent: 21, Received: 20},
			wantSent:     20,
			wantReceived: 20,
		},
		{
			name:         "received off by one",
			args:         pinger.Statistics{Sent: 20, Received: 21},
			wantSent:     20,
			wantReceived: 20,
		},
		{
			name:         "actual packet loss",
			args:         pinger.Statistics{Sent: 20, Received: 10},
			wantSent:     20,
			wantReceived: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSent, gotReceived := adjustedSentReceived(tt.args)
			assert.Equal(t, tt.wantSent, gotSent)
			assert.Equal(t, tt.wantReceived, gotReceived)
		})
	}
}
