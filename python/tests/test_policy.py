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
    tok = Policy._token(partial, bg, xy[0], xy[1])

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
    tok = Policy._token(g, bg, xy[0], xy[1])
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
    assert p.succ[(tok, p._key(lo))] == hi, "the transition FROM the bad state is remembered"
    assert p.succ[(tok, p._key(hi))] == lo, "and so is the one from the good state"



def test_no_credit_is_carried_across_a_replaced_board():
    """Completing a level swaps the board. Crediting the last click with the delta
    across that seam invents a huge reward and writes a junk transition, exactly
    where the score starts paying."""
    bg = 9
    board_a = _bg_grid(7, 14, bg)
    _place(board_a, 1, 1, SYM_BOX)
    _place(board_a, 1, 8, SYM_BOX)
    board_b = _bg_grid(7, 14, bg)          # an unrelated next level
    _place(board_b, 1, 1, SYM_BOX)

    def run(tell_the_policy):
        p = Policy(explore=0.0, rng=random.Random(0))
        p.choose_click(board_a, bg)        # leaves a trace pointing at board A
        if tell_the_policy:
            p.board_replaced()
        p.choose_click(board_b, bg)        # first click of the new level
        return p

    seamed = run(tell_the_policy=False)
    clean = run(tell_the_policy=True)

    assert seamed.succ, "fixture is wrong: the seam should have written a transition"
    assert not clean.succ, f"a transition was still learned across the seam: {clean.succ}"
    assert not any(clean.drive_gain.values()), \
        f"a click was still credited across the seam: {clean.drive_gain}"


def test_a_replaced_board_keeps_what_was_learned():
    """Only the trace is dropped -- the mechanic did not change with the scenery."""
    bg = 9
    g = _bg_grid(7, 14, bg)
    _place(g, 1, 1, SYM_BOX)
    _place(g, 1, 8, SYM_BOX)
    g[3][10] = bg

    p = Policy(explore=0.0, rng=random.Random(0))
    p.choose_click(g, bg)
    g[3][10] = SYM_BOX[2][2]               # the click fixed the anomaly
    p.choose_click(g, bg)
    learned_gain = dict(p.drive_gain)
    learned_tries = dict(p.tries)
    assert learned_gain, "fixture is wrong: nothing was learned to preserve"

    p.board_replaced()
    assert p.drive_gain == learned_gain, "reinforcement was thrown away with the board"
    assert p.tries == learned_tries, "try counts were thrown away with the board"
    assert p._prev_grid is None and p._last_token is None and p._prev_levels is None



def test_the_state_key_is_coarse_enough_to_ever_repeat():
    """Measured on re86: 15 presses produced 15 DISTINCT (control, levels) keys and
    zero reuse, because the move budget ticks inside the compression measurement and
    drifts the vector every action. A world model keyed that finely accumulates
    singletons and predicts nothing."""
    p = Policy(state_bucket=16, rng=random.Random(0))
    drifting = [[45, 130, 82, 56], [44, 129, 82, 56], [42, 128, 82, 56]]
    keys = {p._key(lv) for lv in drifting}
    assert len(keys) == 1, f"clock drift still splits one state into {len(keys)} keys"


def test_the_state_key_still_separates_a_real_mode():
    """re86's two modes read Count 82 and 78. Coarsening must not lump those."""
    p = Policy(state_bucket=16, rng=random.Random(0))
    assert p._key([45, 130, 82, 56]) != p._key([45, 130, 78, 56]), \
        "the two modes collapsed into one key -- the model can no longer tell them apart"


def test_the_measurement_itself_is_untouched():
    """Only the KEY is coarsened. Cropping or rescaling the levels themselves moves
    Reflect by tens of bits and was measured much worse (see bench README)."""
    bg = 9
    g = _bg_grid(7, 14, bg)
    _place(g, 1, 1, SYM_BOX)
    p = Policy(state_bucket=16, rng=random.Random(0))
    assert p._levels(g, bg) == Policy._levels(g, bg)


