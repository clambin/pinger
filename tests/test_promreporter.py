from metrics.reporter import PrometheusReporter


def test_bad_port():
    reporter = PrometheusReporter(12)
    try:
        reporter.start()
        assert False
    # TODO: what exceptions does start_http_server raise?
    except Exception as err:
        pass

