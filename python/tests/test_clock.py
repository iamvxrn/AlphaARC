"""The move budget must not be mistaken for a control doing something."""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from alphaarc.clock import ClockTracker


def _board(h=8, w=24, bg=5):
    return [[bg] * w for _ in range(h)]


def _tick(g, step):
    """The budget strip: row 0, eaten one cell at a time from the right."""
    n = [row[:] for row in g]
    n[0][max(0, len(n[0]) - 1 - step)] = 1
    return n


def test_a_click_that_only_ticks_the_clock_reads_as_dead():
    """The whole point. Exact board equality is FALSE on every one of these steps,
    which is why inhibition of return never fired in vc33, r11l, ft09, g50t, s5i5
    or su15 -- every one of them draws a budget strip."""
    t = ClockTracker()
    g = _board()
    verdicts = []
    for step in range(16):
        n = _tick(g, step)
        assert n != g, "the boards differ on every step: that is the bug"
        verdicts.append(t.clock_only(g, n))
        g = n
    assert not any(verdicts[: ClockTracker.WARMUP]), "must not guess before it has data"
    assert all(verdicts[ClockTracker.WARMUP + 1:]), \
        "after the warmup a clock-only step must read as dead: %r" % verdicts


def test_a_real_effect_elsewhere_is_not_the_clock():
    t = ClockTracker()
    g = _board()
    for step in range(12):
        g2 = _tick(g, step)
        t.clock_only(g, g2)
        g = g2
    live = [row[:] for row in g]
    live[4][4] = 7                      # something happened away from the strip
    live = _tick(live, 12)              # ...and the clock ticked as it always does
    assert not t.clock_only(g, live), "a change off the strip must count as an effect"


def test_a_game_with_no_strip_falls_back_to_exact_equality():
    """tn36 has no budget strip. There the old test was right, and nothing should
    be written off just because some row happens to be busy."""
    t = ClockTracker()
    g = _board()
    for step in range(12):
        n = [row[:] for row in g]
        n[step % 8][step % 8] = 3       # scattered: no line reaches the threshold
        t.clock_only(g, n)
        g = n
    n = [row[:] for row in g]
    n[2][6] = 4
    assert not t.clock_only(g, n)
    assert t.clock_only(g, [row[:] for row in g]), "an identical board is still dead"


def test_a_rescale_is_never_the_clock():
    """vc33 redraws its whole scene at a new size; comparing only the overlap
    would call that nothing at all."""
    t = ClockTracker()
    assert not t.clock_only(_board(8, 24), _board(16, 48))


def _run():
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    bad = 0
    for fn in tests:
        try:
            fn()
            print("PASS", fn.__name__)
        except AssertionError as e:
            bad += 1
            print("FAIL", fn.__name__, "--", e)
    print(f"\n{len(tests) - bad}/{len(tests)} passed")
    return bad


if __name__ == "__main__":
    sys.exit(1 if _run() else 0)
