package pinger

import (
	"runtime"
	"time"

	"github.com/go-ping/ping"
	log "github.com/sirupsen/logrus"

	"pinger/internal/metrics"
	"pinger/internal/pingtracker"
)

type pingFunc func(string, *pingtracker.PingTracker)

func Run(hosts []string, interval time.Duration) {
	RunNTimes(hosts, interval, -1, Pinger)
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
