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
		prometheus.BuildFQName("pinger", "", "packet_count"),
		"Total packet count",
		[]string{"host"},
		nil,
	)
	lossMetric = prometheus.NewDesc(
		prometheus.BuildFQName("pinger", "", "packet_loss_count"),
		"Total measured packet loss",
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
	Trackers Trackers
	Logger   *slog.Logger
}

type Trackers interface {
	Statistics() map[string]pinger.Statistics
}

// Describe implements the Prometheus Collector interface
func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- packetsSentMetric
	ch <- packetsReceivedMetric
	ch <- lossMetric
	ch <- latencyMetric
}

// Collect implements the Prometheus Collector interface
func (c Collector) Collect(ch chan<- prometheus.Metric) {
	for name, t := range c.Trackers.Statistics() {
		loss := t.Loss()
		latency := t.Latency()
		c.Logger.Info("statistics", "target", name, "sent", t.Sent, "rcvd", t.Rcvd, "loss", math.Trunc(loss*1000)/10, "latency", latency)
		ch <- prometheus.MustNewConstMetric(packetsSentMetric, prometheus.GaugeValue, float64(t.Sent), name)
		ch <- prometheus.MustNewConstMetric(packetsReceivedMetric, prometheus.GaugeValue, float64(t.Rcvd), name)
		ch <- prometheus.MustNewConstMetric(lossMetric, prometheus.GaugeValue, loss, name)
		ch <- prometheus.MustNewConstMetric(latencyMetric, prometheus.GaugeValue, latency.Seconds(), name)
	}
}
