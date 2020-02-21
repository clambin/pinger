import logging

from metrics.probe import Probe, Probes
from metrics.reporter import Reporter, Reporters


class SimpleProbe(Probe):
    def __init__(self, test_sequence):
        super().__init__()
        self.test_sequence = test_sequence
        self.index = 0

    def measure(self):
        val = self.test_sequence[self.index]
        self.index += 1
        if self.index >= len(self.test_sequence):
            self.index = 0
        return val


class UnittestReporter(Reporter):
    def __init__(self):
        super().__init__()
        self.last = {}

    def report(self, probe, val):
        self.last[probe] = val

    def measured(self, probe):
        return self.last[probe]


def test_single():
    reporter = UnittestReporter()
    probe = SimpleProbe([1, 2, 3, 4])
    reporter.add(probe, 'test', '')
    for i in range(1, 4):
        probe.run()
        reporter.run()
        assert reporter.measured(probe) == i


def test_multiple():
    reporter = UnittestReporter()
    probes = [
        SimpleProbe([0, 1, 2, 3, 4]),
        SimpleProbe([4, 3, 2, 1, 0]),
        SimpleProbe([0, 1, 2, 3, 4]),
        SimpleProbe([4, 3, 2, 1, 0])
    ]
    for i in range(len(probes)):
        reporter.add(probes[i], f'test{i}', '')
    for i in range(5):
        for p in probes: p.run()
        reporter.run()
        for j in range(len(probes)):
            target = i if j % 2 == 0 else 4 - i
            assert reporter.measured(probes[j]) == target


def test_single_labeled():
    reporter = UnittestReporter()
    probe = SimpleProbe([1, 2, 3, 4])
    reporter.add(probe, 'test', '', 'source', 'dest')
    for i in range(1, 4):
        probe.run()
        reporter.run()
        assert reporter.measured(probe) == i


def test_multiple_labeled():
    reporter = UnittestReporter()
    probes = [
        SimpleProbe([0, 1, 2, 3, 4]),
        SimpleProbe([4, 3, 2, 1, 0]),
        SimpleProbe([0, 1, 2, 3, 4]),
        SimpleProbe([4, 3, 2, 1, 0])
    ]
    for i in range(len(probes)):
        reporter.add(probes[i], 'test', '', 'source', f'dest{i}')
    for i in range(5):
        for p in probes: p.run()
        reporter.run()
        for j in range(len(probes)):
            target = i if j % 2 == 0 else 4 - i
            assert reporter.measured(probes[j]) == target


def test_duplicates():
    reporter = UnittestReporter()
    try:
        reporter.add(SimpleProbe([0]), 'test', '', 'source', 'dest')
    except KeyError:
        assert False
    try:
        reporter.add(SimpleProbe([0]), 'test', '', 'source', 'dest')
        assert False
    except KeyError:
        pass


def test_reporters():
    probes = Probes()
    reporters = Reporters()
    reporters.register(UnittestReporter())
    reporters.register(UnittestReporter())
    reporters.add(probes.register(SimpleProbe([0, 1, 2, 3, 4, 5])), 'test', '')

    assert len(reporters.reporters) == 2
    assert len(reporters.reporters[0].probes.keys()) == 1
    assert len(reporters.reporters[1].probes.keys()) == 1
    p1 = list(reporters.reporters[0].probes.keys())[0]
    p2 = list(reporters.reporters[1].probes.keys())[0]
    assert p1 == p2

    # doesn't really test reporters.run() but checks if we haven't broken the API again
    for i in range(6):
        probes.run()
        reporters.run()
        results = probes.measured()
        assert (len(results) == 1)
        assert results[0] == i
