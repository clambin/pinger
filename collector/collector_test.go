package collector_test

import (
	"context"
	"github.com/clambin/go-metrics/tools"
	"github.com/clambin/pinger/collector"
	"github.com/clambin/pinger/collector/pinger"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPinger_Describe(t *testing.T) {
	p := collector.New([]string{"foo", "bar"})

	ch := make(chan *prometheus.Desc)
	go p.Describe(ch)

	for _, name := range []string{
		"pinger_packet_count",
		"pinger_packet_loss_count",
		"pinger_latency_seconds",
	} {
		metric := <-ch
		assert.Contains(t, metric.String(), "\""+name+"\"")
	}
}

func TestPinger_Collect(t *testing.T) {
	p := collector.New([]string{"foo"})

	p.Trackers["foo"].Track(0, 150*time.Millisecond)
	p.Trackers["foo"].Track(1, 50*time.Millisecond)

	ch := make(chan prometheus.Metric)
	go p.Collect(ch)

	m := <-ch
	assert.Equal(t, "foo", tools.MetricLabel(m, "host"))
	assert.Equal(t, 2.0, tools.MetricValue(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", tools.MetricLabel(m, "host"))
	assert.Equal(t, 0.0, tools.MetricValue(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", tools.MetricLabel(m, "host"))
	assert.Equal(t, 0.2, tools.MetricValue(m).GetGauge().GetValue())

	p.Trackers["foo"].Track(3, 100*time.Millisecond)
	go p.Collect(ch)

	m = <-ch
	assert.Equal(t, "foo", tools.MetricLabel(m, "host"))
	assert.Equal(t, 1.0, tools.MetricValue(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", tools.MetricLabel(m, "host"))
	assert.Equal(t, 1.0, tools.MetricValue(m).GetGauge().GetValue())

	m = <-ch
	assert.Equal(t, "foo", tools.MetricLabel(m, "host"))
	assert.Equal(t, 0.1, tools.MetricValue(m).GetGauge().GetValue())

}

func TestPinger_Run_Quick(t *testing.T) {
	p := collector.New([]string{"127.0.0.1"})
	p.Pinger = fakePinger

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Run(ctx)

	var m prometheus.Metric
	var ch chan prometheus.Metric

	// wait for 4 packets to arrive
	assert.Eventually(t, func() bool {
		ch = make(chan prometheus.Metric)
		go p.Collect(ch)
		m = <-ch
		return tools.MetricName(m) == "pinger_packet_count" && tools.MetricValue(m).GetGauge().GetValue() == 4
	}, 500*time.Millisecond, 10*time.Millisecond)

	m = <-ch
	assert.Equal(t, "pinger_packet_loss_count", tools.MetricName(m))
	assert.Equal(t, 1.0, tools.MetricValue(m).GetGauge().GetValue())
	m = <-ch
	assert.Equal(t, "pinger_latency_seconds", tools.MetricName(m))
	assert.Equal(t, 4e-05, tools.MetricValue(m).GetGauge().GetValue())
}

// fakePinger sends packets rapidly, so we don't have to wait 5 seconds to get some meaningful data
func fakePinger(ch chan pinger.Response, host ...string) (err error) {
	ch <- pinger.Response{Host: host[0], SequenceNr: 0, Latency: 10 * time.Microsecond}
	ch <- pinger.Response{Host: host[0], SequenceNr: 1, Latency: 10 * time.Microsecond}
	ch <- pinger.Response{Host: host[0], SequenceNr: 3, Latency: 10 * time.Microsecond}
	ch <- pinger.Response{Host: host[0], SequenceNr: 4, Latency: 10 * time.Microsecond}
	return
}

func TestPinger_Run(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	p := collector.New([]string{"127.0.0.1"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Run(ctx)

	var m prometheus.Metric
	var ch chan prometheus.Metric

	// wait for 1 packet to arrive
	assert.Eventually(t, func() bool {
		ch = make(chan prometheus.Metric)
		go p.Collect(ch)
		m = <-ch
		return tools.MetricName(m) == "pinger_packet_count" && tools.MetricValue(m).GetGauge().GetValue() > 0
	}, 5*time.Second, 10*time.Millisecond)

	m = <-ch
	assert.Equal(t, "pinger_packet_loss_count", tools.MetricName(m))
	m = <-ch
	assert.Equal(t, "pinger_latency_seconds", tools.MetricName(m))
	assert.NotZero(t, tools.MetricValue(m).GetGauge().GetValue())

}
