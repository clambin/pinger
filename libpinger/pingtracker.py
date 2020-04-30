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
            logging.debug(f'Latencies: {self.latencies}')
            return round(sum(self.latencies) / len(self.latencies), 1)

        def calculate_packet_loss():
            def process_range(series):
                gap = 0
                logging.debug(series)
                # calculate the gap between each (ordered) packet
                gaps = [series[i+1]-series[i]-1 for i in range(len(series)-1)]
                logging.debug(f'gaps: {gaps}')
                gap += sum(gaps)
                # any packets lost between now and the previous batch?
                if self.next_sequence_nr is not None:
                    gap += series[0] - self.next_sequence_nr
                logging.debug(f'Final gap: {gap}')
                return gap
            if not self.sequence_nrs:
                return None
            # sort the sequence nrs and remove all duplicates
            packets = sorted(set(self.sequence_nrs))
            logging.debug(f'Sequence Nrs received: {packets}')
            # if it's the first call, safe to assume the smallest nr is next expected
            if self.next_sequence_nr is None:
                self.next_sequence_nr = packets[0]
            # sequence numbers can wrap around!
            # In this case, we'd get something like [ 0, 1, 2, 3, 65534, 65535 ]
            # split into two series [ 65534, 65535 ] and [ 0, 1, 2 ] using next_sequence_nr as a boundary
            # process the larger range first (pre-wrap) and then the smaller one (post-wrap)
            loss = 0
            larger = list(filter(lambda i: i >= self.next_sequence_nr, packets))
            smaller = list(filter(lambda i: i < self.next_sequence_nr, packets))
            if larger:
                loss += process_range(larger)
                self.next_sequence_nr = larger[-1] + 1
            if smaller:
                self.next_sequence_nr = 0
                loss += process_range(smaller)
                self.next_sequence_nr = smaller[-1] + 1
            return loss
        latency = calculate_latencies()
        packet_loss = calculate_packet_loss()
        # set up next cycle
        self.sequence_nrs = []
        self.latencies = []
        return packet_loss, latency
