package main

import (
	"context"
	"github.com/clambin/go-metrics/server"
	"github.com/clambin/pinger/collector"
	"github.com/clambin/pinger/version"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/xonvanetta/shutdown/pkg/shutdown"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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

	s := server.New(cfg.port)
	go func() {
		err2 := s.Run()
		if err2 != http.ErrServerClosed {
			log.WithError(err2).Error("failed to start http server")
		}
	}()

	<-shutdown.Chan()

	err = s.Shutdown(5 * time.Second)
	if err != nil {
		log.WithError(err).Error("failed to shut down http server")
	}

	log.Info("collector stopped")
}
