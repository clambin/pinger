package pinger

import (
	"cmp"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/clambin/pinger/ping"
)

type Statistics struct {
	Sent     int
	Received int
	Latency  time.Duration
}

var _ slog.LogValuer = Targets{}

type Targets []*Target

func (t Targets) LogValue() slog.Value {
	values := make([]string, len(t))
	for i, target := range t {
		values[i] = cmp.Or(target.Name, target.addr.String())
	}
	return slog.StringValue(strings.Join(values, ","))
}

func (t Targets) Statistics() map[string]Statistics {
	stats := make(map[string]Statistics, len(t))
	for _, target := range t {
		stats[target.Name] = target.statistics()
	}
	return stats
}

type Target struct {
	Name        string
	Host        string
	addr        net.IP
	Sent        int
	Received    int
	outstanding []ping.SequenceNumber
	latencies   []time.Duration
	lock        sync.RWMutex
}

func (t *Target) markRequests(seq ping.SequenceNumber) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.Sent++
	t.outstanding = append(t.outstanding, seq)
}

func (t *Target) markResponse(response ping.Response) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for i, seq := range t.outstanding {
		if response.Request.Seq == seq {
			t.Received++
			t.latencies = append(t.latencies, response.Latency)
			t.outstanding = append(t.outstanding[:i], t.outstanding[i+1:]...)
			return
		}
	}
}

func (t *Target) statistics() Statistics {
	t.lock.Lock()
	defer t.lock.Unlock()
	statistics := Statistics{
		Sent:     t.Sent,
		Received: t.Received,
		Latency:  t.medianLatency(),
	}
	t.latencies = t.latencies[:0]
	// keep up to 10 outstanding packets (i.e., 10 seconds; probably way too much)
	if n := len(t.outstanding); n > 30 {
		t.outstanding = t.outstanding[n-30:]
	}
	return statistics
}

func (t *Target) medianLatency() time.Duration {
	if len(t.latencies) == 0 {
		return 0
	}
	slices.Sort(t.latencies)
	if len(t.latencies)%2 == 0 {
		return (t.latencies[len(t.latencies)/2-1] + t.latencies[len(t.latencies)/2]) / 2
	}
	return t.latencies[len(t.latencies)/2]
}
