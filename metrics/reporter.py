# Copyright 2020 by Christophe Lambin
# All rights reserved.

import logging
import time
from abc import ABC, abstractmethod

from prometheus_client import start_http_server, Gauge


# Convenience call to make code a little simpler when dealing with multiple reporters
class Reporters:
    def __init__(self):
        self.reporters = []

    def register(self, reporter):
        self.reporters.append(reporter)

    def add(self, probe, name, description, label=None, key=None):
        for reporter in self.reporters:
            reporter.add(probe, name, description, label, key)

    def start(self):
        for reporter in self.reporters:
            reporter.start()

    def run(self):
        for reporter in self.reporters:
            reporter.run()


class Reporter(ABC):
    def __init__(self):
        self.probes = {}

    def start(self):
        pass

    def add(self, probe, name, description, label=None, key=None):
        # No duplicates allowed
        for p in self.probes.values():
            if p['name'] == name and p['label'] == label and p['key'] == key:
                raise KeyError("Probe already exists")
        self.probes[probe] = {'name': name, 'description': description, 'label': label, 'key': key}

    def get_probe_info(self, probe):
        info = self.probes[probe]
        return info['name'], info['label'], info['key']

    @abstractmethod
    def report(self, probe, value):
        pass

    def pre_run(self):
        pass

    def post_run(self):
        pass

    def run(self):
        self.pre_run()
        for probe in self.probes:
            val = probe.measured()
            self.report(probe, val)
        self.post_run()


class PrometheusReporter(Reporter):
    def __init__(self, port=8080):
        super().__init__()
        self.port = port
        self.gauges = {}
        self.started = False

    def start(self):
        if not self.started:
            start_http_server(self.port)
            self.started = True

    def find_gauge(self, name, label):
        keyname = f'{name}|{label}' if label else name
        if keyname in self.gauges:
            return self.gauges[keyname]
        return None

    def make_gauge(self, name, description, label):
        if not self.find_gauge(name, label):
            if not label:
                self.gauges[name] = Gauge(name, description)
            else:
                keyname = f'{name}|{label}'
                self.gauges[keyname] = Gauge(name, description, [label])

    def add(self, m, name, description, label=None, key=None):
        super().add(m, name, description, label, key)
        self.make_gauge(name, description, label)

    def report(self, probe, val):
        super().report(probe, val)
        if val is not None:
            name, label, key = self.get_probe_info(probe)
            g = self.find_gauge(name, label)
            if label is not None:
                g = g.labels(key)
            g.set(val)


class FileReporter(Reporter):
    def __init__(self, filename):
        super().__init__()
        self.filename = filename
        self.reported = {}

    def header(self):
        out = []
        for probe in self.probes:
            tag = self.probes[probe]
            out.append(f'{tag["name"]}' if tag['label'] is None else f'{tag["name"]}-{tag["label"]}:{tag["key"]}')
        logging.info(out)
        return ','.join(out)

    def start(self):
        with open(self.filename, 'w') as f:
            f.write(f'Timestamp,{self.header()}\n')

    def report(self, probe, val):
        pass

    def pre_run(self):
        pass

    def post_run(self):
        with open(self.filename, 'a') as f:
            f.write(f'{time.strftime("%Y-%m-%dT%T")}')
            for probe in self.probes:
                f.write(f',{probe.measured()}')
            f.write('\n')

