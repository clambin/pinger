package pinger

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"pinger/internal/pingtracker"
)

func TestStubbedPinger(t *testing.T) {

	hosts := []string{"127.0.0.1"}
	count, loss, latency := RunNTimes(hosts, 1*time.Second, 5, stubbedPinger)

	assert.Greater(t, count, 9)
	assert.Equal(t, 1, loss)
	assert.Equal(t, time.Duration(int64(count*50*1000000)), latency)
}

func stubbedPinger(_ string, tracker *pingtracker.PingTracker) {
	seqNo := 1
	for {
		tracker.Track(seqNo, 50*time.Millisecond)
		seqNo++
		time.Sleep(500 * time.Millisecond)
	}
}

func TestSpawnedPinger(t *testing.T) {
	hosts := []string{"127.0.0.1"}
	count, loss, latency := RunNTimes(hosts, 4*time.Second, 2, SpawnedPinger)

	assert.Greater(t, count, 4)
	assert.Equal(t, 0, loss)
	assert.Greater(t, latency.Nanoseconds(), int64(0))
}
