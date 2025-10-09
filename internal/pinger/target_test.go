package pinger

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/clambin/pinger/ping"
	"github.com/stretchr/testify/assert"
)

func TestTargets_LogValue(t *testing.T) {
	targets := Targets{
		{Name: "localhost", Host: "127.0.0.1"},
		{Name: "example.com", Host: "www.example.com"},
	}
	assert.Equal(t, "localhost,example.com", targets.LogValue().String())
}

func TestTarget_Statistics(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		target := Target{Name: "localhost", Host: "127.0.0.1"}
		target.markRequest(10)
		target.markRequest(11)
		target.markRequest(12)
		target.markRequest(13)
		target.markResponse(ping.Response{Latency: 90 * time.Millisecond, Request: ping.Request{Seq: 10}})
		target.markResponse(ping.Response{Latency: 110 * time.Millisecond, Request: ping.Request{Seq: 13}})

		// two packets are still outstanding
		statistics := target.statistics()
		assert.Equal(t, Statistics{Sent: 4, Received: 2, Latency: 100 * time.Millisecond}, statistics)

		// nothing received: still two packages outstanding
		statistics = target.statistics()
		assert.Equal(t, Statistics{Sent: 2, Received: 0, Latency: 0}, statistics)

		// one packet comes in
		target.markResponse(ping.Response{Latency: 100 * time.Millisecond, Request: ping.Request{Seq: 11}})
		statistics = target.statistics()
		assert.Equal(t, Statistics{Sent: 2, Received: 1, Latency: 100 * time.Millisecond}, statistics)

		// wait for the last outstanding packet to timeout. should be reported as a loss.
		time.Sleep(time.Minute)
		statistics = target.statistics()
		assert.Equal(t, Statistics{Sent: 1, Received: 0, Latency: 0}, statistics)

		// from now on, no loss should be detected
		statistics = target.statistics()
		assert.Equal(t, Statistics{Sent: 0, Received: 0, Latency: 0}, statistics)

		// off we go again
		target.markRequest(14)
		target.markRequest(15)
		target.markResponse(ping.Response{Latency: 100 * time.Millisecond, Request: ping.Request{Seq: 14}})
		statistics = target.statistics()
		assert.Equal(t, Statistics{Sent: 2, Received: 1, Latency: 100 * time.Millisecond}, statistics)

	})
}