def test_the_name_survives_the_level_seam_that_broke_vc33():
    """The seam this signature exists for, pinned as the real vc33 geometry.

    vc33's two colour-9 buttons sit at eighth (row 3, col 7) and (4, 7) on level 1
    and at (3, 0) and (4, 0) on level 2 -- same colour, same size bucket, same
    rows, MIRRORED column. Naming by position in eighths called those four
    different controls, so the mechanic was learned twice and level 2 cost 68
    actions against a baseline of 18 while level 1 had cost 6 against 7.

    Naming by RANK among same-colour, same-size peers carries the pair across:
    mirroring a stacked pair does not reorder it.
    """
    bg = 0
    btn = [[9, 9], [9, 9]]
    right = _bg_grid(64, 64, bg)
    _place(right, 24, 56, btn)          # the pair, right-hand side (level 1)
    _place(right, 28, 56, btn)
    left = _bg_grid(64, 64, bg)
    _place(left, 24, 2, btn)            # the same pair, mirrored (level 2)
    _place(left, 28, 2, btn)

    top_r, bot_r = Policy._token(right, bg, 56, 24), Policy._token(right, bg, 56, 28)
    top_l, bot_l = Policy._token(left, bg, 2, 24), Policy._token(left, bg, 2, 28)
    assert top_r == top_l, "the upper button changed its name across the seam: %s vs %s" % (top_r, top_l)
    assert bot_r == bot_l, "the lower button changed its name across the seam: %s vs %s" % (bot_r, bot_l)
    assert top_r != bot_r, "the two buttons collapsed onto one name: %s" % top_r


def test_the_name_still_survives_a_small_shift():
    """The property the eighths-based name already had, kept: an object that moves
    a few cells is the same object."""
    bg = 9
    a = _bg_grid(64, 64, bg)
    _place(a, 20, 20, SYM_BOX)
    b = _bg_grid(64, 64, bg)
    _place(b, 22, 22, SYM_BOX)
    assert Policy._token(a, bg, 21, 21) == Policy._token(b, bg, 23, 23)


def test_a_pixel_coordinate_alone_is_not_the_name():
    """Guard the inverse: two DIFFERENT objects must not collapse onto one token,
    or a control's value would be polluted by whatever else shares its cell."""
    bg = 9
    g = _bg_grid(7, 14, bg)
    _place(g, 1, 1, SYM_BOX)
    _place(g, 1, 8, SYM_BOX)
    left = Policy._token(g, bg, 2, 2)
    right = Policy._token(g, bg, 9, 2)
    assert left != right, "two boxes in different halves of the frame share a name: %r" % left


def test_a_control_the_model_has_watched_do_nothing_loses_its_prior():
    """An absence of reward is not a KNOWN absence of effect.

    Both used to score 0.0, so a control the model had already watched do nothing
    from this exact state still collected the rank prior and the untried-optimism
    and got clicked again. Measured on vc33 seed 4: of 169 successful model
    lookups 142 predicted no change, while 52 of 90 transitions were nothing but
    the move clock -- the knowledge was there and the policy ignored it.
    """
    bg = 9
    g = _bg_grid(7, 14, bg)
    _place(g, 1, 1, SYM_BOX)
    _place(g, 1, 8, SYM_BOX)
    g[3][10] = bg

    p = Policy(explore=0.0, rng=random.Random(0))
    first = p.choose_click(g, bg)
    levels = Policy._levels(g, bg)
    tok = Policy._token(g, bg, first[0], first[1])
    # the model watched it do exactly nothing from this state
    p.succ[(tok, p._key(levels))] = list(levels)

    again = p.choose_click(g, bg)
    assert again != first, \
        "clicked a control the model says does nothing here: %s" % (again,)


def test_the_model_only_overrides_where_it_actually_knows():
    """A control the model has never seen from this state keeps its prior --
    otherwise nothing would ever be tried a first time."""
    bg = 9
    g = _bg_grid(7, 14, bg)
    _place(g, 1, 1, SYM_BOX)
    _place(g, 1, 8, SYM_BOX)
    g[3][10] = bg
    p = Policy(explore=0.0, rng=random.Random(0))
    assert p.choose_click(g, bg) is not None
    assert not p.succ, "nothing should be known before a transition is observed"


if __name__ == "__main__":
    sys.exit(1 if _run() else 0)
