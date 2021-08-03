package pingtracker

import (
	"sort"
	"sync"
	"time"
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

func (tracker *PingTracker) calculateLatency() (total time.Duration) {
	count := len(tracker.latencies)
	if count == 0 {
		return
	}

	for _, entry := range tracker.latencies {
		total += entry
	}
	return
}

func (tracker *PingTracker) calculateLoss() int {
	if len(tracker.seqNrs) == 0 {
		return 0
	}
	// Sort all sequence numbers and remove duplicates
	tracker.seqNrs = unique(tracker.seqNrs)

	// sequence numbers can wrap around!
	// In this case, we'd get something like [ 0, 1, 2, 3, 65534, 65535 ]
	// Split into two lists [ 65534, 65535 ] and [ 0, 1, 2 ] using nextSeqNr as a boundary
	// Process the higher list first (pre-wrap) and then the lower one (post-wrap)
	index := 0
	for ; index < len(tracker.seqNrs); index++ {
		if tracker.seqNrs[index] >= tracker.NextSeqNr-60 { // Allow up to 60 packets (1 min) of old packets
			break
		}
	}
	total := 0
	// pre-wrap
	// skip to nextSeqNr
	i := index
	for ; i < len(tracker.seqNrs) && tracker.seqNrs[i] < tracker.NextSeqNr; i++ {
	}
	if i < len(tracker.seqNrs) {
		total = tracker.seqNrs[i] - tracker.NextSeqNr
		total += countGaps(tracker.seqNrs[i:])
		tracker.NextSeqNr = tracker.seqNrs[len(tracker.seqNrs)-1] + 1
	}
	// post-wrap
	if index > 0 {
		tracker.NextSeqNr = 0
		total += tracker.seqNrs[0] // - tracker.NextSeqNr
		total += countGaps(tracker.seqNrs[:index])
		tracker.NextSeqNr = tracker.seqNrs[index-1] + 1
	}

	return total
}

func unique(seqNrs []int) (result []int) {
	uniqueSeqNrs := make(map[int]bool)
	for _, seqNr := range seqNrs {
		if _, ok := uniqueSeqNrs[seqNr]; ok == false {
			uniqueSeqNrs[seqNr] = true
			result = append(result, seqNr)
		}
	}
	sort.Ints(result)
	return
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
