# Copyright 2020 by Christophe Lambin
# All rights reserved.

import argparse
import logging
import os
import platform
import re
import time

from metrics.probe import Probe, ProcessProbe, Probes
from metrics.reporter import Reporters, PrometheusReporter, FileReporter

import version


class LatencyProbe(Probe):
    def __init__(self, pinger_probe):
        super().__init__()
        self.pinger = pinger_probe
        pass

    def measure(self):
        return self.pinger.val[0] if self.pinger.val is not None else None


class PacketLossProbe(Probe):
    def __init__(self, pinger_probe):
        super().__init__()
        self.pinger = pinger_probe

    def measure(self):
        return self.pinger.val[1] if self.pinger.val is not None else None


class PingProbe(ProcessProbe):
    def __init__(self, host):
        ping = '/bin/ping' if platform.system() == 'Linux' else '/sbin/ping'
        self.host = host
        super().__init__(f'{ping} {self.host}')
        self.next_seqno = None

    def __str__(self):
        return self.host

    def process(self, lines):
        latencies = []
        packet_losses = []
        for line in lines:
            try:
                for keyword, seqno, latency in re.findall(r' (icmp_seq|seq)=(\d+) .+time=(\d+\.?\d*) ms', line):
                    seqno, latency = int(seqno), float(latency)
                    packet_loss = seqno - self.next_seqno if self.next_seqno else 0
                    latencies.append(latency)
                    packet_losses.append(packet_loss)
                    self.next_seqno = seqno + 1
            except TypeError:
                logging.warning(f'Cannot parse {line}')
        if not latencies:
            return None, None
        latency = round(sum(latencies) / len(latencies), 1)
        packet_loss = sum(packet_losses)
        logging.info(f'{self.host}: {latency} ms, {packet_loss} loss')
        return latency, packet_loss


def get_configuration():
    default_interval = 60
    default_port = 8080
    default_host = ['127.0.0.1']
    default_log = None

    parser = argparse.ArgumentParser()
    parser.add_argument('--version', action='version', version=f'%(prog)s {version.version}')
    parser.add_argument('--interval', type=int, default=default_interval,
                        help=f'Time between measurements (default: {default_interval} sec)')
    parser.add_argument('--port', type=int, default=default_port,
                        help=f'Prometheus port (default: {default_port})')
    parser.add_argument('--logfile', action='store', default=default_log,
                        help=f'metrics output logfile (default: {default_log})')
    parser.add_argument('--once', action='store_true',
                        help='Measure once and then terminate')
    parser.add_argument('--debug', action='store_true',
                        help='Set logging level to debug')
    parser.add_argument('hosts', nargs='*', default=default_host, metavar='host',
                        help='Target host / IP address')
    args = parser.parse_args()
    # env var HOSTS overrides commandline args
    if 'HOSTS' in os.environ:
        args.hosts = os.environ.get('HOSTS').split()
    return args


def print_configuration(config):
    return ', '.join([f'{key}={val}' for key, val in vars(config).items()])


def pinger(config):
    logging.basicConfig(format='%(asctime)s - %(levelname)s - %(message)s', datefmt='%Y-%m-%d %H:%M:%S',
                        level=logging.DEBUG if config.debug else logging.INFO)
    logging.info(f'Starting pinger v{version.version}')
    logging.info(f'Configuration: {print_configuration(config)}')

    reporters = Reporters()
    probes = Probes()

    if config.port:
        reporters.register(PrometheusReporter(config.port))
    if config.logfile:
        reporters.register(FileReporter(config.logfile))

    for target in config.hosts:
        # reporters only deal with one value per probe. PingProbe measures two (latency & packet loss)
        # so we don't add PingProbe itself to the reporter
        # instead we add one dependent probe for each value to measure
        ping = probes.register(PingProbe(target))
        reporters.add(probes.register(LatencyProbe(ping)),
                      'pinger_latency', 'Latency', 'host', target)
        reporters.add(probes.register(PacketLossProbe(ping)),
                      'pinger_packet_loss', 'Latency', 'host', target)
    try:
        reporters.start()
    except OSError as err:
        print(f"Could not start prometheus client on port {config.port}: {err}")
        return 1

    while True:
        probes.run()
        reporters.run()
        if config.once:
            break
        time.sleep(config.interval)
    return 0


if __name__ == '__main__':
    pinger(get_configuration())
