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

