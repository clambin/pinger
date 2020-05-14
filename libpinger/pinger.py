import logging
import platform
import re
import time

from prometheus_client import Gauge, start_http_server
from libpinger.pingtracker import PingTracker
from libpinger.configuration import print_configuration
from pimetrics.probe import ProcessProbe, Probes
from libpinger import version

GAUGES = {
    'packet_loss': Gauge('pinger_packet_loss', 'Network Packet Loss', ['host']),
    'latency': Gauge('pinger_latency', 'Network Latency', ['host']),
}


class Pinger(ProcessProbe, PingTracker):
    def __init__(self, host):
        ping = '/bin/ping' if platform.system() == 'Linux' else '/sbin/ping'
        self.host = host
        ProcessProbe.__init__(self, f'{ping} {self.host}')
        PingTracker.__init__(self)

    def report(self, output):
        super().report(output)
        packet_loss, latency = output[0], output[1]
        if packet_loss is not None:
            GAUGES['packet_loss'].labels(self.host).set(packet_loss)
        if latency is not None:
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
