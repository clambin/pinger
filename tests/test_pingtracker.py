from pinger import PingTracker


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


