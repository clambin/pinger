from metrics.probe import Probe


class SimpleProbe(Probe):
    def __init__(self, test_sequence):
        super().__init__()
        self.test_sequence = test_sequence
        self.index = 0
        self.value = None

    def measure(self):
        self.value = self.test_sequence[self.index]
        self.index += 1
        if self.index >= len(self.test_sequence):
            self.index = 0

    def measured(self):
        return self.value

