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

func (tracker *PingTracker) calculateLoss() (gap int) {
	if len(tracker.seqNrs) == 0 {
		return 0
	}
	// Sort all sequence numbers and remove duplicates
	tracker.seqNrs = unique(tracker.seqNrs)

	// sequence numbers can roll over!
	// In this case, we'd get something like [ 0, 1, 2, 3, 65534, 65535 ]
	// Split into two lists [ 65534, 65535 ] and [ 0, 1, 2 ] using nextSeqNr as a boundary
	// Process the higher list first (pre-rollover) and then the lower one (post-rollover)

	index := 0
	for index < len(tracker.seqNrs) && tracker.seqNrs[index] < tracker.NextSeqNr-60 {
		index++
	}

	// pre-rollover / no rollover
	gap = tracker.processRange(tracker.seqNrs[index:])

	if index > 0 {
		tracker.NextSeqNr = 0
		gap += tracker.processRange(tracker.seqNrs[:index])
	}

	return
}

func unique(seqNrs []int) (result []int) {
	uniqueSeqNrs := make(map[int]struct{})
	for _, seqNr := range seqNrs {
		if _, ok := uniqueSeqNrs[seqNr]; ok == false {
			uniqueSeqNrs[seqNr] = struct{}{}
			result = append(result, seqNr)
		}
	}
	sort.Ints(result)
	return
}

func (tracker *PingTracker) processRange(sequence []int) (gap int) {
	count := len(sequence)
	if count == 0 {
		panic("processRange: sequence range should not be empty")
	}

	index := 0
	// skip older packets
	for ; index < count && sequence[index] < tracker.NextSeqNr; index++ {
	}

	for ; index < count; index++ {
		gap += sequence[index] - tracker.NextSeqNr
		tracker.NextSeqNr = sequence[index] + 1
	}

	return
}
