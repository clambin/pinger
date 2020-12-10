package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

func Init(port int) {
	http.Handle("/metrics", promhttp.Handler())
	listenAddress := fmt.Sprintf(":%d", port)
	go func(listenAddr string) {
		err := http.ListenAndServe(listenAddress, nil)
		fmt.Println(err)
		if err != nil {
			panic(err)
		}
	}(listenAddress)
}

func Measure(host string, packets int, loss int, latency time.Duration) {
	packetsCounter.WithLabelValues(host).Add(float64(packets))
	lossCounter.WithLabelValues(host).Add(float64(packets))
	latencyCounter.WithLabelValues(host).Add(latency.Seconds())

}
