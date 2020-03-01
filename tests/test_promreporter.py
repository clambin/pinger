from metrics.probe import Probes
from metrics.reporter import PrometheusReporter
from tests.probes import SimpleProbe


def test_bad_port():
    reporter = PrometheusReporter(12)
    try:
        reporter.start()
        assert False
    # TODO: what exceptions does start_http_server raise?
    except Exception as err:
        pass


def test_multiple_unlabeled():
    test_data = [
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0],
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0]
    ]
    test_data_count = len(test_data)

    reporter = PrometheusReporter(8081)
    #reporter.start()
    probes = Probes()
    for i in range(test_data_count):
        reporter.add(probes.register(SimpleProbe(test_data[i])),
                     f'test_multiple_unlabeled_{i}', '')
    for i in range(test_data_count):
        probes.run()
        reporter.run()
    assert len(reporter.probes) == test_data_count
    assert len(reporter.gauges) == test_data_count
    for i in range(test_data_count):
        assert list(reporter.gauges.keys())[i] == f'test_multiple_unlabeled_{i}'


def test_multiple_labeled():
    test_data = [
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0],
        [0, 1, 2, 3, 4],
        [4, 3, 2, 1, 0]
    ]
    test_data_count = len(test_data)

    reporter = PrometheusReporter(8081)
    #reporter.start()
    probes = Probes()
    for i in range(test_data_count):
        reporter.add(probes.register(SimpleProbe(test_data[i])),
                     'test_multiple_labeled', '', 'source', f'dest{i}')
    for i in range(len(test_data[0])):
        probes.run()
        reporter.run()
    assert len(reporter.probes) == test_data_count
    assert len(reporter.gauges) == 1
    assert list(reporter.gauges.keys())[0] == "test_multiple_labeled|source"

