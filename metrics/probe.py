# Copyright 2020 by Christophe Lambin
# All rights reserved.

import queue
import shlex
import subprocess
import threading


class Probe:
    def __init__(self):
        self.val = None

    def measure(self):
        return None

    def run(self):
        self.val = self.measure()

    def measured(self):
        return self.val


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
        out = []
        for probe in self.probes:
            out.append(probe.measured())
        return out


class FileProbe(Probe):
    def __init__(self, filename, divider=1):
        super().__init__()
        self.filename = filename
        self.divider = divider
        f = open(self.filename)
        f.close()

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


class ProcessProbe(Probe):
    def __init__(self, cmd):
        super().__init__()
        self.cmd = cmd
        self.reader = ProcessReader(cmd)

    def running(self):
        return self.reader.running()

    def process(self, lines):
        return None

    def measure(self):
        val = None
        # process may not have any data to measure
        while val is None:
            lines = []
            for line in self.reader.read(): lines.append(line)
            val = self.process(lines)
        return val
