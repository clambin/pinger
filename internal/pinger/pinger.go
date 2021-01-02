package pinger

import (
	"bufio"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"pinger/internal/pingtracker"
)

// Run runs a pinger for each specified host and reports the results every 'interval' duration
func Run(hosts []string, interval time.Duration) {
	var trackers = make(map[string]*pingtracker.PingTracker, len(hosts))

	for _, host := range hosts {
		trackers[host] = pingtracker.New()

		go spawnedPinger(host, trackers[host])
	}

	for {
		time.Sleep(interval)

		for name, tracker := range trackers {
			count, loss, latency := tracker.Calculate()

			packetsCounter.WithLabelValues(name).Add(float64(count))
			lossCounter.WithLabelValues(name).Add(float64(loss))
			latencyCounter.WithLabelValues(name).Add(latency.Seconds())

			log.Debugf("%s: received: %d, loss: %d, latency:%v", name, count, loss, latency)
		}
	}
}

// spawnedPinger spawns a ping process and reports to a specified PingTracker
func spawnedPinger(host string, tracker *pingtracker.PingTracker) {
	var (
		cmd     string
		err     error
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

	if pingOut, err = pingProcess.StdoutPipe(); err == nil {
		scanner = bufio.NewScanner(pingOut)

		if err = pingProcess.Start(); err == nil {

			for scanner.Scan() {
				line = scanner.Text()
				for _, match := range r.FindAllStringSubmatch(line, -1) {
					seqNr, _ = strconv.Atoi(match[2])
					rtt, _ = strconv.ParseFloat(match[3], 64)
					latency = time.Duration(int64(rtt * 1000000))

					tracker.Track(seqNr, latency)

					log.Debugf("%s: seqno=%d, latency=%v", host, seqNr, latency)
				}
			}
		}
	}

	log.Error(err)
}
