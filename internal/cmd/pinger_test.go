package cmd

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"
)

// var debugLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestPinger(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping ICMP test in GitHub Actions")
	}

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
		assert.NoError(t, run(ctx, &Cmd, []string{"::1"}, viper.GetViper(), r, discardLogger))
	}()

	//env := strings.Join(os.Environ(), ", ")
	//debugLogger.Debug("env dump", "env", env)
	assert.Eventually(t, func() bool {
		count, err := testutil.GatherAndCount(r, "pinger_packets_received_count")
		return err == nil && count > 0
	}, 10*time.Second, 500*time.Millisecond)
}
