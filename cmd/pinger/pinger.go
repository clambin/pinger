package main

import (
	"context"
	"fmt"
	"github.com/clambin/pinger/collector"
	"github.com/clambin/pinger/configuration"
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
	}
)

func main() {
	if err := cmd.Execute(); err != nil {
		slog.Error("failed to start", err)
		os.Exit(1)
	}
}

func Main(cmd *cobra.Command, args []string) {
	var opts slog.HandlerOptions
	if viper.GetBool("debug") {
		opts.Level = slog.LevelDebug
		//opts.AddSource = true
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &opts)))

	targets := configuration.GetTargets(viper.GetViper(), args)
	slog.Info("pinger started", "targets", targets, "version", cmd.Version)

	p := collector.New(targets)
	prometheus.MustRegister(p)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go p.Run(ctx)

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		var addr string
		if port := viper.GetInt("port"); port > 0 {
			addr = fmt.Sprintf(":%d", port)
		} else {
			addr = viper.GetString("addr")
		}
		if err2 := http.ListenAndServe(addr, nil); err2 != http.ErrServerClosed {
			slog.Error("failed to start http server", err2)
		}
	}()

	<-ctx.Done()

	slog.Info("collector stopped")
}

func init() {
	cobra.OnInitialize(initConfig)
	cmd.Flags().StringVar(&configFilename, "config", "", "Configuration file")
	cmd.Flags().Bool("debug", false, "Log debug messages")
	_ = viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))
	cmd.Flags().Int("port", 0, "Metrics listener port (obsolete)")
	_ = viper.BindPFlag("port", cmd.Flags().Lookup("port"))
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
