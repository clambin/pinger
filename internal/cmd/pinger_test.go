package cmd

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestPinger(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping ICMP test in GitHub Actions")
	}

	r := prometheus.NewPedanticRegistry()

	v := viper.New()
	for key, value := range viper.GetViper().AllSettings() {
		v.Set(key, value)
	}
	// only run ipv6 to not clash w/ ipv4 test in internal/pinger
	v.Set("ipv4", false)

	go func() {
		assert.NoError(t, run(t.Context(), &Cmd, []string{"::1"}, viper.GetViper(), r, slog.New(slog.DiscardHandler)))
	}()

	//env := strings.Join(os.Environ(), ", ")
	//debugLogger.Debug("env dump", "env", env)
	assert.Eventually(t, func() bool {
		count, err := testutil.GatherAndCount(r, "pinger_packets_received_count")
		return err == nil && count > 0
	}, 10*time.Second, 500*time.Millisecond)
}
