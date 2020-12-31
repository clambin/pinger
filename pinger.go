package main

import (
	"os"
	"path/filepath"
	"runtime/pprof"
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
		endpoint string
		debug    bool
		interval time.Duration
		profile  string
	}{}
	a := kingpin.New(filepath.Base(os.Args[0]), "pinger")

	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("port", "Metrics listener port").Default("8080").IntVar(&cfg.port)
	a.Flag("endpoint", "Metrics listener endpoint").Default("/metrics").StringVar(&cfg.endpoint)
	a.Flag("debug", "Log debug messages").BoolVar(&cfg.debug)
	a.Flag("interval", "Measurement interval").Default("5s").DurationVar(&cfg.interval)
	// a.Flag("profile", "CPU profiler filename").StringVar(&cfg.profile)
	hosts := a.Arg("hosts", "hosts to ping").Strings()

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	if cfg.debug {
		log.SetLevel(log.DebugLevel)
	}

	metrics.Init(cfg.endpoint, cfg.port)

	if value, ok := os.LookupEnv("HOSTS"); ok == true {
		values := strings.Fields(value)
		hosts = &values
	}

	log.Infof("pinger %s - hosts: %s", version.BuildVersion, *hosts)

	if cfg.profile == "" {
		pinger.Run(*hosts, cfg.interval)
	} else {
		f, err := os.Create(cfg.profile)
		if err != nil {
			panic(err)
		}
		_ = pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

		pinger.RunNTimes(*hosts, cfg.interval, 10)
	}
}
