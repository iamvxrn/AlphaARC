"""The planner exists for one measured reason: reward that is invisible one step
ahead. Test it on exactly that. Run directly (no pytest)."""

import os
import random
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))

from alphaarc.planner import HybridPolicy, RunPlanner, _gain, rigid_move  # noqa: E402
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



def test_the_hybrid_spends_the_first_actions_cheaply():
    """vc33 clears level 1 in about seven actions with one-step credit and never
    with the planner, because probing costs more than the level is worth. So the
    cheap agent must own the opening."""
    h = HybridPolicy(switch_after=5, rng=random.Random(0))
    g = _blank(9, 24)
    _place(g, 1, 1, SYM_BOX)
    _place(g, 1, 8, SYM_BOX)
    for _ in range(5):
        h.choose_click(g, BG)
        assert h.active is h.policy, "handed over before the cheap agent had its chance"
    h.choose_click(g, BG)
    assert h.active is h.planner, "never escalated, so long levels stay unsolved"


def test_a_new_level_gives_the_cheap_agent_another_chance():
    """The next level's opening may well be short too, so the clock restarts."""
    h = HybridPolicy(switch_after=2, rng=random.Random(0))
    g = _blank(9, 24)
    _place(g, 1, 1, SYM_BOX)
    for _ in range(4):
        h.choose_click(g, BG)
    assert h.active is h.planner
    h.board_replaced()
    assert h.active is h.policy, "stayed expensive into a level that might be cheap"



def test_the_hybrid_hands_over_early_when_its_clicks_do_nothing():
    """lp85's regression, as a rule.

    Its top row is a six-block progress indicator, and "smallest object first"
    ranks those ahead of the 4x4 palette swatches that actually do something. The
    one-step policy spends its whole opening clicking the indicator. Dead clicks
    are the evidence that the cheap agent is not working, so they should hand over
    long before the clock does.
    """
    h = HybridPolicy(switch_after=100, dead_streak=3, rng=random.Random(0))
    g = _blank(9, 24)
    _place(g, 1, 1, SYM_BOX)
    for i in range(5):
        h.choose_click(g, BG)                  # the board never changes
    assert h.active is h.planner, \
        f"still cheap after {h.dead_run} dead clicks with a switch_after of 100"


def test_a_working_cheap_agent_keeps_control():
    """vc33's clicks DO change the board, and it clears level 1 in about seven
    actions. Handing over there costs the level, so a live board must not switch."""
    h = HybridPolicy(switch_after=100, dead_streak=3, rng=random.Random(0))
    w = ValleyWorld()
    # Only while the board is still moving. Once the scalar saturates the board
    # stops changing, and handing over THEN is correct -- the cheap agent really
    # has stopped making progress.
    for _ in range(ValleyWorld.STAGES - 1):
        xy = h.choose_click(w.grid(), BG)
        w.click(xy)
    assert h.active is h.policy, "handed over while the cheap agent was working"



def test_a_key_is_a_control_like_any_other():
    """Eight of seventeen train games are keyboard-driven and none has ever scored,
    because neither agent could express a key and the adapter pressed a random one.
    A key is exactly what this machinery wants: something you can press N times,
    whose name never moves when the board redraws."""
    p = RunPlanner(run_length=3, explore=0.0, rng=random.Random(0))
    g = _blank(9, 24)
    _place(g, 1, 1, SYM_BOX)
    got = p.choose(g, BG, keys=(1, 2, 3))
    assert got is not None and got[0] in ("click", "key"), got

    # With no click candidates at all, only keys remain.
    blank = _blank(9, 24)
    got = p.choose(blank, BG, keys=(1, 2, 3))
    assert got is not None and got[0] == "key" and got[1] in (1, 2, 3), got


def test_a_key_run_survives_a_redraw_by_construction():
    """A key's name is the key. Nothing about the board can move it."""
    p = RunPlanner(run_length=3, explore=0.0, rng=random.Random(0))
    small, big = _blank(9, 24), _blank(18, 48)
    first = p.choose(small, BG, keys=(1, 2))
    assert first[0] == "key"
    p._run_token, p._presses_left = "k%d" % first[1], 2
    p._prev_grid = [row[:] for row in small]
    again = p.choose(big, BG, keys=(1, 2))
    assert again == first, f"lost the key across a redraw: {first} -> {again}"


def test_the_hybrid_gives_a_keyboard_game_to_the_planner_at_once():
    """The one-step policy cannot express a key, so waiting out its clock on a
    keyboard-only game just spends the level on random presses."""
    h = HybridPolicy(switch_after=100, rng=random.Random(0))
    g = _blank(9, 24)
    _place(g, 1, 1, SYM_BOX)
    got = h.choose(g, BG, keys=(1, 2, 3), clickable=False)
    assert got is not None and got[0] == "key", got



def test_rigid_move_finds_the_avatar_and_ignores_a_ticking_hud():
    """An avatar announces itself by translating rigidly. A budget strip does not:
    it changes cells without preserving the count and the offsets."""
    a = _blank(10, 10)
    a[4][4] = 7
    a[4][5] = 7
    b = [row[:] for row in a]
    b[4][4] = BG
    b[4][5] = BG
    b[5][4] = 7
    b[5][5] = 7
    assert rigid_move(a, b, BG) == (7, 1, 0)

    hud = [row[:] for row in a]
    hud[0][3] = 2                                   # one more cell appears
    assert rigid_move(a, hud, BG) is None


class MoveWorld:
    """Four keys translate an avatar; one cell elsewhere breaks the symmetry and is
    therefore what the residual points at. Reaching it is the goal."""

    def __init__(self):
        self.r, self.c = 2, 2
        self.reached = False

    def grid(self):
        g = _blank(12, 12)
        for r in range(6, 11):
            for c in range(6, 11):
                g[r][c] = 3                          # scenery, so the board is not empty
        g[8][8] = 1                                  # the anomaly: the goal
        g[self.r][self.c] = 7                        # the avatar
        return g

    def press(self, key):
        dr, dc = {1: (-1, 0), 2: (1, 0), 3: (0, -1), 4: (0, 1)}[key]
        self.r = max(0, min(11, self.r + dr))
        self.c = max(0, min(11, self.c + dc))
        if (self.r, self.c) == (8, 8):
            self.reached = True


def test_the_planner_learns_the_keys_and_steers_the_avatar_to_the_anomaly():
    """The measured gap on ls20: its keys work and its credit is fine, yet it
    scores zero, because compression rewards a tidier board and not ARRIVAL."""
    p = RunPlanner(run_length=2, explore=0.0, rng=random.Random(0))
    w = MoveWorld()
    for _ in range(40):
        got = p.choose(w.grid(), BG, keys=(1, 2, 3, 4), clickable=False)
        if got is None:
            break
        assert got[0] == "key"
        w.press(got[1])
        if w.reached:
            break
    assert p.avatar == 7, f"never identified the avatar: {p.avatar}"
    assert len(p.moves) >= 2, f"learned too few directions: {p.moves}"
    assert w.reached, f"never arrived; avatar stopped at {(w.r, w.c)}"


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
