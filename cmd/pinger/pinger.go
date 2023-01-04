package main

import (
	"context"
	"fmt"
	"github.com/clambin/pinger/collector"
	"github.com/clambin/pinger/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xonvanetta/shutdown/pkg/shutdown"
	"golang.org/x/exp/slog"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var cfg struct {
		port  int
		debug bool
		hosts []string
	}

	a := kingpin.New(filepath.Base(os.Args[0]), "collector")
	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("port", "Metrics listener port").Default("8080").IntVar(&cfg.port)
	a.Flag("debug", "Log debug messages").BoolVar(&cfg.debug)
	a.Arg("hosts", "hosts to collector").StringsVar(&cfg.hosts)

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	var opts slog.HandlerOptions
	if cfg.debug {
		opts.Level = slog.LevelDebug
		opts.AddSource = true
	}
	slog.SetDefault(slog.New(opts.NewTextHandler(os.Stdout)))

	if value, ok := os.LookupEnv("HOSTS"); ok {
		cfg.hosts = strings.Fields(value)
	}

	slog.Info("collector started", "hosts", cfg.hosts, "version", version.BuildVersion)

	p := collector.New(cfg.hosts)
	prometheus.MustRegister(p)
	go p.Run(context.Background())

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err2 := http.ListenAndServe(fmt.Sprintf(":%d", cfg.port), nil); err2 != http.ErrServerClosed {
			slog.Error("failed to start http server", err2)
		}
	}()

	<-shutdown.Chan()

	slog.Info("collector stopped")
}
