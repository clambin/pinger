from metrics import BaseFactory, Reporter, Metric

class TestGauge:
    def __init__(self, name, description, label=None, key=None):
        self.name = name
        self.description = description

    def set(self, val):
        self.val = val

    def get(self):
        return self.val


class TestFactory(BaseFactory):
    @staticmethod
    def gauge(name, description, label):
        return TestGauge(name, description, label)


class TestMetric(Metric):
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
    r = Reporter.get(8080, TestFactory)
    series = [1, 2, 3, 4]
    t = TestMetric('foo', 'bar', series)
    r.add(t)
    for i in range(0, len(series)):
        r.run()
        assert t.check()


def test_multiple_simple_metric():
    r = Reporter.get(8080, TestFactory)
    data = [[1, 2, 3, 4], [5, 6, 7, 8], [9, 8, 7, 6], [5, 4, 3, 2]]
    metrics = []
    for i in range(0, len(data)):
        t = TestMetric(f'foo_{i}', '', data[i])
        r.add(t)
        metrics.append(t)
    for i in range(0, len(data[0])):
        r.run()
        for t in metrics:
            assert t.check()





