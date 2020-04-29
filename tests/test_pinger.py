import argparse
import os
import pytest
from pinger import pinger, get_configuration, str2bool, Pinger, initialise


def test_str2bool():
    assert str2bool(True) is True
    for arg in ['yes', 'true', 't', 'y', '1', 'on']:
        assert str2bool(arg) is True
    for arg in ['no', 'false', 'f', 'n', '0', 'off']:
        assert str2bool(arg) is False
    with pytest.raises(argparse.ArgumentTypeError) as e:
        assert str2bool('maybe')
    assert str(e.value) == 'Boolean value expected.'


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
    assert config.debug is False
    assert config.interval == 5
    assert config.logfile == 'logfile.csv'
    assert config.once is False
    assert config.port == 8080
    assert config.hosts == ['localhost']


def test_config_envvar_override():
    args = ['localhost']
    os.environ['HOSTS'] = 'www.google.com'
    config = get_configuration(args)
    assert config.hosts == ['www.google.com']


def test_initialise():
    config = argparse.Namespace(interval=0, port=8080,
                                once=True, debug=True,
                                hosts=['localhost', 'www.google.com'])
    probes = initialise(config)
    assert len(probes.probes) == 2
    assert type(probes.probes[0]) is Pinger
    assert type(probes.probes[1]) is Pinger


def test_pinger():
    config = argparse.Namespace(interval=3, port=8080, once=True, debug=True,
                                hosts=['localhost'])
    assert pinger(config) == 0
