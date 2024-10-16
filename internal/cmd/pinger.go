package cmd

import (
	"context"
	"errors"
	"github.com/clambin/go-common/charmer"
	"github.com/clambin/pinger/internal/collector"
	"github.com/clambin/pinger/internal/configuration"
	"github.com/clambin/pinger/internal/pinger"
	"github.com/clambin/pinger/pkg/ping/icmp"
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
	Cmd = cobra.Command{
		Use:     "pinger [flags] [ <host> ... ]",
		Short:   "Pings a set of hosts and exports latency & packet loss as Prometheus metrics",
		RunE:    Main,
		Version: "change-me",
		PreRun: func(cmd *cobra.Command, args []string) {
			charmer.SetTextLogger(cmd, viper.GetBool("debug"))
		},
	}
)

func Main(cmd *cobra.Command, args []string) error {
	l := charmer.GetLogger(cmd)
	targets := configuration.GetTargets(viper.GetViper(), args)
	var tp icmp.Transport
	if viper.GetBool("ipv4") {
		tp |= icmp.IPv4
	}
	if viper.GetBool("ipv6") {
		tp |= icmp.IPv6
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	l.Info("pinger started", "targets", targets, "version", cmd.Version)

	trackers := pinger.New(targets, tp, l)
	done := make(chan struct{})
	go func() {
		trackers.Run(ctx)
		done <- struct{}{}
	}()

	p := collector.Collector{
		Pinger: trackers,
		Logger: l,
	}
	prometheus.MustRegister(p)

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(viper.GetString("addr"), nil); !errors.Is(err, http.ErrServerClosed) {
			l.Error("failed to start http server", "err", err)
		}
	}()

	defer l.Info("collector stopped")
	<-done
	return nil
}

var arguments = charmer.Arguments{
	"config": {Default: "", Help: "Configuration file"},
	"debug":  {Default: false, Help: "log debug messages"},
	"addr":   {Default: ":8080", Help: "Prometheus listener address"},
	"ipv4":   {Default: true, Help: "ping ipv4 address"},
	"ipv6":   {Default: true, Help: "ping ipv6 address"},
}

func init() {
	cobra.OnInitialize(initConfig)
	if err := charmer.SetPersistentFlags(&Cmd, viper.GetViper(), arguments); err != nil {
		slog.Warn("failed to set flags", "err", err)
	}
}

func initConfig() {
	if configFilename := viper.GetString("config"); configFilename != "" {
		viper.SetConfigFile(configFilename)
	} else {
		viper.AddConfigPath("/etc/pinger/")
		viper.AddConfigPath("$HOME/.pinger")
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}
	viper.SetEnvPrefix("PINGER")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("failed to read config file", "error", err)
	}
}
