# Copyright 2020 by Christophe Lambin
# All rights reserved.

import argparse
import logging
import os
import platform
import re
import time

import version
from metrics import Metric, ProcessMetric, Reporter


class PingMetric(ProcessMetric):
    def __init__(self, host):
        ping = '/bin/ping' if platform.system() == 'Linux' else '/sbin/ping'
        self.host = host
        super().__init__('pinger', 'Pinger', f'{ping} {self.host}')
        self.latency = Metric(f'{self.name}_latency', 'Latency', ['host'], self.host)
        self.packet_loss = Metric(f'{self.name}_packetloss', 'Packet loss', ['host'], self.host)
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
                    logging.debug(f'{self.host}: {latency} ms, {packet_loss} loss')
                    latencies.append(latency)
                    packet_losses.append(packet_loss)
                    self.next_seqno = seqno + 1
            except TypeError:
                logging.warning(f'Cannot parse {line}')
        if not latencies:
            return None, None
        latency = float(sum(latencies)) / len(latencies)
        packet_loss = sum(packet_losses)
        logging.info(f'{self.host}: {latency} ms, {packet_loss} loss')
        return latency, packet_loss

    def report(self, val):
        (latency, packet_loss) = val
        if latency is not None:
            self.latency.report(latency)
        if packet_loss is not None:
            self.packet_loss.report(packet_loss)


def get_config():
    default_interval = 60
    default_port = 8080
    default_host = ['127.0.0.1']

    parser = argparse.ArgumentParser()
    parser.add_argument('--version', action='version', version=f'%(prog)s {version.version}')
    parser.add_argument('--interval', type=int, default=default_interval,
                        help=f'Time between measurements (default: {default_interval} sec)')
    parser.add_argument('--port', type=int, default=default_port,
                        help=f'Prometheus port (default: {default_port})')
    parser.add_argument('--debug', action='store_true',
                        help='Set logging level to debug')
    parser.add_argument('hosts', nargs='*', default=default_host, metavar='host',
                        help='Target host / IP address')
    args = parser.parse_args()
    # env var HOSTS overrides commandline args
    if 'HOSTS' in os.environ:
        args.hosts = os.environ.get('HOSTS').split()
    return args


if __name__ == '__main__':
    config = get_config()
    logging.basicConfig(format='%(asctime)s - %(levelname)s - %(message)s', datefmt='%Y-%m-%d %H:%M:%S',
                        level=logging.DEBUG if config.debug else logging.INFO)
    logging.info(f'Starting.  Configuration: {", ".join([f"{key}={val}" for key, val in vars(config).items()])}')

    r = Reporter.get(config.port)
    for target in config.hosts:
        r.add(PingMetric(target))
    r.start()
    while True:
        r.run()
        time.sleep(config.interval)
