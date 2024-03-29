package tracker

import (
	"github.com/clambin/go-common/set"
	"sync"
	"time"
)

// Tracker keeps track of received ICMP replies and calculates packet loss & average latency
type Tracker struct {
	NextSeqNr int
	seqNrs    []int
	latencies []time.Duration
	lock      sync.Mutex
}

// New creates a new Tracker
func New() *Tracker {
	return &Tracker{}
}

// Track measured sequence number and latency
func (t *Tracker) Track(SeqNr int, Latency time.Duration) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.seqNrs = append(t.seqNrs, SeqNr)
	t.latencies = append(t.latencies, Latency)
}

// Calculate packet loss and latency for all data stored in Tracker
func (t *Tracker) Calculate() (int, int, time.Duration) {
	t.lock.Lock()
	defer t.lock.Unlock()

	loss := t.calculateLoss()
	count := len(t.latencies)
	latency := t.calculateLatency()

	// empty the slice but keep the memory
	t.seqNrs = t.seqNrs[:0]
	t.latencies = t.latencies[:0]

	return count, loss, latency
}

func (t *Tracker) calculateLatency() time.Duration {
	var total time.Duration
	for _, entry := range t.latencies {
		total += entry
	}
	return total
}

func (t *Tracker) calculateLoss() int {
	//if len(t.seqNrs) == 0 {
	//	return 0
	//}
	// Sort all sequence numbers and remove duplicates
	t.seqNrs = set.New(t.seqNrs...).ListOrdered()

	// sequence numbers can roll over!
	// In this case, we'd get something like [ 0, 1, 2, 3, 65534, 65535 ]
	// Split into two lists [ 65534, 65535 ] and [ 0, 1, 2 ] using nextSeqNr as a boundary
	// Process the higher list first (pre-rollover) and then the lower one (post-rollover)

	count := len(t.seqNrs)
	index := 0
	for index < count && t.seqNrs[index] < t.NextSeqNr-60 {
		index++
	}

	// pre-rollover / no rollover
	var gap int
	if index < count {
		gap = t.processRange(t.seqNrs[index:])
	}

	if index > 0 {
		t.NextSeqNr = 0
		gap += t.processRange(t.seqNrs[:index])
	}

	return gap
}

func (t *Tracker) processRange(sequence []int) int {
	count := len(sequence)
	if count == 0 {
		panic("processRange: sequence range should not be empty")
	}

	index := 0
	// skip older packets
	for ; index < count && sequence[index] < t.NextSeqNr; index++ {
	}

	var gap int
	for ; index < count; index++ {
		gap += sequence[index] - t.NextSeqNr
		t.NextSeqNr = sequence[index] + 1
	}

	return gap
}
