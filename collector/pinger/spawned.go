package pinger

import (
	"bufio"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// SpawnedPingers spawns a collector process and reports to a specified Tracker
func SpawnedPingers(ch chan Response, hosts ...string) error {
	var wg sync.WaitGroup
	wg.Add(len(hosts))
	for _, host := range hosts {
		go func(host string) {
			defer wg.Done()
			_ = SpawnedPinger(ch, host)
		}(host)
	}
	wg.Wait()
	return nil
}

// SpawnedPinger spawns a collector process and reports to a specified Tracker
func SpawnedPinger(ch chan Response, host string) error {
	var cmd string
	switch runtime.GOOS {
	case "linux":
		cmd = "/bin/collector"
	default:
		cmd = "/sbin/collector"
	}

	r := regexp.MustCompile(`(icmp_seq|seq)=(\d+) .+time=(\d+\.?\d*) ms`)
	pingProcess := exec.Command(cmd, host)

	pingOut, err := pingProcess.StdoutPipe()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(pingOut)
	if err = pingProcess.Start(); err != nil {
		return err
	}

	for scanner.Scan() {
		line := scanner.Text()
		for _, match := range r.FindAllStringSubmatch(line, -1) {
			seqNr, _ := strconv.Atoi(match[2])
			rtt, _ := strconv.ParseFloat(match[3], 64)
			latency := time.Duration(rtt*1000) * time.Microsecond

			ch <- Response{
				Host:       host,
				SequenceNr: seqNr,
				Latency:    latency,
			}
		}
	}

	return err
}
