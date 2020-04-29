# Copyright 2020 by Christophe Lambin
# All rights reserved.

import argparse
import logging
import os
import platform
import re
import time

from prometheus_client import Gauge, start_http_server

from pimetrics.probe import ProcessProbe, Probes
import version


GAUGES = {
    'packet_loss': Gauge('pinger_packet_loss', 'Network Packet Loss', ['host']),
    'latency': Gauge('pinger_latency', 'Network Latency', ['host']),
}


class PingTracker:
    wrap_gap = 65000

    def __init__(self):
        self.latencies = []
        self.seqnos = []
        self.next_seqno = None

    def track(self, seqno, latency):
        self.seqnos.append(seqno)
        self.latencies.append(latency)

    def calculate(self):
        if not self.latencies:
            return None, None
        # Average latency for all packets received
        latency = round(sum(self.latencies) / len(self.latencies), 1)
        # Packet loss:  check gaps between sequence numbers received, \
        # starting with the last packet on the previous run
        if self.next_seqno is not None:
            self.seqnos.insert(0, self.next_seqno)
        # remove any duplicates
        packets = sorted(set(self.seqnos))
        # calculate the gaps between the ordered packets
        gaps = [packets[i+1]-packets[i]-1 for i in range(len(packets)-1)]
        # if the seqno wrapped around, one of the gaps will be *very* large
        gaps = list(filter(lambda x: x < PingTracker.wrap_gap, gaps))
        # packet loss is not just the sum of the gaps
        packet_loss = sum(gaps)
        # set up next track/calculate cycle
        self.next_seqno = 1 + self.seqnos[-1]
        self.seqnos = []
        self.latencies = []
        return packet_loss, latency


class Pinger(ProcessProbe, PingTracker):
    def __init__(self, host):
        ping = '/bin/ping' if platform.system() == 'Linux' else '/sbin/ping'
        self.host = host
        ProcessProbe.__init__(self, f'{ping} {self.host}')
        PingTracker.__init__(self)

    def report(self, output):
        super().report(output)
        logging.debug(output)
        if output == (None, None):
            logging.warning('No output received')
        else:
            packet_loss = output[0]
            latency = output[1]
            GAUGES['packet_loss'].labels(self.host).set(packet_loss)
            GAUGES['latency'].labels(self.host).set(latency)

    def process(self, lines):
        for line in lines:
            try:
                for keyword, seqno, latency in re.findall(r' (icmp_seq|seq)=(\d+) .+time=(\d+\.?\d*) ms', line):
                    self.track(int(seqno), float(latency))
            except TypeError:
                logging.warning(f'Cannot parse {line}')
        packet_loss, latency = self.calculate()
        logging.debug(f'{self.host}: {latency} ms, {packet_loss} loss')
        return packet_loss, latency


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
    probes = Probes()
    for target in config.hosts:
        probes.register(Pinger(target))
    return probes


def pinger(config):
    logging.basicConfig(format='%(asctime)s - %(levelname)s - %(message)s', datefmt='%Y-%m-%d %H:%M:%S',
                        level=logging.DEBUG if config.debug else logging.INFO)
    logging.info(f'Starting pinger v{version.version}')
    logging.info(f'Configuration: {print_configuration(config)}')

    start_http_server(config.port)

    try:
        probes = initialise(config)
    except RuntimeError:
        return 1

    while True:
        time.sleep(config.interval)
        probes.run()
        if config.once:
            break
    return 0


if __name__ == '__main__':
    pinger(get_configuration())
