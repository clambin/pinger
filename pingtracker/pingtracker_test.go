package pingtracker_test

import (
	"github.com/clambin/pinger/pingtracker"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type Entry struct {
	seqNr   int
	latency time.Duration
}
type Outcome struct {
	count, nextSeqNr, loss int
	latency                time.Duration
}

var testCases = []struct {
	description string
	input       []Entry
	output      Outcome
}{
	// No data
	{
		"No data received",
		[]Entry{},
		Outcome{0, 0, 0, 0 * time.Millisecond},
	},
	// 	Packets may come in out of order
	{
		"Packets may come in out of order",
		[]Entry{
			{0, 25 * time.Millisecond},
			{2, 50 * time.Millisecond},
			{1, 75 * time.Millisecond},
		},
		Outcome{3, 3, 0, 150 * time.Millisecond},
	},
	{
		"Duplicate packets are ignored",
		[]Entry{
			{3, 50 * time.Millisecond},
			{4, 50 * time.Millisecond},
			{4, 50 * time.Millisecond},
			{5, 50 * time.Millisecond},
		},
		Outcome{4, 6, 0, 200 * time.Millisecond},
	},
	{
		"Lose one packet",
		[]Entry{
			{6, 50 * time.Millisecond},
			// lose 7
			{8, 50 * time.Millisecond},
		},
		Outcome{2, 9, 1, 100 * time.Millisecond},
	},
	{
		"Lose packets between calls to Calculate",
		[]Entry{
			// lose 9
			{10, 50 * time.Millisecond},
			{11, 50 * time.Millisecond},
			{12, 50 * time.Millisecond},
		},
		Outcome{3, 13, 1, 150 * time.Millisecond},
	},
	{
		"Fast forward to 30000",
		[]Entry{
			{29999, 50 * time.Millisecond},
		},
		Outcome{1, 30000, 29999 - 13, 50 * time.Millisecond},
	},
	{
		"Support wraparound of sequence numbers",
		[]Entry{
			// lose 30000
			{30001, 50 * time.Millisecond},
			{30002, 50 * time.Millisecond},
			// lose 0
			{1, 50 * time.Millisecond},
			{2, 50 * time.Millisecond},
		},
		Outcome{4, 3, 2, 200 * time.Millisecond},
	},
	{
		"Recent (delayed) packets aren't interpreted as a wrap-around",
		[]Entry{
			{0, 50 * time.Millisecond},
			{2, 50 * time.Millisecond},
			{3, 50 * time.Millisecond},
			{4, 50 * time.Millisecond},
		},
		Outcome{4, 5, 0, 200 * time.Millisecond},
	},
}

func TestPingTracker(t *testing.T) {
	tracker := pingtracker.New()

	for _, testCase := range testCases {
		for _, input := range testCase.input {
			tracker.Track(input.seqNr, input.latency)
		}
		count, loss, latency := tracker.Calculate()
		assert.Equal(t, testCase.output.count, count, testCase.description)
		assert.Equal(t, testCase.output.nextSeqNr, tracker.NextSeqNr, testCase.description)
		assert.Equal(t, testCase.output.loss, loss, testCase.description)
		assert.Equal(t, testCase.output.latency, latency, testCase.description)
	}
}
