package pingtracker

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPingTracker(t *testing.T) {
	tracker := New()

	// No data
	count, loss, latency := tracker.Calculate()
	assert.Equal(t, 0, count)
	assert.Equal(t, 0, loss)
	assert.Equal(t, 0*time.Millisecond, latency)
	assert.Equal(t, 0, tracker.NextSeqNr)

	// Out of order
	//tracker.Track(0, 50*time.Millisecond)
	tracker.Track(0, 25*time.Millisecond)
	tracker.Track(2, 50*time.Millisecond)
	tracker.Track(1, 75*time.Millisecond)

	count, loss, latency = tracker.Calculate()

	assert.Equal(t, 3, count)
	assert.Equal(t, 0, loss)
	assert.Equal(t, 50*time.Millisecond, latency)
	assert.Equal(t, 3, tracker.NextSeqNr)

	// Ignore duplicates
	tracker.Track(3, 50*time.Millisecond)
	tracker.Track(4, 50*time.Millisecond)
	tracker.Track(4, 50*time.Millisecond)
	tracker.Track(5, 50*time.Millisecond)

	count, loss, latency = tracker.Calculate()

	assert.Equal(t, 4, count)
	assert.Equal(t,0, loss)
	assert.Equal(t, 50*time.Millisecond, latency)
	assert.Equal(t,6, tracker.NextSeqNr)

	// Lose one packet
	tracker.Track(6, 50*time.Millisecond)
	// lose 7
	tracker.Track(8, 50*time.Millisecond)

	count, loss, latency = tracker.Calculate()

	assert.Equal(t, 2, count)
	assert.Equal(t, 1, loss)
	assert.Equal(t, 50*time.Millisecond, latency)
	assert.Equal(t, 9, tracker.NextSeqNr)

	// Lose packets between calculations
	// lose 9
	tracker.Track(10, 50*time.Millisecond)
	tracker.Track(11, 50*time.Millisecond)
	tracker.Track(12, 50*time.Millisecond)

	count, loss, latency = tracker.Calculate()

	assert.Equal(t, 3, count)
	assert.Equal(t, 1, loss)
	assert.Equal(t, 50*time.Millisecond, latency)
	assert.Equal(t, 13, tracker.NextSeqNr)

	// Fast forward to 30,000
	tracker.Track(30000, 50*time.Millisecond)

	count, loss, latency = tracker.Calculate()

	assert.Equal(t, 1, count)
	assert.Equal(t, 30000-13, loss)
	assert.Equal(t, 50*time.Millisecond, latency)
	assert.Equal(t, 30001, tracker.NextSeqNr)


	// Support wraparound of sequence numbers
	// lose 30001
	tracker.Track(30002, 50*time.Millisecond)
	tracker.Track(30003, 50*time.Millisecond)
	// wrap around & lose 0
	tracker.Track(1, 50*time.Millisecond)
	tracker.Track(2, 50*time.Millisecond)
	tracker.Track(3, 50*time.Millisecond)

	count, loss, latency = tracker.Calculate()

	assert.Equal(t, 5, count)
	assert.Equal(t, 2, loss)
	assert.Equal(t, 50*time.Millisecond, latency)
	assert.Equal(t, 4, tracker.NextSeqNr)

	// recent (delayed) packets aren't interpreted as a wrap-around
	tracker.Track(2, 50*time.Millisecond)
	tracker.Track(3, 50*time.Millisecond)
	tracker.Track(4, 50*time.Millisecond)

	count, loss, latency = tracker.Calculate()

	assert.Equal(t, 3, count)
	assert.Equal(t, 0, loss)
	assert.Equal(t, 50*time.Millisecond, latency)
	assert.Equal(t, 5, tracker.NextSeqNr)
}