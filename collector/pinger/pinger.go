package pinger

import "time"

type Response struct {
	Host       string
	SequenceNr int
	Latency    time.Duration
}
