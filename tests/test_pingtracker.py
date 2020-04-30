from libpinger.pingtracker import PingTracker


def test_latency():
    tracker = PingTracker()
    # no data
    assert tracker.calculate() == (None, None)
    # simple test
    tracker.track(1, 6)
    tracker.track(2, 5)
    tracker.track(3, 1)
    assert tracker.calculate() == (0, 4)
    # reset stats after each call to calculate
    tracker.track(4, 90)
    tracker.track(5, 110)
    assert tracker.calculate() == (0, 100)


def test_loss():
    tracker = PingTracker()
    # no data
    assert tracker.calculate() == (None, None)
    # first packet shouldn't be used to calculate packet loss
    tracker.track(1, 0)
    assert tracker.calculate() == (0, 0)
    # zero packet loss
    tracker.track(2, 0)
    tracker.track(3, 0)
    assert tracker.calculate() == (0, 0)
    # lose one packet
    tracker.track(5, 0)
    assert tracker.calculate() == (1, 0)
    # reset stats after every call to calculate
    tracker.track(6, 0)
    assert tracker.calculate() == (0, 0)
    # lose a bunch of data
    tracker.track(16, 0)
    assert tracker.calculate() == (9, 0)


def test_duplicates():
    tracker = PingTracker()
    tracker.track(500, 0)
    tracker.track(501, 0)
    tracker.track(501, 0)
    assert tracker.calculate() == (0, 0)


def test_wrap():
    tracker = PingTracker()
    tracker.track(500, 0)
    tracker.track(501, 0)
    tracker.track(502, 0)
    assert tracker.calculate() == (0, 0)
    tracker.track(0, 0)
    tracker.track(1, 0)
    tracker.track(2, 0)
    assert tracker.calculate() == (0, 0)
    tracker.track(499, 0)
    assert tracker.calculate()  # reset tracking
    tracker.track(500, 0)
    tracker.track(501, 0)
    tracker.track(502, 0)
    assert tracker.calculate() == (0, 0)
    tracker.track(1, 0)
    tracker.track(2, 0)
    assert tracker.calculate() == (1, 0)


def test_out_of_order():
    tracker = PingTracker()
    # packets can arrive out of order
    tracker.track(1, 0)
    tracker.track(4, 0)
    tracker.track(2, 0)
    tracker.track(3, 0)
    assert tracker.calculate() == (0, 0)
    # the next expected seqno is remembered
    tracker.track(6, 0)
    tracker.track(9, 0)
    tracker.track(7, 0)
    tracker.track(10, 0)
    assert tracker.calculate() == (2, 0)
    # Duplicates are ignored
    tracker.track(11, 0)
    tracker.track(12, 0)
    tracker.track(12, 0)
    tracker.track(13, 0)
    assert tracker.calculate() == (0, 0)
