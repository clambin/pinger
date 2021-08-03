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

func TestPinger_Run(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	p := pinger.New([]string{"127.0.0.1"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Run(ctx)

	time.Sleep(5 * time.Second)

	ch := make(chan prometheus.Metric)
	go p.Collect(ch)

	m := <-ch
	assert.Equal(t, "pinger_packet_count", metricName(m))
	assert.GreaterOrEqual(t, getMetric(m).GetGauge().GetValue(), 4.0)
	m = <-ch
	assert.Equal(t, "pinger_packet_loss_count", metricName(m))
	assert.LessOrEqual(t, getMetric(m).GetGauge().GetValue(), 3.0)
	m = <-ch
	assert.Equal(t, "pinger_latency_seconds", metricName(m))
	assert.Greater(t, getMetric(m).GetGauge().GetValue(), 0.00001)
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
