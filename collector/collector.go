package collector

import (
	"context"
	"github.com/clambin/pinger/collector/pinger"
	"github.com/clambin/pinger/collector/tracker"
	"github.com/clambin/pinger/configuration"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"time"
)

// Collector pings a number of hosts and measures latency & packet loss
type Collector struct {
	Pinger   *pinger.Pinger
	Trackers map[configuration.Target]*tracker.Tracker
	Packets  chan pinger.Response
}

// New creates a Collector for the specified hosts
func New(targets configuration.Targets) (monitor *Collector) {
	ch := make(chan pinger.Response)
	monitor = &Collector{
		Pinger:   pinger.MustNew(ch, targets),
		Trackers: make(map[configuration.Target]*tracker.Tracker),
		Packets:  ch,
	}

	for _, target := range targets {
		monitor.Trackers[target] = tracker.New()
	}

	return
}

// Run starts the collector(s)
func (c *Collector) Run(ctx context.Context) {
	go c.Pinger.Run(ctx, time.Second)

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case packet := <-c.Packets:
			c.Trackers[packet.Target].Track(packet.SequenceNr, packet.Latency)
		}
	}
}

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

// Describe implements the Prometheus Collector interface
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- packetsMetric
	ch <- lossMetric
	ch <- latencyMetric
}

// Collect implements the Prometheus Collector interface
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	for host, t := range c.Trackers {
		count, loss, latency := t.Calculate()

		if count > 0 {
			slog.Debug("stats", "host", host.GetName(), "count", count, "loss", loss, "latency", latency)
		}

		ch <- prometheus.MustNewConstMetric(packetsMetric, prometheus.GaugeValue, float64(count), host.GetName())
		ch <- prometheus.MustNewConstMetric(lossMetric, prometheus.GaugeValue, float64(loss), host.GetName())
		ch <- prometheus.MustNewConstMetric(latencyMetric, prometheus.GaugeValue, latency.Seconds(), host.GetName())
	}
}
