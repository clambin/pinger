import os
from metrics import Metric, FileMetric, ProcessMetric, Reporter


class UnittestMetric(Metric):
    def __init__(self, name, description, testdata):
        super().__init__(name, description)
        self.testdata = testdata
        self.index = -1
        self.last = None

    def report(self, val):
        self.last = val

    def measure(self):
        self.index += 1
        if self.index == len(self.testdata): self.index = 0
        return self.testdata[self.index]

    def check(self):
        return self.last


class UnittestFileMetric(FileMetric):
    def __init__(self, name, description, filename):
        super().__init__(name, description, filename)
        self.last = None

    def report(self, val):
        self.last = val

    def check(self):
        return self.last


class UnittestProcessMetric(ProcessMetric):
    def __init__(self, name, description, command):
        super().__init__(name, description, command)
        self.out = 0

    def process(self, lines):
        val = 0
        for line in lines:
            val += int(line)
        return val

    def report(self, val):
        self.out += val

    def check(self):
        out = self.out
        self.out = 0
        return out


def test_metric():
    testdata = [1, 2, 3, 4]
    metric = UnittestMetric('foo', 'bar', testdata)
    for val in testdata:
        metric.run()
        assert metric.check() == val


def test_file_metric():
    # create the file
    open('testfile.txt', 'w')
    metric = UnittestFileMetric('file_metric', '', 'testfile.txt')
    for val in range(1, 10):
        with open('testfile.txt', 'w') as f:
            f.write(f'{val}')
        metric.run()
        assert metric.check() == val
    os.remove('testfile.txt')


def test_bad_file_metric():
    bad_file = False
    try:
        UnittestFileMetric('file_metric', '', 'testfile.txt')
    except FileNotFoundError:
        bad_file = True
    assert bad_file


def test_process_metric():
    metric = UnittestProcessMetric('process_metric', '', '/bin/sh -c ./process_ut.sh')
    while metric.running():
        metric.run()
    assert metric.check() == 55


def test_bad_process_metric():
    bad_file = False
    try:
        UnittestProcessMetric('process_metric', '', 'missing_process_ut.sh')
    except FileNotFoundError:
        bad_file = True
    assert bad_file


def test_reporter():
    r = Reporter(8080)
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

