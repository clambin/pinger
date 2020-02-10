import subprocess
import shlex
import re
import os
import platform
import logging
from prometheus_client import start_http_server, Gauge


class Reporter:
    def __init__(self, portno=8080):
        self.portno = portno
        self.latency_gauge = Gauge('my_pinger_latency', 'Ping latency', ['host'])
        self.packet_loss_gauge = Gauge('my_pinger_packetloss', 'Packet loss', ['host'])
        start_http_server(self.portno)

    def report(self, host, latency, packet_loss):
        self.latency_gauge.labels(host).set(latency)
        self.packet_loss_gauge.labels(host).set(packet_loss)


class Pinger:
    def __init__(self, host, reporter):
        self.host = host
        self.reporter = reporter
        self.next_seqno = None
        ping = '/bin/ping' if platform.system() == 'Linux' else '/sbin/ping'
        self.proc = subprocess.Popen(shlex.split(f'{ping} {self.host}'), stdout=subprocess.PIPE, encoding='utf-8')

    def is_stopped(self):
        return self.proc.poll() is not None

    def returncode(self):
        return self.proc.returncode if self.is_stopped() else None

    def process(self):
        logging.debug(f'processing {self.host}')
        # FIXME: make this a non-blocking call
        output = self.proc.stdout.readline()
        logging.debug(output)
        for keyword, seqno, latency in re.findall(r' (icmp_seq|seq)=(\d+) .+time=(\d+\.?\d*) ms', output):
            seqno = int(seqno)
            latency = float(latency)
            packet_loss = seqno-self.next_seqno if self.next_seqno else 0
            logging.info(f'{self.host}: {latency} ms, {packet_loss} loss')

            self.reporter.report(self.host, latency, packet_loss)
            self.next_seqno = seqno+1


def test2(hosts):
    pingers = {}

    logging.info(f'Starting with hosts: {hosts}')

    reporter = Reporter(8080)
    for host in hosts:
        try:
            pingers[host] = Pinger(host, reporter)
            logging.debug(f'Started pinger for {host}: {pingers[host]}')
        except Exception as e:
            logging.warning(f'Could not start pinger for {host}: {e}')

    running = list(pingers.keys())
    while running:
        for host in running:
            pingers[host].process()
            if pingers[host].is_stopped():
                logging.warning(f'pinger for {host} exited with {pingers[host].returncode()}')
                del pingers[host]
        running = list(pingers.keys())


if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    logging.info('Starting')
    testhosts = os.environ.get('HOSTS', '192.168.0.1 www.telenet.be 103.22.245.50').split()
    test2(testhosts)
    logging.info('Out of pingers. Shutting down.')
