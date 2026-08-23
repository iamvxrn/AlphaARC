"""Python CausalMapper port -- parity with Go pkg/actuate. Run directly."""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from alphaarc.actuate import CausalMapper, CellChange, Control  # noqa: E402


class TriggerEnv:
    """A button paints a distant target cell; a direct click on the target is inert."""

    def __init__(self, bx, by, tr, tc, color, h=8, w=8, bg=0):
        self.bx, self.by, self.tr, self.tc, self.color = bx, by, tr, tc, color
        self.h, self.w, self.bg = h, w, bg
        self.painted = False

    def reset(self):
        self.painted = False
        return self._g()

    def _g(self):
        g = [[self.bg] * self.w for _ in range(self.h)]
        g[self.by][self.bx] = 7
        if self.painted:
            g[self.tr][self.tc] = self.color
        return g

    def step(self, ctrl: Control):
        if ctrl.kind == "click" and ctrl.x == self.bx and ctrl.y == self.by:
            self.painted = True
        return self._g()


def test_learns_indirect_actuator():
    env = TriggerEnv(bx=1, by=1, tr=6, tc=5, color=5)
    m = CausalMapper()
    m.explore(env, [Control("click", x=5, y=6), Control("click", x=1, y=1)])
    ctrl = m.control_for_cell(6, 5, 5)
    assert ctrl == Control("click", x=1, y=1), ctrl


def test_direct_click_useless():
    env = TriggerEnv(bx=0, by=0, tr=5, tc=5, color=9)
    m = CausalMapper()
    m.explore(env, [Control("click", x=5, y=5)])  # only the inert direct click
    assert m.control_for_cell(5, 5, 9) is None


def test_plan_dedups_and_reports_unmet():
    env = TriggerEnv(bx=2, by=3, tr=6, tc=6, color=4)
    m = CausalMapper()
    m.explore(env, [Control("click", x=2, y=3)])
    plan, unmet = m.plan([CellChange(6, 6, 0, 4), CellChange(0, 0, 0, 1)])
    assert plan == [Control("click", x=2, y=3)], plan
    assert len(unmet) == 1 and unmet[0].to == 1, unmet


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
