package main

import (
	"context"
	"fmt"
	"github.com/clambin/pinger/pinger"
	"github.com/clambin/pinger/version"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	cfg := struct {
		port     int
		endpoint string
		debug    bool
		interval time.Duration
	}{}
	a := kingpin.New(filepath.Base(os.Args[0]), "pinger")

	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("port", "Metrics listener port").Default("8080").IntVar(&cfg.port)
	a.Flag("endpoint", "Metrics listener endpoint").Default("/metrics").StringVar(&cfg.endpoint)
	a.Flag("debug", "Log debug messages").BoolVar(&cfg.debug)
	// a.Flag("interval", "Measurement interval").Default("5s").DurationVar(&cfg.interval)
	hosts := a.Arg("hosts", "hosts to ping").Strings()

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	if cfg.debug {
		log.SetLevel(log.DebugLevel)
	}

	if value, ok := os.LookupEnv("HOSTS"); ok == true {
		values := strings.Fields(value)
		hosts = &values
	}

	log.WithField("hosts", *hosts).Infof("pinger %s", version.BuildVersion)

	p := pinger.New(*hosts)
	prometheus.MustRegister(p)

	go p.Run(context.Background())

	// Run the metrics server
	listenAddress := fmt.Sprintf(":%d", cfg.port)
	r := mux.NewRouter()
	r.Use(prometheusMiddleware)
	r.Path(cfg.endpoint).Handler(promhttp.Handler())
	err = http.ListenAndServe(listenAddress, r)

	log.WithError(err).Error("failed to start http server")
}

// Prometheus metrics
var (
	httpDuration = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of HTTP requests",
	}, []string{"path"})
)

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
	})
}
