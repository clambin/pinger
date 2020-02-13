# Copyright 2020 by Christophe Lambin
# All rights reserved.

import logging
import queue
import shlex
import subprocess
import threading

from prometheus_client import Gauge, start_http_server


class Reporter:
    reporter = None

    @classmethod
    def get(cls, portno=8080):
        if cls.reporter is None:
            cls.reporter = Reporter(portno)
        return cls.reporter

    def __init__(self, portno):
        self.portno = portno
        self.metrics = []
        self.gauges = {}

    def start(self):
        start_http_server(self.portno)

    def gauge(self, name, description, label=None):
        if name not in self.gauges.keys():
            self.gauges[name] = Gauge(name, description) if label is None else Gauge(name, description, label)
        return self.gauges[name]

    def add(self, metric):
        logging.info(f'New metric {metric.name} for {metric}')
        self.metrics.append(metric)

    def run(self):
        for metric in self.metrics:
            metric.run()


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

    def report(self, val):
        if self.label:
            logging.debug(f'{self.name}[{self.label}={self.key}] = {val}')
            self.gauge.labels(self.key).set(val)
        else:
            logging.debug(f'{self.name} = {val}')
            self.gauge.set(val)

    def run(self):
        val = self.measure()
        if val:
            logging.debug(f'{self.name}: {val}')
            self.report(val)


class FileMetric(Metric):
    def __init__(self, name, description, filename, divider=1):
        super().__init__(name, description)
        self.filename = filename
        self.divider = divider
        f = open(self.filename)
        f.close()

    def __str__(self):
        return self.filename

    def measure(self):
        with open(self.filename) as f:
            return float(f.readline())/self.divider


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
        out = []
        try:
            while True:
                line = self.queue.get_nowait()
                out.append(line)
        except queue.Empty:
            pass
        return out

    def running(self):
        return self.thread.is_alive() or not self.queue.empty()


class ProcessMetric:
    def __init__(self, name, description, cmd):
        self.name = name
        self.description = description
        self.cmd = cmd
        self.reader = ProcessReader(cmd)

    def __str__(self):
        return self.cmd

    def running(self):
        return self.reader.running()

    def process(self, lines):
        return None

    def measure(self) -> object:
        lines = []
        for line in self.reader.read(): lines.append(line)
        return self.process(lines)

    def report(self, val):
        pass

    def run(self):
        val = self.measure()
        logging.debug(f'{self.name}: {val}')
        self.report(val)
