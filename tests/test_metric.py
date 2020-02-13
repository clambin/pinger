import os
from metrics import BaseFactory, Reporter, Metric, FileMetric

class UnittestGauge:
    # TODO: supporting labels is non-trivial. worth the effort?
    def __init__(self, name, description, label=None, key=None):
        self.name = name
        self.description = description
        self.val = None

    def set(self, val):
        self.val = val

    def get(self):
        return self.val


class UnittestFactory(BaseFactory):
    @staticmethod
    def gauge(name, description, label):
        return UnittestGauge(name, description, label)


class UnittestMetric(Metric):
    def __init__(self, name, description, testdata):
        super().__init__(name, description)
        self.series = testdata
        self.index = -1

    def measure(self):
        self.index += 1
        if self.index == len(self.series): self.index = 0
        return self.series[self.index]

    def check(self):
        return self.gauge.get() == self.series[self.index]


def test_single_simple_metric():
    r = Reporter.get(8080, UnittestFactory)
    series = [1, 2, 3, 4]
    t = UnittestMetric('foo', 'bar', series)
    r.add(t)
    for i in range(0, len(series)):
        r.run()
        assert t.check()


def test_multiple_simple_metric():
    r = Reporter.get(8080, UnittestFactory)
    data = [[1, 2, 3, 4], [5, 6, 7, 8], [9, 8, 7, 6], [5, 4, 3, 2]]
    metrics = []
    for i in range(0, len(data)):
        t = UnittestMetric(f'foo_{i}', '', data[i])
        r.add(t)
        metrics.append(t)
    for i in range(0, len(data[0])):
        r.run()
        for t in metrics:
            assert t.check()


def test_filemetric():
    r = Reporter.get(808, UnittestFactory)
    r.add(FileMetric('file_metric', '', 'testfile.txt'))
    data = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    for val in data:
        with open('testfile.txt', 'w') as f:
            f.write(f'{val}')
        r.run()
        assert r.gauges['file_metric'].get() == val
    os.remove('testfile.txt')


