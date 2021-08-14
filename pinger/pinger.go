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

// Monitor pings a number of hosts and measures latency & packet loss
type Monitor struct {
	Pinger        func(host string, ch chan PingResponse) (err error)
	Trackers      map[string]*pingtracker.PingTracker
	packets       chan PingResponse
	packetsMetric *prometheus.Desc
	lossMetric    *prometheus.Desc
	latencyMetric *prometheus.Desc
}

type PingResponse struct {
	Host       string
	SequenceNr int
	Latency    time.Duration
}

// New creates a Monitor for the specified hosts
func New(hosts []string) (monitor *Monitor) {
	monitor = &Monitor{
		Pinger:   spawnedPinger,
		Trackers: make(map[string]*pingtracker.PingTracker),
		packets:  make(chan PingResponse),
		packetsMetric: prometheus.NewDesc(
			prometheus.BuildFQName("pinger", "", "packet_count"),
			"Monitor total packet count",
			[]string{"host"},
			nil,
		),
		lossMetric: prometheus.NewDesc(
			prometheus.BuildFQName("pinger", "", "packet_loss_count"),
			"Monitor total measured packet loss",
			[]string{"host"},
			nil,
		),
		latencyMetric: prometheus.NewDesc(
			prometheus.BuildFQName("pinger", "", "latency_seconds"),
			"Monitor latency in seconds",
			[]string{"host"},
			nil,
		),
	}

	for _, host := range hosts {
		monitor.Trackers[host] = pingtracker.New()
	}

	return
}

// Run starts the pinger(s)
func (monitor *Monitor) Run(ctx context.Context) {
	monitor.startPingers()

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case packet := <-monitor.packets:
			monitor.Trackers[packet.Host].Track(packet.SequenceNr, packet.Latency)
		}
	}
}

func (monitor *Monitor) startPingers() {
	for host := range monitor.Trackers {
		log.WithField("host", host).Debug("starting pinger")
		go func(host string) {
			err := monitor.Pinger(host, monitor.packets)

			if err != nil {
				log.WithError(err).Fatal("failed to run pinger")
			}
		}(host)
	}
}

// spawnedPinger spawns a ping process and reports to a specified PingTracker
func spawnedPinger(host string, ch chan PingResponse) (err error) {
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

				ch <- PingResponse{
					Host:       host,
					SequenceNr: seqNr,
					Latency:    latency,
				}
			}
		}
	}

	return
}
