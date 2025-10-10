package collector

import (
	"log/slog"

	"github.com/clambin/pinger/internal/pinger"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	packetsSentMetric = prometheus.NewDesc(
		prometheus.BuildFQName("pinger", "", "packets_sent_count"),
		"Total packets sent",
		[]string{"host"},
		nil,
	)
	packetsReceivedMetric = prometheus.NewDesc(
		prometheus.BuildFQName("pinger", "", "packets_received_count"),
		"Total packet received",
		[]string{"host"},
		nil,
	)
	latencyMetric = prometheus.NewDesc(
		prometheus.BuildFQName("pinger", "", "latency_seconds"),
		"Average latency in seconds",
		[]string{"host"},
		nil,
	)
)

type Targets interface {
	Statistics() map[string]pinger.Statistics
}

// Collector pings a number of hosts and measures latency & packet loss
type Collector struct {
	Targets Targets
	Logger  *slog.Logger
}

// Describe implements the Prometheus Collector interface
func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- packetsSentMetric
	ch <- packetsReceivedMetric
	ch <- latencyMetric
}

// Collect implements the Prometheus Collector interface
func (c Collector) Collect(ch chan<- prometheus.Metric) {
	for name, statistics := range c.Targets.Statistics() {
		c.Logger.Info("statistics", "target", name, "sent", statistics.Sent, "rcvd", statistics.Received, "latency", statistics.Latency)
		ch <- prometheus.MustNewConstMetric(packetsSentMetric, prometheus.CounterValue, float64(statistics.Sent), name)
		ch <- prometheus.MustNewConstMetric(packetsReceivedMetric, prometheus.CounterValue, float64(statistics.Received), name)
		ch <- prometheus.MustNewConstMetric(latencyMetric, prometheus.GaugeValue, statistics.Latency.Seconds(), name)
	}
}
