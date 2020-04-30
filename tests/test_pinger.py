import argparse
from pinger import pinger, Pinger, initialise


def test_initialise():
    config = argparse.Namespace(interval=0, port=8080,
                                once=True, debug=True,
                                hosts=['localhost', 'www.google.com'])
    probes = initialise(config)
    assert len(probes.probes) == 2
    assert type(probes.probes[0]) is Pinger
    assert type(probes.probes[1]) is Pinger


def test_pinger():
    config = argparse.Namespace(interval=5, port=8080, once=True, debug=True,
                                hosts=['localhost'])
    assert pinger(config) == 0
