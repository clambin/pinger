package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"pinger/internal/pinger"
	"pinger/internal/version"
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
	a.Flag("interval", "Measurement interval").Default("5s").DurationVar(&cfg.interval)
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

	log.Infof("pinger %s - hosts: %s", version.BuildVersion, *hosts)

	go pinger.Run(*hosts, cfg.interval)

	// Run initialized & runs the metrics
	listenAddress := fmt.Sprintf(":%d", cfg.port)
	http.Handle(cfg.endpoint, promhttp.Handler())
	_ = http.ListenAndServe(listenAddress, nil)
}
