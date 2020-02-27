import argparse
import os
from pinger import pinger


def test_pinger():
    config = argparse.Namespace(interval=5, port=8080, once=True, logfile='logfile.txt', debug=True,
                                hosts=['localhost'])
    assert pinger(config) == 0
    os.remove('logfile.txt')