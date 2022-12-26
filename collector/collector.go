package collector

import (
	"context"
	"github.com/clambin/pinger/collector/pinger"
	"github.com/clambin/pinger/collector/tracker"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// Collector pings a number of hosts and measures latency & packet loss
type Collector struct {
	Pinger   func(ch chan pinger.Response, hosts ...string) (err error)
	Trackers map[string]*tracker.Tracker
	packets  chan pinger.Response
}

// New creates a Collector for the specified hosts
func New(hosts []string) (monitor *Collector) {
	monitor = &Collector{
		//Pinger:   pinger.SpawnedPingers,
		Pinger:   pinger.ICMPPingers,
		Trackers: make(map[string]*tracker.Tracker),
		packets:  make(chan pinger.Response),
	}

	for _, host := range hosts {
		monitor.Trackers[host] = tracker.New()
	}

	return
}

// Run starts the collector(s)
func (c *Collector) Run(ctx context.Context) {
	go c.startPingers()

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case packet := <-c.packets:
			c.Trackers[packet.Host].Track(packet.SequenceNr, packet.Latency)
		}
	}
}

func (c *Collector) startPingers() {
	var hosts []string
	for host := range c.Trackers {
		hosts = append(hosts, host)
	}
	_ = c.Pinger(c.packets, hosts...)
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

		log.WithFields(log.Fields{"host": host, "count": count, "loss": loss, "latency": latency}).Debug()

		ch <- prometheus.MustNewConstMetric(packetsMetric, prometheus.GaugeValue, float64(count), host)
		ch <- prometheus.MustNewConstMetric(lossMetric, prometheus.GaugeValue, float64(loss), host)
		ch <- prometheus.MustNewConstMetric(latencyMetric, prometheus.GaugeValue, latency.Seconds(), host)
	}
}
