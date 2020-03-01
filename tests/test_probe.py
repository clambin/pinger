import os
from metrics.probe import Probe, FileProbe, ProcessProbe, Probes
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
    bad_file = False
    try:
        FileProbe('testfile.txt')
    except FileNotFoundError:
        bad_file = True
    assert bad_file


def test_process():
    probe = SimpleProcessProbe('/bin/sh -c ./process_ut.sh')
    out = 0
    while probe.running():
        probe.run()
        out += probe.measured()
    assert out == 55


def test_bad_process():
    bad_file = False
    try:
        SimpleProcessProbe('missing_process_ut.sh')
    except FileNotFoundError:
        bad_file = True
    assert bad_file


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
