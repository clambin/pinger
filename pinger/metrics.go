package pinger

import (
	"github.com/clambin/gotools/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	packetsCounter = metrics.NewCounterVec(prometheus.CounterOpts{
		Name: "pinger_packet_count",
		Help: "Pinger total packet count",
	},
		[]string{"host"})

	lossCounter = metrics.NewCounterVec(prometheus.CounterOpts{
		Name: "pinger_packet_loss_count",
		Help: "Pinger total measured packet loss",
	},
		[]string{"host"})

	// TODO: better as a Gauge?
	latencyCounter = metrics.NewCounterVec(prometheus.CounterOpts{
		Name: "pinger_latency_seconds",
		Help: "Pinger total measured packet loss",
	},
		[]string{"host"})
)
