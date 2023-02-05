package main

import (
	"context"
	"github.com/clambin/pinger/collector"
	"github.com/clambin/pinger/configuration"
	"github.com/clambin/pinger/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xonvanetta/shutdown/pkg/shutdown"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
)

var (
	configFilename string
	cmd            = cobra.Command{
		Use:   "pinger",
		Short: "Pings a set of hosts and exports latency & packet loss as Prometheus metrics",
		Run:   Main,
	}
)

func main() {
	if err := cmd.Execute(); err != nil {
		slog.Error("failed to start", err)
		os.Exit(1)
	}
}

func Main(_ *cobra.Command, args []string) {
	var opts slog.HandlerOptions
	if viper.GetBool("debug") {
		opts.Level = slog.LevelDebug
		//opts.AddSource = true
	}
	slog.SetDefault(slog.New(opts.NewTextHandler(os.Stderr)))

	var cfg configuration.Configuration
	if err := viper.Unmarshal(&cfg); err != nil {
		slog.Error("failed to read configuration", err)
		return
	}

	targets := configuration.GetTargets(viper.GetViper(), args)
	slog.Info("pinger started", "targets", targets, "version", version.BuildVersion)

	p := collector.New(targets)
	prometheus.MustRegister(p)
	go p.Run(context.Background())

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err2 := http.ListenAndServe(cfg.Addr, nil); err2 != http.ErrServerClosed {
			slog.Error("failed to start http server", err2)
		}
	}()

	<-shutdown.Chan()

	slog.Info("collector stopped")
}

func init() {
	cobra.OnInitialize(initConfig)
	cmd.Version = version.BuildVersion
	cmd.Flags().StringVar(&configFilename, "config", "", "Configuration file")
	cmd.Flags().Bool("debug", false, "Log debug messages")
	_ = viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))
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
	viper.SetDefault("addr", ":9090")

	viper.SetEnvPrefix("PINGER")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("failed to read config file", "error", err)
	}
}
