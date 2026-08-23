"""Rung-2 proof: discover the (target, workspace) pairing on a board, then the
mismatch drive completes it. Run: python3 python/tests/test_goalfind.py"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from alphaarc.goalfind import discover_match_tasks, next_fix  # noqa: E402
from alphaarc.mdl import best_primitive_delta  # noqa: E402

BG = 9

# A framed 6x6 exemplar: colour-2 border, ARBITRARY asymmetric/aperiodic 4x4
# interior (no interior 2s -> stays separate from the frame; compression-poor).
BOX = [
    [2, 2, 2, 2, 2, 2],
    [2, 1, 3, 4, 6, 2],
    [2, 5, 7, 8, 1, 2],
    [2, 3, 6, 5, 4, 2],
    [2, 7, 1, 5, 3, 2],
    [2, 2, 2, 2, 2, 2],
]


def _bg_grid(h, w):
    return [[BG] * w for _ in range(h)]


def _place(g, top, left, box):
    for r in range(len(box)):
        for c in range(len(box[r])):
            g[top + r][left + c] = box[r][c]


def _make_board():
    """Complete exemplar at (1,1); an incomplete copy at (1,10) missing 3 interior
    cells (set to bg)."""
    g = _bg_grid(9, 18)
    _place(g, 1, 1, BOX)
    _place(g, 1, 10, BOX)
    # knock out 3 interior cells of the second box (absolute coords)
    for (r, c) in [(2, 11), (3, 13), (4, 12)]:
        g[r][c] = BG
    return g


def test_discovers_target_and_peer():
    g = _make_board()
    tasks = discover_match_tasks(g, BG)
    assert len(tasks) == 1, "expected one match task, got %d" % len(tasks)
    ref, peers = tasks[0]
    assert ref == BOX, "target should be the COMPLETE box"
    assert len(peers) == 1 and peers[0][0:2] == (1, 10), peers


def test_next_fix_points_at_a_real_missing_cell():
    g = _make_board()
    fix = next_fix(g, BG)
    assert fix is not None
    r, c, k = fix
    # it must be one of the knocked-out cells, and the target colour must match BOX
    assert (r, c) in {(2, 11), (3, 13), (4, 12)}, fix
    assert k == BOX[r - 1][c - 10], "target colour must come from the exemplar"


def test_full_solve_loop_completes_the_copy():
    """Discover + mismatch-drive drives the incomplete copy to match the exemplar,
    and the fills it makes are compression-blind (arbitrary interior) -- so only
    the target-matching goal could have driven them."""
    g = _make_board()
    steps = 0
    while True:
        fix = next_fix(g, BG)
        if fix is None:
            break
        r, c, k = fix
        before = [row[:] for row in g]
        g[r][c] = k
        # each fix is a genuine mismatch reduction the compression drive is ~blind to
        assert abs(best_primitive_delta(before, g, BG)) <= 2, "fill should be compression-neutral"
        steps += 1
        assert steps < 50, "loop did not converge"
    # both boxes now identical -> the copy equals the exemplar
    for r in range(6):
        for c in range(6):
            assert g[1 + r][1 + c] == g[1 + r][10 + c] == BOX[r][c]
    assert steps == 3, "should have taken exactly the 3 missing cells, took %d" % steps


def test_no_false_positive_on_colour_variants():
    """Two same-size framed regions that differ only in FOREGROUND colour (a
    palette/legend, not an incomplete copy) must NOT produce a match task -- the
    lp85 false positive (two colour-3 bands, diffs 3->5, no holes)."""
    # Same frame (colour 2), same size, but interiors are different SOLID-ish
    # colours -- no bg holes anywhere.
    box_a = [
        [2, 2, 2, 2, 2],
        [2, 1, 1, 1, 2],
        [2, 1, 3, 1, 2],
        [2, 1, 1, 1, 2],
        [2, 2, 2, 2, 2],
    ]
    box_b = [
        [2, 2, 2, 2, 2],
        [2, 6, 6, 6, 2],
        [2, 6, 7, 6, 2],
        [2, 6, 6, 6, 2],
        [2, 2, 2, 2, 2],
    ]
    g = _bg_grid(7, 14)
    _place(g, 1, 1, box_a)
    _place(g, 1, 8, box_b)
    assert discover_match_tasks(g, BG) == [], "colour variants must not be a fill task"
    assert next_fix(g, BG) is None


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
