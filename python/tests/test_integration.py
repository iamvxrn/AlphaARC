"""Integration capstone: the five layers compose to solve an INDIRECT
match-the-target task end-to-end.

goalfind (what should change) + CausalMapper (which indirect control makes it) ->
execute -> repeat, until the incomplete copy matches the exemplar. The copy's
cells can ONLY be set via trigger buttons; a direct click on a copy cell is
inert (the vc33/ft09 shape). Compression alone is blind to the arbitrary
interior, and a direct click never works -- only goal-discovery + actuation
together solve it.

Run: python3 python/tests/test_integration.py
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from alphaarc.actuate import CausalMapper, Control  # noqa: E402
from alphaarc.goalfind import next_fix  # noqa: E402

BG = 9

# 6x6 framed exemplar (colour-2 border, arbitrary asymmetric interior).
BOX = [
    [2, 2, 2, 2, 2, 2],
    [2, 1, 3, 4, 6, 2],
    [2, 5, 7, 8, 1, 2],
    [2, 3, 6, 5, 4, 2],
    [2, 7, 1, 5, 3, 2],
    [2, 2, 2, 2, 2, 2],
]

MISSING = [(2, 11), (3, 13), (4, 12)]          # copy cells knocked out (absolute)
BUTTONS = [(8, 1), (8, 3), (8, 5)]             # (row, col) of each trigger button


def _target_color(r, c):
    return BOX[r - 1][c - 10]  # copy box top-left is (1,10)


class IndirectMatchEnv:
    """Exemplar at (1,1) complete; copy at (1,10) missing 3 interior cells that
    are painted ONLY by their trigger buttons. Direct clicks on cells: inert."""

    def __init__(self):
        self.h, self.w, self.bg = 9, 18, BG
        self.painted = {}

    def reset(self):
        self.painted = {}
        return self._grid()

    def _grid(self):
        g = [[BG] * self.w for _ in range(self.h)]
        for r in range(6):
            for c in range(6):
                g[1 + r][1 + c] = BOX[r][c]          # exemplar (complete)
                g[1 + r][10 + c] = BOX[r][c]          # copy (start complete...)
        for (r, c) in MISSING:                        # ...then knock out
            g[r][c] = BG
        for (br, bc) in BUTTONS:
            g[br][bc] = 7                             # trigger buttons (lone marks)
        for (r, c), col in self.painted.items():      # apply what buttons painted
            g[r][c] = col
        return g

    def step(self, ctrl: Control):
        if ctrl.kind == "click":
            for i, (br, bc) in enumerate(BUTTONS):
                if ctrl.x == bc and ctrl.y == br:     # x=col, y=row
                    tr, tc = MISSING[i]
                    self.painted[(tr, tc)] = _target_color(tr, tc)
        return self._grid()


def test_integration_solves_indirect_match():
    env = IndirectMatchEnv()
    grid = env.reset()

    # 1) Learn the affordance model: try clicking every button AND every missing
    #    cell directly (to prove direct clicks are inert).
    candidates = [Control("click", x=bc, y=br) for (br, bc) in BUTTONS]
    candidates += [Control("click", x=c, y=r) for (r, c) in MISSING]
    mapper = CausalMapper()
    mapper.explore(env, candidates)

    # 2) Closed loop: goalfind says WHAT, mapper says HOW (indirect), execute.
    grid = env.reset()
    used = []
    for _ in range(20):
        fix = next_fix(grid, BG)
        if fix is None:
            break
        r, c, to = fix
        ctrl = mapper.control_for_cell(r, c, to)
        assert ctrl is not None, "mapper has no actuator for (%d,%d)->%d" % (r, c, to)
        # the actuator must be a BUTTON, never a direct click on the target cell
        assert not (ctrl.x == c and ctrl.y == r), "used a direct (inert) click on the target"
        assert (ctrl.y, ctrl.x) in BUTTONS, "actuator should be a trigger button, got %r" % ctrl
        used.append((ctrl.y, ctrl.x))
        grid = env.step(ctrl)

    # 3) The copy now matches the exemplar exactly.
    for r in range(6):
        for c in range(6):
            assert grid[1 + r][10 + c] == BOX[r][c], "copy cell (%d,%d) not matched" % (r, c)
    # solved purely via indirect trigger buttons (one per missing cell)
    assert set(used) == set(BUTTONS), "should have used each trigger button, used=%s" % used


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
