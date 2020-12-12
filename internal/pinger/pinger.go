package pinger

import (
	"bufio"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"github.com/go-ping/ping"
	log "github.com/sirupsen/logrus"

	"pinger/internal/metrics"
	"pinger/internal/pingtracker"
)

type pingFunc func(string, *pingtracker.PingTracker)

func Run(hosts []string, interval time.Duration) {
	RunNTimes(hosts, interval, -1, SpawnedPinger)
}

func RunNTimes(hosts []string, interval time.Duration, passes int, pinger pingFunc) (int, int, time.Duration) {
	var trackers = make(map[string]*pingtracker.PingTracker, len(hosts))

	for _, host := range hosts {
		trackers[host] = pingtracker.New()

		go pinger(host, trackers[host])
	}

	totalCount := 0
	totalLoss := 0
	totalLatency := int64(0)

	for {
		if passes != -1 {
			if passes == 0 {
				break
			}
			passes--
		}

		time.Sleep(interval)

		for name, tracker := range trackers {
			count, loss, latency := tracker.Calculate()
			metrics.Measure(name, count, loss, latency)

			log.Debugf("%s: received: %d, loss: %d, latency:%v", name, count, loss, latency)

			totalCount += count
			totalLoss += loss
			totalLatency += latency.Nanoseconds()
		}
	}

	return totalCount, totalLoss, time.Duration(totalLatency)
}

func Pinger(host string, tracker *pingtracker.PingTracker) {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		panic(err)
	}

	if runtime.GOOS == "linux" {
		pinger.SetPrivileged(true)
	}

	pinger.Interval = 10 * time.Second

	pinger.OnRecv = func(pkt *ping.Packet) {
		log.Debugf("%s: seq nr %d, latency %v", host, pkt.Seq, pkt.Rtt)
		tracker.Track(pkt.Seq, pkt.Rtt)
	}
	if err = pinger.Run(); err != nil {
		panic(err)
	}
}

// SpawnedPinger: go-ping is fairly expensive.  Replace with spawned ping process
func SpawnedPinger(host string, tracker *pingtracker.PingTracker) {
	var cmd string
	switch runtime.GOOS {
	case "linux":
		cmd = "/bin/ping"
	default:
		cmd = "/sbin/ping"
	}

	pingProcess := exec.Command(cmd, host)
	pingOut, err := pingProcess.StdoutPipe()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(pingOut)

	err = pingProcess.Start()
	if err != nil {
		panic(err)
	}

	r := regexp.MustCompile(`(icmp_seq|seq)=(\d+) .+time=(\d+\.?\d*) ms`)

	for scanner.Scan() {
		line := scanner.Text()
		for _, match := range r.FindAllStringSubmatch(line, -1) {
			seqNr, _ := strconv.Atoi(match[2])
			rtt, _ := strconv.ParseFloat(match[3], 64)
			latency := time.Duration(int64(rtt * 1000000))

			tracker.Track(seqNr, latency)

			log.Debugf("%s: seqno=%d, latency=%v", host, seqNr, latency)
		}
	}
}
