import logging
import platform
import queue
import re
import shlex
import subprocess
import threading
import time

from prometheus_client import Gauge, start_http_server


class Metric:
    def __init__(self, name, description, label=None, key=None):
        self.name = name
        self.description = description
        self.label = label
        self.key = key
        gauge = Reporter.get().gauge(name, description, label)
        self.gauge = gauge

    def __str__(self):
        return ""

    def measure(self):
        return None

    def run(self):
        val = self.measure()
        if val:
            logging.debug(f'{self.name}: {val}')
            self.report(val)

    def report(self, val):
        if self.label:
            logging.info(f'{self.name}[{self.label}={self.key}] = {val}')
            self.gauge.labels(self.key).set(val)
        else:
            logging.info(f'{self.name} = {val}')
            self.gauge.set(val)


class FileMetric(Metric):
    def __init__(self, name, description, filename, divider=1):
        self.filename = filename
        self.divider = divider
        super().__init__(name, description)

    def __str__(self):
        return self.filename

    def measure(self):
        try:
            with open(self.filename) as f:
                data = float(f.readline())/self.divider
        except IOError as error:
            logging.error(f'Could not read {self.filename}: {error}')
        return data


class Reporter:
    reporter = None

    @classmethod
    def get(cls, portno=8080):
        if not cls.reporter:
            cls.reporter = Reporter(portno)
        return cls.reporter

    def __init__(self, portno):
        self.portno = portno
        self.metrics = []
        self.gauges = {}
        start_http_server(self.portno)

    def gauge(self, name, description, label=None):
        if not name in self.gauges.keys():
            self.gauges[name] = Gauge(name, description, label)
        return self.gauges[name]

    def add(self, metric):
        logging.info(f'New metric {metric.name} for {metric}')
        self.metrics.append(metric)

    def run(self):
        for metric in self.metrics:
            metric.run()


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


class ProcessMetric(ProcessReader):
    def __init__(self, name, description, cmd):
        self.cmd = cmd
        super().__init__(f'{cmd}')
        self.name = name
        self.description = description

    def __str__(self):
        return self.cmd

    def process(self, lines):
        return None

    def measure(self):
        lines = []
        for line in self.read(): lines.append(line)
        return self.process(lines)

    def report(self, val):
        pass

    def run(self):
        val = self.measure()
        logging.debug(f'{self.name}: {val}')
        self.report(val)


class PingMetric(ProcessMetric):
    def __init__(self, host):
        ping = '/bin/ping' if platform.system() == 'Linux' else '/sbin/ping'
        self.host = host
        super().__init__('pinger', 'Pinger', f'{ping} {self.host}')
        self.latency = Metric(f'{self.name}_latency', 'Latency', ['host'], self.host)
        self.packet_loss = Metric(f'{self.name}_packet_loss', 'Packet loss', ['host'], self.host)
        self.next_seqno = None

    def __str__(self):
        return self.host

    def process(self, lines):
        latencies = []
        packet_losses = []
        for line in lines:
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
        return latency, packet_loss

    def report(self, val):
        (latency, packet_loss) = val
        if latency is not None: self.latency.report(latency)
        if packet_loss is not None: self.packet_loss.report(packet_loss)


if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    r = Reporter.get(8080)
    r.add(PingMetric('www.telenet.be'))
    r.add(PingMetric('192.168.0.221'))
    while True:
        r.run()
        time.sleep(10)
