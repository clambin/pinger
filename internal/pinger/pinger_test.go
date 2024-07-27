package pinger

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStatistics(t *testing.T) {
	tests := []struct {
		name        string
		stats       Statistics
		wantLoss    float64
		wantLatency time.Duration
	}{
		{
			name: "statistics",
			stats: Statistics{
				Sent:      10,
				Rcvd:      5,
				Latencies: []time.Duration{time.Second, 1500 * time.Millisecond, 500 * time.Millisecond},
			},
			wantLoss:    .5,
			wantLatency: time.Second,
		},
		{
			name:        "no statistics",
			stats:       Statistics{},
			wantLoss:    0,
			wantLatency: 0,
		},
		{
			name:        "late arrival",
			stats:       Statistics{Sent: 1, Rcvd: 2},
			wantLoss:    0,
			wantLatency: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantLoss, tt.stats.Loss())
			assert.Equal(t, tt.wantLatency, tt.stats.Latency())
		})
	}
}
