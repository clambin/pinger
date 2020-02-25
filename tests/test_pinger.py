import argparse
from pinger import pinger


def test_pinger():
    config = argparse.Namespace(interval=5, port=8080, once=True, logfile=None, debug=True, hosts=['localhost'])
    if pinger(config) != 0:
        assert False
