package collector

import (
	"context"
	"github.com/clambin/pinger/internal/collector/tracker"
	"github.com/clambin/pinger/pkg/pinger"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"time"
)

// Collector pings a number of hosts and measures latency & packet loss
type Collector struct {
	pinger    *pinger.Pinger
	responses chan pinger.Response
	trackers  map[pinger.Target]*tracker.Tracker
}

// New creates a Collector for the specified hosts
func New(targets pinger.Targets) *Collector {
	responses := make(chan pinger.Response)
	trackers := make(map[pinger.Target]*tracker.Tracker)
	for _, target := range targets {
		trackers[target] = tracker.New()
	}
	return &Collector{
		pinger:    pinger.MustNew(responses, targets),
		responses: responses,
		trackers:  trackers,
	}
}

// Run starts the collector(s)
func (c *Collector) Run(ctx context.Context) {
	go c.pinger.Run(ctx, time.Second)

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case packet := <-c.responses:
			c.trackers[packet.Target].Track(packet.SequenceNr, packet.Latency)
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
	for host, t := range c.trackers {
		count, loss, latency := t.Calculate()
		if count > 0 {
			slog.Debug("stats", "host", host.GetName(), "count", count, "loss", loss, "latency", latency)
		}
		ch <- prometheus.MustNewConstMetric(packetsMetric, prometheus.GaugeValue, float64(count), host.GetName())
		ch <- prometheus.MustNewConstMetric(lossMetric, prometheus.GaugeValue, float64(loss), host.GetName())
		ch <- prometheus.MustNewConstMetric(latencyMetric, prometheus.GaugeValue, latency.Seconds(), host.GetName())
	}
}
