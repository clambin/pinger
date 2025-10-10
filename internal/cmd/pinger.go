package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"codeberg.org/clambin/go-common/charmer"
	"codeberg.org/clambin/go-common/httputils"
	"github.com/clambin/pinger/internal/collector"
	"github.com/clambin/pinger/internal/configuration"
	"github.com/clambin/pinger/internal/pinger"
	"github.com/clambin/pinger/ping"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	Cmd = cobra.Command{
		Use:   "pinger [flags] [ <host> ... ]",
		Short: "Pings a set of hosts and exports latency & packet loss as Prometheus metrics",
		PreRun: func(cmd *cobra.Command, args []string) {
			charmer.SetTextLogger(cmd, viper.GetBool("debug"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			l := charmer.GetLogger(cmd)
			v := viper.GetViper()
			r := prometheus.DefaultRegisterer
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return run(ctx, cmd, args, v, r, l)
		},
	}

	arguments = charmer.Arguments{
		"config":    {Default: "", Help: "Configuration file"},
		"debug":     {Default: false, Help: "log debug messages"},
		"addr":      {Default: ":8080", Help: "Prometheus listener address"},
		"ipv4":      {Default: true, Help: "ping ipv4 address"},
		"ipv6":      {Default: true, Help: "ping ipv6 address"},
		"ignore-id": {Default: false, Help: "ignore ICMP MsgID (use this when running inside a container)"},
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	if err := charmer.SetFlags(&Cmd, viper.GetViper(), arguments); err != nil {
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

func run(ctx context.Context, cmd *cobra.Command, args []string, v *viper.Viper, r prometheus.Registerer, l *slog.Logger) error {
	targets := configuration.GetTargets(v, args)

	socketOptions := []ping.SocketOption{
		ping.WithLogger(l.With("component", "socket")),
	}
	if v.GetBool("ipv4") {
		socketOptions = append(socketOptions, ping.WithIPv4())
	}
	if v.GetBool("ipv6") {
		socketOptions = append(socketOptions, ping.WithIPv6())
	}
	if v.GetBool("ignore-id") {
		socketOptions = append(socketOptions, ping.WithoutCheckID())
	}

	l.Info("pinger started", "targets", targets, "version", cmd.Version)

	s, err := ping.New(socketOptions...)
	if err != nil {
		return fmt.Errorf("failed to create icmp socket: %w", err)
	}

	targetPinger := pinger.New(targets, s, l)
	p := collector.Collector{
		Targets: targets,
		Logger:  l,
	}
	r.MustRegister(p)

	var wg sync.WaitGroup
	wg.Go(func() {
		m := http.NewServeMux()
		m.Handle("/metrics", promhttp.Handler())
		promServer := http.Server{
			Addr:    v.GetString("addr"),
			Handler: m,
		}
		if err := httputils.RunServer(ctx, &promServer); !errors.Is(err, http.ErrServerClosed) {
			l.Error("failed to start http server", "err", err)
		}
	})
	wg.Go(func() {
		targetPinger.Run(ctx)
	})

	defer l.Info("collector stopped")
	wg.Wait()
	return nil
}
