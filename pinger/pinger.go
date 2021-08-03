package pinger

import (
	"bufio"
	"context"
	"github.com/clambin/pinger/pingtracker"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

type Pinger struct {
	Trackers      map[string]*pingtracker.PingTracker
	packetsMetric *prometheus.Desc
	lossMetric    *prometheus.Desc
	latencyMetric *prometheus.Desc
}

// New creates a Pinger for the specified hosts
func New(hosts []string) (pinger *Pinger) {
	pinger = &Pinger{
		Trackers: make(map[string]*pingtracker.PingTracker),
		packetsMetric: prometheus.NewDesc(
			prometheus.BuildFQName("pinger", "", "packet_count"),
			"Pinger total packet count",
			[]string{"host"},
			nil,
		),
		lossMetric: prometheus.NewDesc(
			prometheus.BuildFQName("pinger", "", "packet_loss_count"),
			"Pinger total measured packet loss",
			[]string{"host"},
			nil,
		),
		latencyMetric: prometheus.NewDesc(
			prometheus.BuildFQName("pinger", "", "latency_seconds"),
			"Pinger latency in seconds",
			[]string{"host"},
			nil,
		),
	}

	for _, host := range hosts {
		pinger.Trackers[host] = pingtracker.New()
	}

	return
}

// Run starts the pingers
func (pinger *Pinger) Run(ctx context.Context) {
	for host, tracker := range pinger.Trackers {
		log.WithField("host", host).Debug("starting tracker")
		go func(host string, tracker *pingtracker.PingTracker) {
			err := spawnedPinger(host, tracker)

			if err != nil {
				log.WithError(err).Error("failed to run tracker")
			}
		}(host, tracker)
	}

	<-ctx.Done()
}

// spawnedPinger spawns a ping process and reports to a specified PingTracker
func spawnedPinger(host string, tracker *pingtracker.PingTracker) (err error) {
	var (
		cmd     string
		pingOut io.ReadCloser
		scanner *bufio.Scanner
		line    string
		seqNr   int
		rtt     float64
		latency time.Duration
	)
	switch runtime.GOOS {
	case "linux":
		cmd = "/bin/ping"
	default:
		cmd = "/sbin/ping"
	}

	r := regexp.MustCompile(`(icmp_seq|seq)=(\d+) .+time=(\d+\.?\d*) ms`)
	pingProcess := exec.Command(cmd, host)

	pingOut, err = pingProcess.StdoutPipe()

	if err == nil {
		scanner = bufio.NewScanner(pingOut)
		err = pingProcess.Start()
	}

	if err == nil {
		for scanner.Scan() {
			line = scanner.Text()
			for _, match := range r.FindAllStringSubmatch(line, -1) {
				seqNr, _ = strconv.Atoi(match[2])
				rtt, _ = strconv.ParseFloat(match[3], 64)
				latency = time.Duration(rtt*1000) * time.Microsecond

				tracker.Track(seqNr, latency)

				// log.Debugf("%s: seqno=%d, latency=%v", host, seqNr, latency)
			}
		}
	}

	return
}
