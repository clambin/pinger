import os
import pytest
from metrics.probe import FileProbe, ProcessProbe, Probes, ProbeAggregator
from tests.probes import SimpleProbe


class SimpleProcessProbe(ProcessProbe):
    def __init__(self, command):
        super().__init__(command)

    def process(self, lines):
        val = 0
        for line in lines:
            val += int(line)
        return val


def test_simple():
    testdata = [1, 2, 3, 4]
    probe = SimpleProbe(testdata)
    for val in testdata:
        probe.run()
        assert probe.measured() == val


def test_file():
    # create the file
    open('testfile.txt', 'w')
    probe = FileProbe('testfile.txt')
    for val in range(1, 10):
        with open('testfile.txt', 'w') as f:
            f.write(f'{val}')
        probe.run()
        assert probe.measured() == val
    os.remove('testfile.txt')


def test_bad_file():
    with pytest.raises(FileNotFoundError):
        FileProbe('testfile.txt')


def test_process():
    probe = SimpleProcessProbe('/bin/sh -c ./process_ut.sh')
    out = 0
    while probe.running():
        probe.run()
        out += probe.measured()
    assert out == 55


def test_bad_process():
    with pytest.raises(FileNotFoundError):
        SimpleProcessProbe('missing_process_ut.sh')


def test_probes():
    test_data = [
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0],
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0]
    ]
    probes = Probes()
    for test in test_data:
        probes.register(SimpleProbe(test))
    for i in range(len(test_data[0])):
        probes.run()
        results = probes.measured()
        for j in range(len(results)):
            target = i if j % 2 == 0 else 4 - i
            assert results[j] == target


class DataGenerator:
    def __init__(self, test_data):
        self.test_data = test_data
        self.index = 0
        self.len = len(test_data)

    def next(self):
        val = self.test_data[self.index]
        self.index = (self.index+1) % self.len
        return val


class Aggregator(ProbeAggregator):
    def __init__(self, test_data):
        assert type(test_data) is list
        assert type(test_data[0]) is list
        names = [f'probe_{i}' for i in range(len(test_data))]
        super().__init__(names)
        self.generators = {f'probe_{i}': DataGenerator(test_data[i]) for i in range(len(test_data))}

    def measure(self):
        for probe in self.probes:
            self.set_value(probe, self.generators[probe].next())


def test_aggregator():
    test_data = [
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0],
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0]
    ]
    probe = Aggregator(test_data)
    for i in range(len(test_data[0])):
        probe.run()
        expected = [test_data[n][i] for n in range(len(test_data))]
        assert probe.get_values() == expected




