package pingtracker

import (
	"sort"
	"sync"
	"time"

	"github.com/mpvl/unique"
)

// PingTracker handle containing all required data
type PingTracker struct {
	NextSeqNr int
	seqNrs    []int
	latencies []time.Duration
	lock      sync.Mutex
}

// New creates a new PingTracker
func New() *PingTracker {
	return &PingTracker{}
}

// Track measured sequence number and latency
func (tracker *PingTracker) Track(SeqNr int, Latency time.Duration) {
	tracker.lock.Lock()
	defer tracker.lock.Unlock()

	tracker.seqNrs = append(tracker.seqNrs, SeqNr)
	tracker.latencies = append(tracker.latencies, Latency)
}

// Calculate packet loss and latency for all data stored in PingTracker
func (tracker *PingTracker) Calculate() (int, int, time.Duration) {
	tracker.lock.Lock()
	defer tracker.lock.Unlock()

	loss := tracker.calculateLoss()
	count := len(tracker.latencies)
	latency := tracker.calculateLatency()

	// empty the slice but keep the memory
	tracker.seqNrs = tracker.seqNrs[:0]
	tracker.latencies = tracker.latencies[:0]

	return count, loss, latency
}

func (tracker *PingTracker) calculateLatency() time.Duration {
	count := len(tracker.latencies)
	if count == 0 {
		return 0 * time.Nanosecond
	}
	total := int64(0)
	for _, entry := range tracker.latencies {
		total += entry.Nanoseconds()
	}
	return time.Duration(total)
}

func (tracker *PingTracker) calculateLoss() int {
	if len(tracker.seqNrs) == 0 {
		return 0
	}
	// Sort all sequence numbers and remove duplicates
	sort.Ints(tracker.seqNrs)
	unique.Ints(&tracker.seqNrs)
	// sequence numbers can wrap around!
	// In this case, we'd get something like [ 0, 1, 2, 3, 65534, 65535 ]
	//Split into two slices [ 65534, 65535 ] and [ 0, 1, 2 ] using nextSeqNr as a boundary
	// Process the higher slice first (pre-wrap) and then the lower one (post-wrap)
	// TODO: avoid allocating new slices: determine the index where 'higher' starts and work w/ sub-slices instead
	higher := make([]int, 0)
	lower := make([]int, 0)
	for _, seqNr := range tracker.seqNrs {
		if seqNr >= tracker.NextSeqNr {
			higher = append(higher, seqNr)
		} else if seqNr < tracker.NextSeqNr-10000 {
			lower = append(lower, seqNr)
		}

	}
	total := 0
	if len(higher) > 0 {
		total = higher[0] - tracker.NextSeqNr
		total += countGaps(higher)
		tracker.NextSeqNr = higher[len(higher)-1] + 1
	}
	if len(lower) > 0 {
		tracker.NextSeqNr = 0
		total += lower[0] - tracker.NextSeqNr
		total += countGaps(lower)
		tracker.NextSeqNr = lower[len(lower)-1] + 1
	}

	return total
}

func countGaps(sequence []int) int {
	count := len(sequence)
	if count < 2 {
		return 0
	}
	total := 0
	for i := 0; i < count-1; i++ {
		total += sequence[i+1] - sequence[i] - 1
	}
	return total
}
