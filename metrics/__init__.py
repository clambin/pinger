import logging
import platform
import queue
import re
import shlex
import subprocess
import threading
import time

from prometheus_client import Gauge, start_http_server


class BaseMetric:
    def __init__(self, name, description):
        self.name = name
        self.description = description

    def __str__(self):
        return ""

    def measure(self):
        return None

    def report(self, val):
        pass

    def run(self):
        val = self.measure()
        if val:
            logging.debug(f'{self.name}: {val}')
            self.report(val)


class Metric(BaseMetric):
    def __init__(self, name, description):
        super().__init__(name, description)
        self.gauge = Gauge(name, description)

    def report(self, val):
        self.gauge.set(val)


class FileMetric(Metric):
    def __init__(self, name, description, fname, divider=1):
        super().__init__(name, description)
        self.fname = fname
        self.divider = divider

    def __str__(self):
        return self.fname

    def measure(self):
        f = open(self.fname)
        data = f.readline()
        f.close()
        data = float(data)/self.divider
        return data


class ProcessReader:
    def __init__(self, cmd):
        self.cmd = cmd
        self.proc = subprocess.Popen(shlex.split(cmd), stdout=subprocess.PIPE, encoding='utf-8')
        self.queue = queue.Queue()
        self.thread = threading.Thread(target=self._enqueue_output)
        self.thread.daemon = True
        self.thread.start()

    def _enqueue_output(self):
        for line in iter(self.proc.stdout.readline, ''):
            self.queue.put(line)
            logging.debug(f'ProcessReader got [{line}]')
        self.proc.stdout.close()

    def __str__(self):
        return self.cmd

    def read(self):
        # TODO: check if process hasn't exited
        out = []
        try:
            while True:
                line = self.queue.get_nowait()
                out.append(line)
        except queue.Empty:
            pass
        return out


class Reporter:
    def __init__(self, portno):
        self.portno = portno
        self.metrics = {}

    def start(self):
        start_http_server(self.portno)

    def add(self, metric):
        logging.info(f'New metric {metric.name} for {metric}')
        self.metrics[metric.name] = metric

    def run(self):
        for metric in self.metrics.keys():
            self.metrics[metric].run()


class PingMetric(BaseMetric, ProcessReader):
    def __init__(self, host):
        self.host = host
        self.latency = Metric('pinger_latency', 'Latency')
        self.packet_loss = Metric('pinger_packet_loss', 'Packet loss')
        self.next_seqno = None
        ping = '/bin/ping' if platform.system() == 'Linux' else '/sbin/ping'
        BaseMetric.__init__(self, 'pinger', '')
        ProcessReader.__init__(self, f'{ping} {self.host}')

    def measure(self):
        latencies = []
        packet_losses = []
        for line in self.read():
            try:
                for keyword, seqno, latency in re.findall(r' (icmp_seq|seq)=(\d+) .+time=(\d+\.?\d*) ms', line):
                    seqno, latency = int(seqno), float(latency)
                    packet_loss = seqno - self.next_seqno if self.next_seqno else 0
                    logging.debug(f'{self.host}: {latency} ms, {packet_loss} loss')
                    latencies.append(latency)
                    packet_losses.append(packet_loss)
                    self.next_seqno = seqno + 1
            except TypeError:
                logging.warning(f'Cannot parse {line}')
        if not latencies:
            return None, None
        latency = float(sum(latencies))/len(latencies)
        packet_loss = sum(packet_losses)
        logging.info(f'{self.host}: {latency} ms, {packet_loss} loss')
        if latency is None or packet_loss is None:
            logging.error('failed')
        return latency, packet_loss

    def report(self, val):
        (latency, packet_loss) = val
        self.latency.report(latency)
        self.packet_loss.report(latency)



if __name__ == '__main__':
    logging.basicConfig(level=logging.DEBUG)
    reporter = Reporter(8081)
    reporter.add(PingMetric('192.168.0.1'))
    while True:
        reporter.run()
        time.sleep(1)
