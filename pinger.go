package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/go-ping/ping"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"pinger/internal/metrics"
	"pinger/internal/pingtracker"
	"pinger/internal/version"
)

func main() {
	cfg := struct {
		port     int
		debug    bool
		interval int
	}{}
	a := kingpin.New(filepath.Base(os.Args[0]), "pinger")

	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("port", "Metrics listener port").Default("8080").IntVar(&cfg.port)
	a.Flag("debug", "Log debug messages").BoolVar(&cfg.debug)
	a.Flag("interval", "Interval").Default("8080").IntVar(&cfg.interval)
	hosts := a.Arg("hosts", "hosts to ping").Strings()

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	if cfg.debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Infof("pinger v%s", version.BuildVersion)

	metrics.Init(cfg.port)

	var trackers = make(map[string]*pingtracker.PingTracker, len(*hosts))

	for _, host := range *hosts {
		trackers[host] = pingtracker.New()

		go func(host string) {
			pinger, err := ping.NewPinger(host)
			if err != nil {
				panic(err)
			}
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
		time.Sleep(5 * time.Second)

		for name, tracker := range trackers {
			count, loss, latency := tracker.Calculate()

			metrics.Measure(name, count, loss, latency)

			log.Debugf("%s: received: %d, loss: %d, latency:%v\n", name, count, loss, latency)
		}
	}
}
