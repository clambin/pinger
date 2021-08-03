package pinger

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// Describe interface for Prometheus collector
func (pinger *Pinger) Describe(ch chan<- *prometheus.Desc) {
	ch <- pinger.packetsMetric
	ch <- pinger.lossMetric
	ch <- pinger.latencyMetric
}

// Collect interface for Prometheus collector
func (pinger *Pinger) Collect(ch chan<- prometheus.Metric) {
	for host, tracker := range pinger.Trackers {
		count, loss, latency := tracker.Calculate()

		log.WithFields(log.Fields{"host": host, "count": count, "loss": loss, "latency": latency}).Debug()

		ch <- prometheus.MustNewConstMetric(pinger.packetsMetric, prometheus.GaugeValue, float64(count), host)
		ch <- prometheus.MustNewConstMetric(pinger.lossMetric, prometheus.GaugeValue, float64(loss), host)
		ch <- prometheus.MustNewConstMetric(pinger.latencyMetric, prometheus.GaugeValue, latency.Seconds(), host)
	}
}
