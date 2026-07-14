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
	outstanding map[ping.SequenceNumber]time.Time
	Name        string
	Host        string
	addr        net.IP
	latencies   []time.Duration
	Sent        int
	Received    int
	lock        sync.Mutex
}

func (t *Target) markRequest(seq ping.SequenceNumber) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.Sent++
	if t.outstanding == nil {
		t.outstanding = make(map[ping.SequenceNumber]time.Time)
	}
	t.outstanding[seq] = time.Now()
}

func (t *Target) markResponse(response ping.Response) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if _, ok := t.outstanding[response.Request.Seq]; ok {
		delete(t.outstanding, response.Request.Seq)
		t.Received++
		t.latencies = append(t.latencies, response.Latency)
	}
}

func (t *Target) statistics() Statistics {
	t.lock.Lock()
	defer t.lock.Unlock()
	// calculate statistics
	statistics := Statistics{
		Sent:     t.Sent,
		Received: t.Received,
		Latency:  t.medianLatency(),
	}
	// keep up to 10 outstanding requests (i.e., 10 seconds; probably way too much)
	for seq, sent := range t.outstanding {
		if time.Since(sent) > 10*time.Second {
			delete(t.outstanding, seq)
		}
	}
	// reset counters
	t.Sent = len(t.outstanding) // keep track of outstanding requests
	t.Received = 0
	t.latencies = t.latencies[:0]
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
