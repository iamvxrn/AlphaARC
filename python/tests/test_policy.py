"""Behavioural tests for the model-free policy. Run directly (no pytest)."""

import os
import random
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from alphaarc.policy import Policy  # noqa: E402
from alphaarc.residual import residual_targets  # noqa: E402

SYM_BOX = [
    [2, 2, 2, 2, 2],
    [2, 4, 3, 4, 2],
    [2, 3, 4, 3, 2],
    [2, 4, 3, 4, 2],
    [2, 2, 2, 2, 2],
]


def _bg_grid(h, w, bg):
    return [[bg] * w for _ in range(h)]


def _place(grid, top, left, box):
    for r in range(len(box)):
        for c in range(len(box[r])):
            grid[top + r][left + c] = box[r][c]


def test_choose_returns_a_real_candidate():
    """No exploration -> the click is always one of the residual target points."""
    bg = 9
    g = _bg_grid(7, 14, bg)
    _place(g, 1, 1, SYM_BOX)
    _place(g, 1, 8, SYM_BOX)
    g[3][10] = bg  # a Correspondence residual
    p = Policy(explore=0.0, rng=random.Random(1))
    xy = p.choose_click(g, bg)
    targets = {(t.x, t.y) for t in residual_targets(g, bg, 8)}
    assert xy in targets, "%s not in %s" % (xy, targets)


def test_blank_grid_yields_no_candidate():
    p = Policy(explore=0.0)
    assert p.choose_click([[0, 0, 0], [0, 0, 0]], 0) is None


def test_positive_delta_reinforces_the_clicked_token():
    """A click followed by a compression-raising change must raise that token's drive_gain."""
    bg = 9
    partial = _bg_grid(7, 14, bg)
    _place(partial, 1, 1, SYM_BOX)
    _place(partial, 1, 8, SYM_BOX)
    partial[3][10] = bg  # broken; Correspondence savings = 23

    p = Policy(explore=0.0, rng=random.Random(0))
    xy = p.choose_click(partial, bg)      # picks some candidate, records it
    tok = "%d,%d" % (xy[0], xy[1])

    fixed = [row[:] for row in partial]
    fixed[3][10] = 4                       # fill the cell -> Correspondence 23->24
    p.choose_click(fixed, bg)              # credits the previous token by +ΔL
    assert p.drive_gain.get(tok, 0.0) > 0.0, "reinforcement did not credit a compression-raising click"


def test_dead_click_is_tabooed():
    """A click that changes nothing accrues an inhibition-of-return penalty."""
    bg = 9
    g = _bg_grid(7, 14, bg)
    _place(g, 1, 1, SYM_BOX)
    _place(g, 1, 8, SYM_BOX)
    g[3][10] = bg
    p = Policy(explore=0.0, rng=random.Random(0))
    xy = p.choose_click(g, bg)
    tok = "%d,%d" % (xy[0], xy[1])
    p.choose_click(g, bg)  # identical grid -> previous click "did nothing"
    assert p.dead.get(tok, 0.0) > 0.0, "a no-op click should be tabooed"


def _run():
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    failed = 0
    for t in tests:
        try:
            t()
            print("PASS", t.__name__)
        except AssertionError as e:
            failed += 1
            print("FAIL", t.__name__, "--", e)
        except Exception as e:  # noqa: BLE001
            failed += 1
            print("ERROR", t.__name__, "--", repr(e))
    print("\n%d/%d passed" % (len(tests) - failed, len(tests)))
    return failed



def _toggle_boards():
    """Two 4x4 boxes; box B's fill toggles between matching A and not.

    B stays a solid object in BOTH states, so its centroid is always a click
    candidate -- exactly like an ft09 tile, and unlike a residual that vanishes
    once fixed. This is the shape of world that defeats an averaged credit.
    """
    bg = 9
    g = [[bg] * 16 for _ in range(8)]
    for r in range(2, 6):
        for c in range(2, 6):
            g[r][c] = 8
    for r in range(2, 6):
        for c in range(10, 14):
            g[r][c] = 3          # start MISmatched
    return g, bg


def _run_toggle_world(model_on, steps=40, seed=0):
    """Return how many steps the board spent in the compressible state."""
    grid, bg = _toggle_boards()
    p = Policy(explore=0.0, rng=random.Random(seed))
    good = 0
    for _ in range(steps):
        if not model_on:
            p.succ.clear()               # disable the state-keyed prediction
        xy = p.choose_click(grid, bg)
        if xy is not None and 2 <= xy[1] < 6 and 10 <= xy[0] < 14:
            new = 3 if grid[3][11] == 8 else 8
            for r in range(2, 6):
                for c in range(10, 14):
                    grid[r][c] = new
        good += 1 if grid[3][11] == 8 else 0
    return good


def test_state_keyed_prediction_beats_averaged_credit_on_a_toggle():
    """ft09's defect, as a differential test: an averaged credit cancels on an
    involutive control, so the agent flips it forever. Predicting what the control
    does FROM HERE tells the two states apart."""
    off = _run_toggle_world(model_on=False)
    on = _run_toggle_world(model_on=True)
    assert on > off, f"the prediction bought nothing: {on} vs {off} good steps"


def test_the_ema_alone_really_does_cancel_on_a_toggle():
    """The measurement behind the fix, pinned: +d then -d on one token."""
    good, bg = _toggle_boards()
    matched = [row[:] for row in good]
    for r in range(2, 6):
        for c in range(10, 14):
            matched[r][c] = 8

    p = Policy(explore=0.0, rng=random.Random(1))
    tok = p._tok(11, 3)
    lo, hi = Policy._levels(good, bg), Policy._levels(matched, bg)
    assert lo != hi, "fixture must actually change the compression levels"

    p._prev_grid, p._prev_levels, p._last_token = good, lo, tok
    p._credit_last(matched, bg, hi)
    p._prev_grid, p._prev_levels, p._last_token = matched, hi, tok
    p._credit_last(good, bg, lo)

    spread = max(abs(a - b) for a, b in zip(lo, hi))
    assert abs(p.drive_gain[tok]) < spread, "the EMA should largely cancel -- the defect"
    assert p.succ[(tok, tuple(lo))] == hi, "but the transition FROM the bad state is remembered"
    assert p.succ[(tok, tuple(hi))] == lo, "and so is the one from the good state"


if __name__ == "__main__":
    sys.exit(1 if _run() else 0)
