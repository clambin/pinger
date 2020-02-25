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

