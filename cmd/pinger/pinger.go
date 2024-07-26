package main

import (
	"context"
	"errors"
	"github.com/clambin/go-common/charmer"
	"github.com/clambin/pinger/internal/collector"
	"github.com/clambin/pinger/internal/configuration"
	"github.com/clambin/pinger/internal/pinger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
)

var (
	version        = "change-me"
	configFilename string
	cmd            = cobra.Command{
		Use:     "pinger [flags] [ <host> ... ]",
		Short:   "Pings a set of hosts and exports latency & packet loss as Prometheus metrics",
		Run:     Main,
		Version: version,
		PreRun: func(cmd *cobra.Command, args []string) {
			charmer.SetTextLogger(cmd, viper.GetBool("debug"))
		},
	}
)

func main() {
	if err := cmd.Execute(); err != nil {
		slog.Error("failed to start", err)
		os.Exit(1)
	}
}

func Main(cmd *cobra.Command, args []string) {
	l := charmer.GetLogger(cmd)
	targets := configuration.GetTargets(viper.GetViper(), args)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	l.Info("pinger started", "targets", targets, "version", cmd.Version)

	trackers := pinger.NewMultiPinger(targets, l.With("module", "multipinger"))
	go func() {
		if err := trackers.Run(ctx); err != nil {
			panic(err)
		}
	}()

	p := collector.Collector{
		Trackers: trackers,
	}
	prometheus.MustRegister(p)

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(viper.GetString("addr"), nil); !errors.Is(err, http.ErrServerClosed) {
			l.Error("failed to start http server", err)
		}
	}()

	<-ctx.Done()

	l.Info("collector stopped")
}

func init() {
	cobra.OnInitialize(initConfig)
	cmd.Flags().StringVar(&configFilename, "config", "", "Configuration file")
	cmd.Flags().Bool("debug", false, "Log debug messages")
	_ = viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))
	cmd.Flags().String("addr", ":8080", "Metrics listener address")
	_ = viper.BindPFlag("addr", cmd.Flags().Lookup("addr"))
}

func initConfig() {
	if configFilename != "" {
		viper.SetConfigFile(configFilename)
	} else {
		viper.AddConfigPath("/etc/pinger/")
		viper.AddConfigPath("$HOME/.pinger")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.SetDefault("debug", false)
	viper.SetDefault("addr", ":8080")

	viper.SetEnvPrefix("PINGER")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("failed to read config file", "error", err)
	}
}
