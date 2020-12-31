package metrics

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	log "github.com/sirupsen/logrus"
)

var (
	packetsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pinger_packet_count",
		Help: "Pinger total packet count",
	},
		[]string{"host"})

	lossCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pinger_packet_loss_count",
		Help: "Pinger total measured packet loss",
	},
		[]string{"host"})

	// TODO: better as a Gauge?
	latencyCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pinger_latency_seconds",
		Help: "Pinger total measured packet loss",
	},
		[]string{"host"})
)

// Init initializes the prometheus metrics server
func Init(endpoint string, port int) {
	http.Handle(endpoint, promhttp.Handler())
	listenAddress := fmt.Sprintf(":%d", port)
	go func(listenAddr string) {
		_ = http.ListenAndServe(listenAddress, nil)
	}(listenAddress)
}

// Measure reports the metrics against their respective Counter
func Measure(host string, packets int, loss int, latency time.Duration) {
	packetsCounter.WithLabelValues(host).Add(float64(packets))
	lossCounter.WithLabelValues(host).Add(float64(loss))
	latencyCounter.WithLabelValues(host).Add(latency.Seconds())

}

// LoadValue gets the last value reported so unit tests can verify the correct value was reported
func LoadValue(metric string, labels ...string) (float64, error) {
	var m dto.Metric

	log.Debugf("%s(%s)", metric, labels)
	switch metric {
	case "pinger_packet_count":
		_ = packetsCounter.WithLabelValues(labels...).Write(&m)
	case "pinger_packet_loss_count":
		_ = lossCounter.WithLabelValues(labels...).Write(&m)
	case "pinger_latency_seconds":
		_ = latencyCounter.WithLabelValues(labels...).Write(&m)
	default:
		return 0, errors.New("could not find " + metric)
	}

	return m.Counter.GetValue(), nil
}
