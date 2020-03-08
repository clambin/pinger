# Copyright 2020 by Christophe Lambin
# All rights reserved.

import argparse
import logging
import os
import platform
import re
import time

import version
from metrics.probe import ProcessProbe, Probes, ProbeAggregator
from metrics.reporter import Reporters, PrometheusReporter, FileReporter


class PingTracker:
    def __init__(self):
        self.latencies = []
        self.packet_losses = []
        self.next_seqno = None

    def track(self, seqno, latency):
        loss = 0 if self.next_seqno is None else seqno - self.next_seqno
        self.packet_losses.append(loss)
        self.latencies.append(latency)
        self.next_seqno = seqno + 1

    def calculate(self):
        if not self.latencies:
            return None, None
        packet_loss = sum(self.packet_losses)
        latency = round(sum(self.latencies) / len(self.latencies), 1)
        self.packet_losses = []
        self.latencies = []
        return packet_loss, latency


class Pinger(ProcessProbe, ProbeAggregator, PingTracker):
    def __init__(self, host):
        ping = '/bin/ping' if platform.system() == 'Linux' else '/sbin/ping'
        self.host = host
        ProcessProbe.__init__(self, f'{ping} {self.host}')
        ProbeAggregator.__init__(self, ['latency', 'packet_loss'])
        PingTracker.__init__(self)

    def process(self, lines):
        for line in lines:
            try:
                for keyword, seqno, latency in re.findall(r' (icmp_seq|seq)=(\d+) .+time=(\d+\.?\d*) ms', line):
                    self.track(int(seqno), float(latency))
            except TypeError:
                logging.warning(f'Cannot parse {line}')
        packet_loss, latency = self.calculate()
        logging.debug(f'{self.host}: {latency} ms, {packet_loss} loss')
        self.set_value('latency', latency)
        self.set_value('packet_loss', packet_loss)


def str2bool(v):
    if isinstance(v, bool):
        return v
    if v.lower() in ('yes', 'true', 't', 'y', '1', 'on'):
        return True
    elif v.lower() in ('no', 'false', 'f', 'n', '0', 'off'):
        return False
    else:
        raise argparse.ArgumentTypeError('Boolean value expected.')


def get_configuration(args=None):
    default_interval = 5
    default_port = 8080
    default_host = ['127.0.0.1']
    default_log = 'logfile.csv'

    parser = argparse.ArgumentParser()
    parser.add_argument('--version', action='version', version=f'%(prog)s {version.version}')
    parser.add_argument('--interval', type=int, default=default_interval,
                        help=f'Time between measurements (default: {default_interval} sec)')
    parser.add_argument('--once', action='store_true',
                        help='Measure once and then terminate')
    parser.add_argument('--debug', action='store_true',
                        help='Set logging level to debug')
    # Reporters
    parser.add_argument('--reporter-prometheus', type=str2bool, nargs='?', default=True,
                        help='Report metrics to Prometheus')
    parser.add_argument('--port', type=int, default=default_port,
                        help=f'Prometheus port (default: {default_port})')
    parser.add_argument('--reporter-logfile', type=str2bool, nargs='?', default=False,
                        help='Report metrics to a CSV logfile')
    parser.add_argument('--logfile', action='store', default=default_log,
                        help=f'metrics output logfile (default: {default_log})')
    # Hosts to ping
    parser.add_argument('hosts', nargs='*', default=default_host, metavar='host',
                        help='Target host / IP address')
    args = parser.parse_args(args)
    # env var HOSTS overrides commandline args
    if 'HOSTS' in os.environ:
        args.hosts = os.environ.get('HOSTS').split()
    return args


def print_configuration(config):
    return ', '.join([f'{key}={val}' for key, val in vars(config).items()])


def initialise(config):
    reporters = Reporters()
    probes = Probes()

    # Reporters
    if config.reporter_prometheus:
        reporters.register(PrometheusReporter(config.port))
    if config.reporter_logfile:
        reporters.register(FileReporter(config.logfile))
    if not config.reporter_prometheus and not config.reporter_logfile:
        logging.warning('No reporters configured')

    # Ideally this should be done after initialise() but since we can only register prometheus metrics
    # once (limiting what we can cover in unit testing), we do it here.
    try:
        reporters.start()
    except Exception as err:
        logging.fatal(f"Could not start prometheus client on port {config.port}: {err}")
        raise RuntimeError

    # Probes
    for target in config.hosts:
        ping = probes.register(Pinger(target))
        reporters.add(ping.get_probe('latency'), 'pinger_latency', 'Latency', 'host', target)
        reporters.add(ping.get_probe('packet_loss'), 'pinger_packet_loss', 'Latency', 'host', target)

    return probes, reporters


def pinger(config):
    logging.basicConfig(format='%(asctime)s - %(levelname)s - %(message)s', datefmt='%Y-%m-%d %H:%M:%S',
                        level=logging.DEBUG if config.debug else logging.INFO)
    logging.info(f'Starting pinger v{version.version}')
    logging.info(f'Configuration: {print_configuration(config)}')

    try:
        probes, reporters = initialise(config)
    except RuntimeError:
        return 1

    while True:
        time.sleep(config.interval)
        probes.run()
        reporters.run()
        if config.once:
            break
    return 0


if __name__ == '__main__':
    pinger(get_configuration())
