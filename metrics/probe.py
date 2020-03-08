# Copyright 2020 by Christophe Lambin
# All rights reserved.

import queue
import shlex
import subprocess
import threading
from abc import ABC, abstractmethod


class Probe(ABC):
    def __init__(self):
        self.val = None

    @abstractmethod
    def measure(self):
        """Implement measurement logic in the inherited class"""

    def measured(self):
        return self.val

    def run(self):
        self.val = self.measure()


# Convenience class to make code a little simpler
class Probes:
    def __init__(self):
        self.probes = []

    def register(self, probe):
        self.probes.append(probe)
        return probe

    def run(self):
        for probe in self.probes:
            probe.run()

    def measured(self):
        return [probe.measured() for probe in self.probes]


class FileProbe(Probe):
    def __init__(self, filename, divider=1):
        super().__init__()
        self.filename = filename
        self.divider = divider
        f = open(self.filename)
        f.close()

    def process(self, content):
        return float(content) / self.divider

    def measure(self):
        with open(self.filename) as f:
            content = ''.join(f.readlines())
            return self.process(content)


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
        self.proc.stdout.close()

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


class ProcessProbe(Probe, ABC):
    def __init__(self, cmd):
        super().__init__()
        self.cmd = cmd
        self.reader = ProcessReader(cmd)

    @abstractmethod
    def process(self, lines):
        """Implement measurement logic in the inherited class"""

    def running(self):
        return self.reader.running()

    def measure(self):
        lines = []
        for line in self.reader.read(): lines.append(line)
        return self.process(lines)


class SubProbe(Probe):
    def __init__(self, name, parent):
        super().__init__()
        self.name = name
        self.parent = parent

    def measure(self):
        raise NotImplementedError('This should never be called')


class ProbeAggregator(ABC):
    def __init__(self, names):
        self.probes = {name: SubProbe(name, self) for name in names}

    def get_probe(self, name):
        return self.probes[name]

    def get_value(self, name):
        return self.probes[name].val

    def set_value(self, name, value):
        self.probes[name].val = value

    def get_values(self):
        return [self.get_value(probe) for probe in self.probes]

    @abstractmethod
    def measure(self):
        """Implement measurement logic in the inherited class"""

    def run(self):
        self.measure()
