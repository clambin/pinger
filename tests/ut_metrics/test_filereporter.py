import os

from metrics.probe import Probe, Probes
from metrics.reporter import FileReporter, Reporters


class SimpleProbe(Probe):
    def __init__(self, test_sequence):
        super().__init__()
        self.test_sequence = test_sequence
        self.index = 0
        self.value = None

    def measure(self):
        self.value = self.test_sequence[self.index]
        self.index += 1
        if self.index >= len(self.test_sequence):
            self.index = 0

    def measured(self):
        return self.value


def process_file(filename):
    output = []
    with open(filename, 'r') as f:
        # skip header
        f.readline()
        line = 0
        for entry in f.readlines():
            # format is: <timestamp>,<val>[,val] ...
            fields = entry.split(',')
            index = 0
            for field in fields[1:]:
                if line == 0:
                    output.append([int(field)])
                else:
                    output[index].append(int(field))
                index += 1
            line += 1
    if len(output) == 1:
        output = output[0]
    return output


def test_single():
    test_data = [0, 1, 2, 3, 4]
    reporters = Reporters()
    probes = Probes()
    reporters.register(FileReporter('reporter.log'))
    reporters.add(probes.register(SimpleProbe(test_data)), 'test_single', '')
    reporters.start()
    for i in test_data:
        probes.run()
        reporters.run()
    assert test_data == process_file('reporter.log')
    os.remove('reporter.log')


def test_multiple():
    test_data = [
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0],
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0]
    ]
    reporter = FileReporter('reporter.log')
    probes = Probes()
    for i in range(len(test_data)):
        reporter.add(probes.register(SimpleProbe(test_data[i])),
                     'test_multiple_labeled', '', 'source', f'dest{i}')
    reporter.start()
    for i in range(len(test_data[0])):
        probes.run()
        reporter.run()
    assert test_data == process_file('reporter.log')
    os.remove('reporter.log')


def test_single_labeled():
    reporter = FileReporter('reporter.log')
    test_data = [1, 2, 3, 4]
    probe = SimpleProbe(test_data)
    reporter.add(probe, 'test_single_labeled', '', 'source', 'dest')
    reporter.start()
    for i in test_data:
        probe.measure()
        reporter.run()
    assert test_data == process_file('reporter.log')
    os.remove('reporter.log')


def test_multiple_labeled():
    test_data = [
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0],
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0]
    ]
    reporter = FileReporter('reporter.log')
    probes = Probes()
    for i in range(len(test_data)):
        reporter.add(probes.register(SimpleProbe(test_data[i])),
                     'test_multiple_labeled', '', 'source', f'dest{i}')
    reporter.start()
    for i in range(len(test_data[0])):
        probes.run()
        reporter.run()
    assert test_data == process_file('reporter.log')
    os.remove('reporter.log')
