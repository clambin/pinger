package cmd

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"testing"
	"time"
)

// var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
var debugLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

func TestPinger(t *testing.T) {
	r := prometheus.NewPedanticRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	v := viper.New()
	for key, value := range viper.GetViper().AllSettings() {
		v.Set(key, value)
	}
	// only run ipv6 to not clash w/ ipv4 test in internal/pinger
	v.Set("ipv4", false)

	go func() {
		assert.NoError(t, run(ctx, &Cmd, []string{"127.0.0.1"}, viper.GetViper(), r, debugLogger))
	}()

	assert.Eventually(t, func() bool {
		count, err := testutil.GatherAndCount(r, "pinger_packets_received_count")
		return err == nil && count > 0
	}, time.Minute, time.Second)
}
