package collector

import (
	"github.com/clambin/pinger/internal/pinger"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"math"
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

// Collector pings a number of hosts and measures latency & packet loss
type Collector struct {
	Pinger Pinger
	Logger *slog.Logger
}

type Pinger interface {
	Statistics() map[string]pinger.Statistics
}

// Describe implements the Prometheus Collector interface
func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- packetsSentMetric
	ch <- packetsReceivedMetric
	ch <- latencyMetric
}

// Collect implements the Prometheus Collector interface
func (c Collector) Collect(ch chan<- prometheus.Metric) {
	for name, statistics := range c.Pinger.Statistics() {
		sent, received := statistics.Sent, statistics.Received //adjustedSentReceived(statistics)
		c.Logger.Info("statistics", "target", name, "sent", statistics.Sent, "rcvd", statistics.Received, "latency", statistics.Latency)
		ch <- prometheus.MustNewConstMetric(packetsSentMetric, prometheus.CounterValue, float64(sent), name)
		ch <- prometheus.MustNewConstMetric(packetsReceivedMetric, prometheus.CounterValue, float64(received), name)
		ch <- prometheus.MustNewConstMetric(latencyMetric, prometheus.GaugeValue, statistics.Latency.Seconds(), name)
	}
}

func adjustedSentReceived(statistics pinger.Statistics) (int, int) {
	// sent/received may be off by one (packet sent, but response not yet received)
	// if this is the case, adjust the numbers
	sent, received := statistics.Sent, statistics.Received
	if math.Abs(float64(sent-received)) == 1 {
		sent = min(sent, received)
		received = sent
	}
	return sent, received
}
