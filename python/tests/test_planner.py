"""The planner exists for one measured reason: reward that is invisible one step
ahead. Test it on exactly that. Run directly (no pytest)."""

import os
import random
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))

from alphaarc.planner import RunPlanner, _gain  # noqa: E402
from alphaarc.policy import Policy  # noqa: E402

BG = 9
SYM_BOX = [
    [2, 2, 2, 2, 2],
    [2, 4, 3, 4, 2],
    [2, 3, 4, 3, 2],
    [2, 4, 3, 4, 2],
    [2, 2, 2, 2, 2],
]


def _blank(h, w):
    return [[BG] * w for _ in range(h)]


def _place(g, top, left, box):
    for r in range(len(box)):
        for c in range(len(box[r])):
            g[top + r][left + c] = box[r][c]


class ValleyWorld:
    """vc33's shape: one control, and pressing it walks a scalar whose compression
    DIPS before it pays.

    Measured on the real game, Reflect along the up-direction goes
    308 -> 200 -> 256 -> 364. Here the same shape is built by breaking a clean twin
    and then over-restoring it, and the stage levels are asserted below so the test
    cannot quietly stop being a valley: gains are 0, -24, -24, +25.
    """

    STAGES = 4

    def __init__(self):
        self.n = 0
        self.decoys = [0, 0, 0, 0]

    def grid(self):
        g = _blank(9, 24)
        _place(g, 1, 1, SYM_BOX)
        _place(g, 1, 8, SYM_BOX)
        # Decoy controls: they visibly change the board, so they are never written
        # off as inert, but they never advance the scalar. Real games are full of
        # these -- vc33 has ten components, ft09 eight tiles -- and an agent that
        # abandons the valley control after its bad first press spends its budget
        # here instead.
        for i, on in enumerate(self.decoys):
            g[7][2 + 4 * i] = 1 if on else BG
        stage = min(self.n, self.STAGES - 1)
        if stage == 1:
            for r in range(1, 6):
                g[r][9] = 1
                g[r][11] = 1                 # badly broken
        elif stage == 2:
            for r in range(1, 6):
                g[r][9] = 1                  # half mended
        elif stage >= 3:
            _place(g, 1, 15, SYM_BOX)        # a third twin: the payoff
        return g

    def click(self, xy):
        """Only the control on the left box moves the scalar; the rest are decoys."""
        if xy is None:
            return
        x, y = xy
        if 1 <= x <= 5 and 1 <= y <= 5:
            self.n = min(self.n + 1, self.STAGES - 1)
            return
        for i in range(len(self.decoys)):
            if abs(x - (2 + 4 * i)) <= 1 and abs(y - 7) <= 1:
                self.decoys[i] ^= 1
                return


def _levels(g):
    return RunPlanner._levels(g, BG)


def test_the_fixture_really_has_a_valley():
    """If the world does not dip before it climbs, the test proves nothing."""
    w = ValleyWorld()
    base = _levels(ValleyWorld().grid())
    series = []
    for n in range(ValleyWorld.STAGES):
        w.n = n
        series.append(_gain(base, _levels(w.grid())))
    assert series[1] < 0 and series[2] < 0, f"no valley to cross: {series}"
    assert series[3] > 0, f"no payoff at the end: {series}"
    assert series[3] > series[1], series


def _run(agent, steps=18, seed=0):
    w = ValleyWorld()
    for _ in range(steps):
        xy = agent.choose_click(w.grid(), BG)
        w.click(xy)
    return w.n


def test_the_planner_crosses_the_valley():
    reached = _run(RunPlanner(run_length=3, explore=0.0, rng=random.Random(0)))
    assert reached >= ValleyWorld.STAGES - 1, \
        f"the planner stopped at stage {reached} of {ValleyWorld.STAGES - 1}"


def test_a_control_that_moved_but_did_not_pay_is_escalated_to_a_run():
    """The budget policy, tested as a rule rather than through a world.

    A first press is CHEAP -- one action -- because probing every candidate with a
    full run costs more than a short level is worth: vc33's level-1 baseline is
    seven actions, and eight controls at three presses each spends the whole level
    before exploiting anything (measured: vc33 4.34 -> 0.05). But a control that
    moved the board WITHOUT paying is exactly where a valley hides, so that one
    earns a full run.
    """
    p = RunPlanner(run_length=3, explore=0.0, rng=random.Random(0))
    g = _blank(9, 24)
    _place(g, 1, 1, SYM_BOX)
    _place(g, 1, 8, SYM_BOX)

    first = p.choose_click(g, BG)
    assert p._presses_left == 0, \
        f"a first press should cost one action, queued {p._presses_left + 1}"

    # Cheap breadth comes first by design, so mark the rest as already tried and
    # paying nothing; only then is there a reason to spend a run on anything.
    here = p._levels(g, BG)
    for c in p._candidates(g, BG):
        sig = p._signature(g, BG, c.x, c.y)
        p.profiles[sig] = [here, list(here)]                   # moved, gained nothing
    tok = p._signature(g, BG, *first)
    p.profiles[tok] = [here, [v - 5 for v in here]]            # this one LOST ground
    p._run_token = None
    p._prev_grid = None

    again = p.choose_click(g, BG)
    assert p._presses_left == p.run_length - 1, \
        f"nothing was escalated to a full run: queued {p._presses_left + 1} of {p.run_length}"
    assert p._signature(g, BG, *again) in p.ran, "the escalated control was not recorded"


