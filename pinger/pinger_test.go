package pinger_test

import (
	"context"
	"github.com/clambin/pinger/pinger"
	"github.com/prometheus/client_golang/prometheus"
	pcg "github.com/prometheus/client_model/go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
	"time"
)

func TestPinger_Run_Quick(t *testing.T) {
	p := pinger.New([]string{"127.0.0.1"})
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
		return metricName(m) == "pinger_packet_count" && getMetric(m).GetGauge().GetValue() == 4
	}, 500*time.Millisecond, 10*time.Millisecond)

	m = <-ch
	assert.Equal(t, "pinger_packet_loss_count", metricName(m))
	assert.Equal(t, 1.0, getMetric(m).GetGauge().GetValue())
	m = <-ch
	assert.Equal(t, "pinger_latency_seconds", metricName(m))
	assert.Equal(t, 4e-05, getMetric(m).GetGauge().GetValue())
}

// fakePinger sends packets rapidly, so we don't have to wait 5 seconds to get some meaningful data
func fakePinger(host string, ch chan pinger.PingResponse) (err error) {
	ch <- pinger.PingResponse{Host: host, SequenceNr: 0, Latency: 10 * time.Microsecond}
	ch <- pinger.PingResponse{Host: host, SequenceNr: 1, Latency: 10 * time.Microsecond}
	ch <- pinger.PingResponse{Host: host, SequenceNr: 3, Latency: 10 * time.Microsecond}
	ch <- pinger.PingResponse{Host: host, SequenceNr: 4, Latency: 10 * time.Microsecond}
	return
}

func TestPinger_Run(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	p := pinger.New([]string{"127.0.0.1"})

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
		return metricName(m) == "pinger_packet_count" && getMetric(m).GetGauge().GetValue() > 0
	}, 5*time.Second, 10*time.Millisecond)

	m = <-ch
	assert.Equal(t, "pinger_packet_loss_count", metricName(m))
	m = <-ch
	assert.Equal(t, "pinger_latency_seconds", metricName(m))
	assert.NotZero(t, getMetric(m).GetGauge().GetValue())

}

// metricName returns the metric name
func metricName(metric prometheus.Metric) string {
	desc := metric.Desc().String()

	r := regexp.MustCompile(`fqName: "([a-z,_]+)"`)
	match := r.FindStringSubmatch(desc)

	if len(match) < 2 {
		return ""
	}

	return match[1]
}

// getMetric returns the matric so we can get its value
func getMetric(metric prometheus.Metric) *pcg.Metric {
	m := new(pcg.Metric)
	if metric.Write(m) != nil {
		panic("failed to parse metric")
	}

	return m
}
