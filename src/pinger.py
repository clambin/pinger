import logging
import platform
import re
import time

from prometheus_client import Counter, start_http_server
from src.pingtracker import PingTracker
from src.configuration import print_configuration
from pimetrics.probe import ProcessProbe, Probes
from src import version

COUNTERS = {
    'packet_total': Counter('pinger_packet', 'Network Packet Total', ['host']),
    'packet_latency': Counter('pinger_packet_latency', 'Network Packet Latency Total', ['host']),
    'packet_loss': Counter('pinger_packet_loss', 'Network Packet Loss Total', ['host']),
}


class Pinger(ProcessProbe, PingTracker):
    def __init__(self, host):
        ping = '/bin/ping' if platform.system() == 'Linux' else '/sbin/ping'
        self.host = host
        ProcessProbe.__init__(self, f'{ping} {self.host}')
        PingTracker.__init__(self)

    def report(self, output):
        packet_count, packet_latency, packet_loss = output[0], output[1], output[2]
        if packet_count:
            COUNTERS['packet_total'].labels(self.host).inc(packet_count)
            COUNTERS['packet_latency'].labels(self.host).inc(packet_latency)
            COUNTERS['packet_loss'].labels(self.host).inc(packet_loss)

    def process(self, lines):
        for line in lines:
            try:
                for keyword, seqno, latency in re.findall(r' (icmp_seq|seq)=(\d+) .+time=(\d+\.?\d*) ms', line):
                    self.track(int(seqno), float(latency))
            except TypeError:
                logging.warning(f'Cannot parse {line}')
        packet_count, packet_latency, packet_loss = self.calculate()
        logging.debug(f'{self.host}: {packet_count} packets, total latency: {packet_latency} ms, {packet_loss} loss')
        return packet_count, packet_latency, packet_loss


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
    probes = initialise(config)
    while True:
        time.sleep(config.interval)
        probes.run()
        if config.once:
            break
    return 0
