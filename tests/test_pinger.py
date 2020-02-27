import argparse
import os
from pinger import pinger, get_configuration


def test_pinger():
    config = argparse.Namespace(interval=5, port=8080, once=True, logfile='logfile.txt', debug=True,
                                hosts=['localhost'])
    assert pinger(config) == 0
    os.remove('logfile.txt')


def test_get_config():
    args = '--interval 25 --port 1234 --logfile log.txt --once --debug localhost'.split()
    config = get_configuration(args)
    assert config.interval == 25
    assert config.port == 1234
    assert config.logfile == 'log.txt'
    assert config.once
    assert config.debug
    assert config.hosts == ['localhost']


def test_default_config():
    args = ['localhost']
    config = get_configuration(args)
    assert config.interval == 60
    assert config.port == 8080
    assert config.logfile is None
    assert config.once is False
    assert config.debug is False
    assert config.hosts == ['localhost']


def test_config_envvar_override():
    args = ['localhost']
    os.environ['HOSTS'] = 'www.google.com'
    config = get_configuration(args)
    assert config.hosts == ['www.google.com']
