package cmd

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPinger(t *testing.T) {

	Cmd.SetArgs([]string{"--ipv4=false", "localhost"})
	go func() {
		_ = Cmd.Execute()
	}()

	assert.Eventually(t, func() bool {
		count, err := testutil.GatherAndCount(prometheus.DefaultGatherer, "pinger_packet_count")
		return err == nil && count > 0
	}, time.Minute, 5*time.Second)
}
