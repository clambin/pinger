from metrics.reporter import PrometheusReporter


class SimpleProbe:
    def __init__(self, test_sequence):
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


class UnittestReporter(PrometheusReporter):
    def __init__(self, port=8080):
        super().__init__(port)
        self.last = {}

    def report(self, probe, val):
        super().report(probe, val)
        self.last[probe] = val

    def measured(self, probe):
        return self.last[probe]


def test_single():
    reporter = UnittestReporter()
    probe = SimpleProbe([1, 2, 3, 4])
    reporter.add(probe, 'test_single', '')
    for i in range(1, 4):
        probe.measure()
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
        reporter.add(probes[i], f'test_multiple_{i}', '')
    for i in range(5):
        for p in probes: p.measure()
        reporter.run()
        for j in range(len(probes)):
            target = i if j % 2 == 0 else 4 - i
            assert reporter.measured(probes[j]) == target


def test_single_labeled():
    reporter = UnittestReporter()
    probe = SimpleProbe([1, 2, 3, 4])
    reporter.add(probe, 'test_single_labeled', '', 'source', 'dest')
    for i in range(1, 4):
        probe.measure()
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
        reporter.add(probes[i], 'test_multiple_labeled', '', 'source', f'dest{i}')
    for i in range(5):
        for p in probes: p.measure()
        reporter.run()
        for j in range(len(probes)):
            target = i if j % 2 == 0 else 4 - i
            assert reporter.measured(probes[j]) == target


def test_duplicates():
    reporter = UnittestReporter()
    try:
        reporter.add(SimpleProbe([0]), 'test_duplicates', '', 'source', 'dest')
    except KeyError:
        assert False
    try:
        reporter.add(SimpleProbe([0]), 'test_duplicates', '', 'source', 'dest')
        assert False
    except KeyError:
        pass


def test_bad_port():
    reporter = UnittestReporter(12)
    try:
        reporter.start()
        assert False
    except OSError as err:
        pass
    # TODO: what exceptions does start_http_server raise?
    except Exception as err:
        pass

