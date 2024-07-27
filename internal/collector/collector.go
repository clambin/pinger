package collector

import (
	"github.com/clambin/pinger/internal/pinger"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
)

var (
	packetsMetric = prometheus.NewDesc(
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
	ch <- packetsMetric
	ch <- lossMetric
	ch <- latencyMetric
}

// Collect implements the Prometheus Collector interface
func (c Collector) Collect(ch chan<- prometheus.Metric) {
	for name, t := range c.Trackers.Statistics() {
		loss := t.Loss()
		latency := t.Latency()
		c.Logger.Info("statistics", "target", name, "rcvd", t.Rcvd, "loss", loss, "latency", latency)
		ch <- prometheus.MustNewConstMetric(packetsMetric, prometheus.GaugeValue, float64(t.Rcvd), name)
		ch <- prometheus.MustNewConstMetric(lossMetric, prometheus.GaugeValue, loss, name)
		ch <- prometheus.MustNewConstMetric(latencyMetric, prometheus.GaugeValue, latency.Seconds(), name)
	}
}
