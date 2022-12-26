package main

import (
	"context"
	"fmt"
	"github.com/clambin/pinger/collector"
	"github.com/clambin/pinger/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/xonvanetta/shutdown/pkg/shutdown"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	cfg := struct {
		port  int
		debug bool
	}{}
	a := kingpin.New(filepath.Base(os.Args[0]), "collector")

	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("port", "Metrics listener port").Default("8080").IntVar(&cfg.port)
	a.Flag("debug", "Log debug messages").BoolVar(&cfg.debug)
	hosts := a.Arg("hosts", "hosts to collector").Strings()

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	if cfg.debug {
		log.SetLevel(log.DebugLevel)
	}

	if value, ok := os.LookupEnv("HOSTS"); ok {
		values := strings.Fields(value)
		hosts = &values
	}

	log.WithFields(log.Fields{
		"hosts":   *hosts,
		"version": version.BuildVersion,
	}).Info("collector started")

	p := collector.New(*hosts)
	prometheus.MustRegister(p)
	go p.Run(context.Background())

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err2 := http.ListenAndServe(fmt.Sprintf(":%d", cfg.port), nil); err2 != http.ErrServerClosed {
			log.WithError(err2).Error("failed to start http server")
		}
	}()

	<-shutdown.Chan()

	log.Info("collector stopped")
}
