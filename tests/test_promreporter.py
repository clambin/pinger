from metrics.reporter import PrometheusReporter


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

