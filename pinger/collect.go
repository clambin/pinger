package pinger

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// Describe interface for Prometheus collector
func (monitor *Monitor) Describe(ch chan<- *prometheus.Desc) {
	ch <- monitor.packetsMetric
	ch <- monitor.lossMetric
	ch <- monitor.latencyMetric
}

// Collect interface for Prometheus collector
func (monitor *Monitor) Collect(ch chan<- prometheus.Metric) {
	for host, tracker := range monitor.Trackers {
		count, loss, latency := tracker.Calculate()

		log.WithFields(log.Fields{"host": host, "count": count, "loss": loss, "latency": latency}).Debug()

		ch <- prometheus.MustNewConstMetric(monitor.packetsMetric, prometheus.GaugeValue, float64(count), host)
		ch <- prometheus.MustNewConstMetric(monitor.lossMetric, prometheus.GaugeValue, float64(loss), host)
		ch <- prometheus.MustNewConstMetric(monitor.latencyMetric, prometheus.GaugeValue, latency.Seconds(), host)
	}
}
