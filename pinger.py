# Copyright 2020 by Christophe Lambin
# All rights reserved.

import re
import os
import time
import platform
import logging

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
        latency = float(sum(latencies))/len(latencies)
        packet_loss = sum(packet_losses)
        logging.info(f'{self.host}: {latency} ms, {packet_loss} loss')
        return latency, packet_loss

    def report(self, val):
        (latency, packet_loss) = val
        if latency is not None: self.latency.report(latency)
        if packet_loss is not None: self.packet_loss.report(packet_loss)


if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    # TODO: add parameters for configuration
    r = Reporter.get(8080)
    for target in os.environ.get('HOSTS', '192.168.0.1').split():
        r.add(PingMetric(target))
    r.start()
    while True:
        r.run()
        time.sleep(1)
