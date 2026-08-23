"""Rung-1 proof: the mismatch drive guides a task the compression drive is blind
to. Run: python3 python/tests/test_mismatch.py"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from alphaarc.mdl import best_primitive, best_primitive_delta  # noqa: E402
from alphaarc.mismatch import mismatch_cells, mismatch_delta, region_mismatch  # noqa: E402


def _clone(g):
    return [row[:] for row in g]


# An ARBITRARY, asymmetric, aperiodic 4x4 target: no mirror axis, no period, and
# every colour distinct enough that Count/Reflect/Translate find ~nothing. The
# compression drive has no gradient toward this pattern -- only "match the
# target" does.
TARGET = [
    [1, 2, 3, 4],
    [5, 6, 7, 8],
    [2, 4, 6, 1],
    [7, 3, 5, 8],
]
BG = 0


def test_target_is_compression_blind():
    """Sanity for the whole point: the target region has ~no self-regularity, so
    the compression drive can't guide building it (all primitives near 0)."""
    p, sav = best_primitive(TARGET, BG)
    assert sav <= 2, "target should be compression-poor, got %s=%d" % (p.name, sav)


def test_mismatch_measures_and_points_at_the_wrong_cells():
    ws = _clone(TARGET)
    ws[1][1] = 0  # blank one cell
    ws[2][3] = 9  # wrong colour on another
    assert region_mismatch(ws, TARGET, BG) == 2
    cells = set(mismatch_cells(ws, TARGET, BG))
    assert cells == {(1, 1), (2, 3)}, cells


def test_filling_reduces_mismatch_while_compression_stays_flat():
    ws = _clone(TARGET)
    ws[1][1] = 0
    fixed = _clone(ws)
    fixed[1][1] = TARGET[1][1]  # the correct fill
    # mismatch drive sees the improvement...
    assert mismatch_delta(ws, fixed, TARGET, BG) == 1
    # ...while the compression drive is ~blind to it (arbitrary target).
    assert abs(best_primitive_delta(ws, fixed, BG)) <= 1


def test_mismatch_hillclimb_recovers_the_target():
    """A model-free loop that only ever reduces mismatch fills the workspace to
    the target exactly -- the drive is a correct, sufficient goal signal here."""
    ws = [[BG] * 4 for _ in range(4)]  # start blank
    for _ in range(100):
        cells = mismatch_cells(ws, TARGET, BG)
        if not cells:
            break
        r, c = cells[0]
        # act = set this cell toward the target; accept only if mismatch drops.
        before = region_mismatch(ws, TARGET, BG)
        cand = _clone(ws)
        cand[r][c] = TARGET[r][c]
        if region_mismatch(cand, TARGET, BG) < before:
            ws = cand
    assert ws == TARGET, "mismatch drive failed to recover the target:\n%s" % ws
    assert region_mismatch(ws, TARGET, BG) == 0


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


if __name__ == "__main__":
    sys.exit(1 if _run() else 0)
