package pinger

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"pinger/internal/pingtracker"
)

func TestPinger(t *testing.T) {

	hosts := []string{"127.0.0.1"}
	count, loss, latency := RunNTimes(hosts, 1*time.Second, 5, testPinger)

	assert.Greater(t, count, 9)
	assert.Equal(t, 1, loss)
	assert.Equal(t, time.Duration(int64(count*50*1000000)), latency)
}

func testPinger(_ string, tracker *pingtracker.PingTracker) {
	seqNo := 1
	for {
		tracker.Track(seqNo, 50*time.Millisecond)
		seqNo++
		time.Sleep(500 * time.Millisecond)
	}
}
