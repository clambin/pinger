import logging


class PingTracker:
    def __init__(self):
        self.next_sequence_nr = None
        self.sequence_nrs = []
        self.latencies = []

    def track(self, sequence_nr, latency):
        self.sequence_nrs.append(sequence_nr)
        self.latencies.append(latency)

    def calculate(self):
        def calculate_latencies():
            if not self.latencies:
                return None
            return round(sum(self.latencies) / len(self.latencies), 1)

        def calculate_packet_loss():
            def process_range(series):
                gap = 0
                # calculate the gap between each (ordered) packet
                gaps = [series[i+1]-series[i]-1 for i in range(len(series)-1)]
                gap += sum(gaps)
                # any packets lost between now and the previous batch?
                if self.next_sequence_nr is not None:
                    gap += series[0] - self.next_sequence_nr
                return gap
            if not self.sequence_nrs:
                return None
            # sort the sequence nrs and remove all duplicates
            packets = sorted(set(self.sequence_nrs))
            # if it's the first call, safe to assume the smallest nr is next expected
            if self.next_sequence_nr is None:
                self.next_sequence_nr = packets[0]
            # sequence numbers can wrap around!
            # In this case, we'd get something like [ 0, 1, 2, 3, 65534, 65535 ]
            # split into two series [ 65534, 65535 ] and [ 0, 1, 2 ] using next_sequence_nr as a boundary
            # process the higher range first (pre-wrap) and then the lower one (post-wrap)
            loss = 0
            higher = list(filter(lambda i: i >= self.next_sequence_nr, packets))
            lower = list(filter(lambda i: i < self.next_sequence_nr, packets))
            if higher:
                loss += process_range(higher)
                self.next_sequence_nr = higher[-1] + 1
            if lower:
                logging.info(f'seqno\'s wrapped: expected {self.next_sequence_nr} but received {lower[0]}')
                self.next_sequence_nr = 0
                loss += process_range(lower)
                self.next_sequence_nr = lower[-1] + 1
            return loss
        latency = calculate_latencies()
        packet_loss = calculate_packet_loss()
        # set up next cycle
        self.sequence_nrs = []
        self.latencies = []
        return packet_loss, latency