def test_an_inert_control_is_dropped_mid_run():
    """A control that does nothing must not burn the rest of its run."""
    p = RunPlanner(run_length=4, explore=0.0, rng=random.Random(0))
    g = _blank(9, 18)
    _place(g, 1, 1, SYM_BOX)
    _place(g, 1, 8, SYM_BOX)
    first = p.choose_click(g, BG)
    second = p.choose_click(g, BG)             # same board back: nothing happened
    assert p._signature(g, BG, *first) in p.inert, "a dead control was not noticed"
    assert second != first, "kept pressing a control that changed nothing"


def test_learning_survives_a_new_board_but_the_run_does_not():
    p = RunPlanner(run_length=3, explore=0.0, rng=random.Random(0))
    w = ValleyWorld()
    p.choose_click(w.grid(), BG)
    p.board_replaced()
    assert p._presses_left == 0 and p._run_token is None
    assert p.tries, "try counts should survive -- the mechanic did not change"



def test_both_agents_cross_this_particular_valley():
    """Honest note on what this fixture does NOT show.

    The synthetic valley was built to discriminate the run-planner from one-step
    credit and it does not: both cross it in the minimum three actions. The reason
    is instructive -- the one-step policy keys its credit to the CENTROID of a
    residual cluster, and that centroid moves as the board changes, so the penalty
    for the bad first press lands on a token that is never revisited and never
    accumulates. The valley punishes it in principle and not in this fixture.

    So the tests above pin the planner's mechanics (a run survives a bad first
    press, an inert control is dropped, learning survives a new board), and the
    question of whether runs beat one-step credit is settled on the real games by
    `make quick`, not here.
    """
    old = _run(Policy(explore=0.0, rng=random.Random(0)))
    new = _run(RunPlanner(run_length=3, explore=0.0, rng=random.Random(0)))
    assert old == new == ValleyWorld.STAGES - 1, (old, new)



def test_a_control_keeps_its_name_when_the_board_rescales():
    """The measured failure this exists to fix.

    Instrumented over 30 clicks on vc33 -- which redraws its whole scene at a new
    scale on every press -- the planner hit 30 DISTINCT positions, repeated none,
    and so never exploited anything. A control has to be named by the object under
    it, not by where that object happened to be.
    """
    def scene(n, side):
        g = [[BG] * n for _ in range(n)]
        for r in range(side, 3 * side):
            for c in range(side, 3 * side):
                g[r][c] = 8
        return g

    small, big = scene(16, 2), scene(32, 4)
    assert RunPlanner._signature(small, BG, 3, 3) == RunPlanner._signature(big, BG, 7, 7), \
        "the same object got a different name at a different scale"


def test_a_run_follows_its_control_across_a_redraw():
    """And the plan must be 'press control S again', not 'click that pixel again'."""
    def scene(n, side):
        g = [[BG] * n for _ in range(n)]
        for r in range(side, 3 * side):
            for c in range(side, 3 * side):
                g[r][c] = 8
        for i in range(2):                       # a second object, so there is a choice
            g[n - 2][2 + 3 * i] = 1
        return g

    p = RunPlanner(run_length=3, explore=0.0, rng=random.Random(0))
    small = scene(16, 2)
    first = p.choose_click(small, BG)
    sig = p._signature(small, BG, *first)
    p._run_token, p._presses_left = sig, 2       # mid-run on that control

    big = scene(32, 4)                           # the board redraws at double scale
    p._prev_grid = [row[:] for row in small]     # ... so the board DID change
    nxt = p.choose_click(big, BG)
    assert nxt is not None
    assert p._signature(big, BG, *nxt) == sig, \
        f"lost the control across the redraw: {sig} -> {p._signature(big, BG, *nxt)}"


if __name__ == "__main__":
    failures = 0
    for name, fn in sorted(globals().items()):
        if name.startswith("test_") and callable(fn):
            try:
                fn()
                print(f"ok   {name}")
            except AssertionError as e:
                failures += 1
                print(f"FAIL {name}: {e}")
    print("PASS" if not failures else f"{failures} FAILURE(S)")
    sys.exit(1 if failures else 0)
