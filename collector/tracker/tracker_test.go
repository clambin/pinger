package tracker_test

import (
	"github.com/clambin/pinger/collector/tracker"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type Entry struct {
	seqNr   int
	latency time.Duration
}

type Outcome struct {
	count     int
	nextSeqNr int
	loss      int
	latency   time.Duration
}

var testCases = []struct {
	description string
	input       []Entry
	output      Outcome
}{
	// No data
	{
		description: "No data received",
		input:       []Entry{},
		output:      Outcome{0, 0, 0, 0 * time.Millisecond},
	},
	// 	Packets may come in out of order
	{
		description: "Packets may come in out of order",
		input: []Entry{
			{0, 25 * time.Millisecond},
			{2, 50 * time.Millisecond},
			{1, 75 * time.Millisecond},
		},
		output: Outcome{3, 3, 0, 150 * time.Millisecond},
	},
	{
		description: "Duplicate packets are ignored",
		input: []Entry{
			{3, 50 * time.Millisecond},
			{4, 50 * time.Millisecond},
			{4, 50 * time.Millisecond},
			{5, 50 * time.Millisecond},
		},
		output: Outcome{4, 6, 0, 200 * time.Millisecond},
	},
	{
		description: "Lose one packet",
		input: []Entry{
			{6, 50 * time.Millisecond},
			// lose 7
			{8, 50 * time.Millisecond},
		},
		output: Outcome{2, 9, 1, 100 * time.Millisecond},
	},
	{
		description: "Lose packets between calls to Calculate",
		input: []Entry{
			// lose 9
			{10, 50 * time.Millisecond},
			{11, 50 * time.Millisecond},
			{12, 50 * time.Millisecond},
		},
		output: Outcome{3, 13, 1, 150 * time.Millisecond},
	},
	{
		description: "Fast forward to 30000",
		input: []Entry{
			{29999, 50 * time.Millisecond},
		},
		output: Outcome{1, 30000, 29999 - 13, 50 * time.Millisecond},
	},
	{
		description: "Support rollover of sequence numbers",
		input: []Entry{
			// lose 30000
			{30001, 50 * time.Millisecond},
			{30002, 50 * time.Millisecond},
			// lose 0
			{1, 50 * time.Millisecond},
			{2, 50 * time.Millisecond},
		},
		output: Outcome{4, 3, 2, 200 * time.Millisecond},
	},
	{
		description: "Recent (delayed) packets aren't interpreted as a rollover",
		input: []Entry{
			{0, 50 * time.Millisecond},
			{2, 50 * time.Millisecond},
			{3, 50 * time.Millisecond},
			{4, 50 * time.Millisecond},
		},
		output: Outcome{4, 5, 0, 200 * time.Millisecond},
	},
	{
		description: "fast-forward to 30000",
		input:       []Entry{{29999, 50 * time.Millisecond}},
		output:      Outcome{1, 30000, 29994, 50 * time.Millisecond},
	},
	{
		description: "delayed packets before rollover are ignored",
		input: []Entry{
			{29998, 50 * time.Millisecond},
			{29999, 50 * time.Millisecond},
			{30000, 50 * time.Millisecond},
			{30002, 50 * time.Millisecond},
			{30001, 50 * time.Millisecond},
			{0, 50 * time.Millisecond},
			{1, 50 * time.Millisecond},
			{2, 50 * time.Millisecond},
		},
		output: Outcome{8, 3, 0, 400 * time.Millisecond},
	},
}

func TestPingTracker(t *testing.T) {
	tr := tracker.New()

	for _, testCase := range testCases {
		for _, input := range testCase.input {
			tr.Track(input.seqNr, input.latency)
		}
		count, loss, latency := tr.Calculate()
		assert.Equal(t, testCase.output.count, count, testCase.description+" (count)")
		assert.Equal(t, testCase.output.nextSeqNr, tr.NextSeqNr, testCase.description+" (next sequence nr)")
		assert.Equal(t, testCase.output.loss, loss, testCase.description+" (loss)")
		assert.Equal(t, testCase.output.latency, latency, testCase.description+" (latency)")
	}
}

func TestPingTracker_Panic(t *testing.T) {
	tr := tracker.New()

	tr.Track(1000, 50*time.Millisecond)
	count, loss, _ := tr.Calculate()
	assert.Equal(t, 1, count)
	assert.Equal(t, 1000, loss)

	tr.Track(0, 50*time.Millisecond)
	tr.Track(1, 50*time.Millisecond)
	tr.Track(3, 50*time.Millisecond)

	count, loss, _ = tr.Calculate()
	assert.Equal(t, 3, count)
	assert.Equal(t, 1, loss)
}
