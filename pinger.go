package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"pinger/internal/metrics"
	"pinger/internal/pinger"
	"pinger/internal/version"
)

func main() {
	cfg := struct {
		port     int
		debug    bool
		interval string
	}{}
	a := kingpin.New(filepath.Base(os.Args[0]), "pinger")

	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("port", "Metrics listener port").Default("8080").IntVar(&cfg.port)
	a.Flag("debug", "Log debug messages").BoolVar(&cfg.debug)
	a.Flag("interval", "Interval (e.g. \"5s\"").Default("5s").StringVar(&cfg.interval)
	hosts := a.Arg("hosts", "hosts to ping").Strings()

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	if cfg.debug {
		log.SetLevel(log.DebugLevel)
	}

	metrics.Init(cfg.port)

	if value, ok := os.LookupEnv("HOSTS"); ok == true {
		values := strings.Fields(value)
		hosts = &values
	}

	log.Infof("pinger %s - hosts: %s", version.BuildVersion, *hosts)

	duration, err := time.ParseDuration(cfg.interval)

	if err != nil {
		log.Warningf("Could not parse interval '%s'. Defaulting to 5s", cfg.interval)
		duration = 5 * time.Second
	}

	pinger.Run(*hosts, duration)
}
