package pinger

import (
	"runtime"
	"time"

	"github.com/go-ping/ping"
	log "github.com/sirupsen/logrus"

	"pinger/internal/metrics"
	"pinger/internal/pingtracker"
)

func Run(hosts []string, interval time.Duration) {
	RunNTimes(hosts, interval, -1)
}

func RunNTimes(hosts []string, interval time.Duration, passes int) {
	var trackers = make(map[string]*pingtracker.PingTracker, len(hosts))

	for _, host := range hosts {
		trackers[host] = pingtracker.New()

		go func(host string) {
			pinger, err := ping.NewPinger(host)
			if err != nil {
				panic(err)
			}

			if runtime.GOOS == "linux" {
				pinger.SetPrivileged(true)
			}

			pinger.Interval = 5 * time.Second

			pinger.OnRecv = func(pkt *ping.Packet) {
				log.Debugf("%s: seq nr %d, latency %v", host, pkt.Seq, pkt.Rtt)
				trackers[host].Track(pkt.Seq, pkt.Rtt)
			}
			if err = pinger.Run(); err != nil {
				panic(err)
			}
		}(host)
	}

	for {
		if passes != -1 {
			passes--
			if passes == 0 {
				break
			}
		}

		time.Sleep(interval)

		for name, tracker := range trackers {
			count, loss, latency := tracker.Calculate()

			metrics.Measure(name, count, loss, latency)

			log.Debugf("%s: received: %d, loss: %d, latency:%v", name, count, loss, latency)
		}
	}
}
