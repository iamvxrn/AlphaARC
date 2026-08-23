"""Offline tests for the motor layer's pure logic. Run: python3 python/tests/test_motor.py"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from alphaarc.motor import AvatarMove, Navigator, astar, calibrate_kinematics, detect_avatar_move  # noqa: E402


class SimEnv:
    """A tiny simulated movement game: a 2x2 avatar (colour 9 with a 7 mark)
    that steps by unit vectors, blocked by walls (colour 5) or the border. An
    optional stationary same-colour decoy tests avatar tracking."""

    MOVES = {1: (-1, 0), 2: (1, 0), 3: (0, -1), 4: (0, 1)}  # up/down/left/right

    def __init__(self, H, W, bg, avatar_tl, walls, decoy_tl=None):
        self.H, self.W, self.bg = H, W, bg
        self.walls = set(walls)
        self.pos = avatar_tl
        self.decoy = decoy_tl

    def _footprint(self, tl):
        r, c = tl
        return [(r, c), (r, c + 1), (r + 1, c), (r + 1, c + 1)]

    def _blocked(self, tl):
        for (r, c) in self._footprint(tl):
            if not (0 <= r < self.H and 0 <= c < self.W) or (r, c) in self.walls:
                return True
        return False

    def grid(self):
        g = [[self.bg] * self.W for _ in range(self.H)]
        for (r, c) in self.walls:
            g[r][c] = 5
        for blob in ([self.pos] + ([self.decoy] if self.decoy else [])):
            r, c = blob
            g[r][c] = 9
            g[r][c + 1] = 9
            g[r + 1][c] = 9
            g[r + 1][c + 1] = 7
        return g

    def reset(self):
        return self.grid()

    def step(self, action):
        dr, dc = self.MOVES[action]
        nt = (self.pos[0] + dr, self.pos[1] + dc)
        if not self._blocked(nt):
            self.pos = nt
        return self.grid()


def _bg(h, w, b):
    return [[b] * w for _ in range(h)]


def _place(g, top, left, box):
    for r in range(len(box)):
        for c in range(len(box[r])):
            g[top + r][left + c] = box[r][c]


# A distinctive 2x2 avatar (colour 9 with an asymmetric mark so its shape key is
# stable) that we translate between frames.
AVATAR = [[9, 9], [9, 7]]


def test_detects_rightward_step():
    bg = 4
    before = _bg(8, 8, bg)
    _place(before, 3, 2, AVATAR)
    after = _bg(8, 8, bg)
    _place(after, 3, 3, AVATAR)  # moved right one column
    mv = detect_avatar_move(before, after, bg)
    assert mv == AvatarMove(0, 1, 9), mv


def test_detects_upward_step():
    bg = 4
    before = _bg(8, 8, bg)
    _place(before, 4, 4, AVATAR)
    after = _bg(8, 8, bg)
    _place(after, 2, 4, AVATAR)  # moved up two rows
    mv = detect_avatar_move(before, after, bg)
    assert mv == AvatarMove(-2, 0, 9), mv


def test_no_move_returns_none():
    bg = 4
    g = _bg(8, 8, bg)
    _place(g, 3, 3, AVATAR)
    assert detect_avatar_move(g, [row[:] for row in g], bg) is None


def test_astar_straight_line():
    # 4-direction moves; empty grid; path from (2,2) to (2,5) = three "right".
    moves = {1: (-1, 0), 2: (1, 0), 3: (0, -1), 4: (0, 1)}
    path = astar((2, 2), (2, 5), moves, lambda r, c: 0 <= r < 8 and 0 <= c < 8)
    assert path == [4, 4, 4], path


def test_astar_routes_around_wall():
    # A vertical wall at column 4 (rows 0..6) with a gap at row 7; avatar must
    # detour down through the gap to reach the far side.
    walls = {(r, 4) for r in range(0, 7)}
    moves = {1: (-1, 0), 2: (1, 0), 3: (0, -1), 4: (0, 1)}

    def walkable(r, c):
        return 0 <= r < 8 and 0 <= c < 8 and (r, c) not in walls

    path = astar((3, 3), (3, 5), moves, walkable)
    assert path is not None, "should find a detour"
    # verify the path actually reaches the goal and never steps on a wall
    r, c = 3, 3
    for a in path:
        dr, dc = moves[a]
        r, c = r + dr, c + dc
        assert walkable(r, c), "stepped onto a wall at %s" % ((r, c),)
    assert (r, c) == (3, 5), "ended at %s" % ((r, c),)


def test_astar_unreachable_returns_none():
    # Goal fully walled off.
    walls = {(r, 4) for r in range(8)}
    moves = {1: (-1, 0), 2: (1, 0), 3: (0, -1), 4: (0, 1)}

    def walkable(r, c):
        return 0 <= r < 8 and 0 <= c < 8 and (r, c) not in walls

    assert astar((3, 3), (3, 5), moves, walkable) is None


def test_calibrate_votes_out_false_positive():
    """3 actions move the avatar (colour 9); 1 blocked action yields a spurious
    colour-8 detection. Voting must pick avatar=9 and drop the colour-8 action."""
    bg = 4
    positions = {
        "start": (5, 5),
        1: (3, 5),   # up
        3: (5, 3),   # left
        4: (5, 7),   # right
        2: None,     # blocked (down) -> we'll fake a counter blip instead
    }

    def frame_with_avatar_at(pos):
        g = _bg(12, 12, bg)
        _place(g, pos[0], pos[1], AVATAR)  # colour-9 avatar
        # a per-move counter blip elsewhere (colour 8), 2 identical marks that
        # shift by a step -> a decoy the naive detector could grab.
        g[0][0] = 8
        g[0][1] = 8
        return g

    def reset_fn():
        return frame_with_avatar_at(positions["start"])

    def step_fn(aid):
        if aid == 2:  # blocked: avatar stays; only the colour-8 counter shifts
            g = frame_with_avatar_at(positions["start"])
            g[0][0] = bg
            g[0][2] = 8  # the 8-mark "moved" right by 1 -> decoy
            return g
        return frame_with_avatar_at(positions[aid])

    kin, avatar = calibrate_kinematics(reset_fn, step_fn, [1, 2, 3, 4], bg)
    assert avatar == 9, "avatar colour should win the vote, got %r" % avatar
    assert kin.get(1) == (-2, 0) and kin.get(3) == (0, -2) and kin.get(4) == (0, 2), kin
    assert 2 not in kin, "blocked action (colour-8 decoy) must be dropped: %r" % kin


def test_capstone_navigates_around_wall_to_arbitrary_cell():
    """The motor end-to-end: given an arbitrary empty target cell, the Navigator
    discovers the wall (zero-delta) and A*-routes the avatar to it -- with a
    same-colour decoy present to prove avatar tracking."""
    bg = 4
    # Vertical wall at col 5, rows 0..6 (2x2 avatar can slip through the rows 7-9 gap).
    walls = {(r, 5) for r in range(0, 7)} | {(r, 6) for r in range(0, 7)}
    env = SimEnv(11, 12, bg, avatar_tl=(2, 2), walls=walls, decoy_tl=(0, 10))
    kin = dict(SimEnv.MOVES)  # fully calibrated
    nav = Navigator(kin, avatar_color=9, bg=bg, all_actions=[1, 2, 3, 4])
    ok, pos = nav.navigate_to((2, 2), (2, 8), env.step, env.grid(), max_steps=400)
    assert ok and pos == (2, 8), "did not reach target: ok=%s pos=%s" % (ok, pos)
    assert nav.blocked, "should have discovered at least one wall cell"


def test_capstone_learns_a_blocked_direction_then_reaches():
    """Calibration only gave up/left/right (down was blocked at the start cell).
    The target is straight down; the Navigator must opportunistically probe the
    uncalibrated DOWN action, learn its vector, and reach."""
    bg = 4
    env = SimEnv(12, 8, bg, avatar_tl=(2, 2), walls=set())
    kin = {1: (-1, 0), 3: (0, -1), 4: (0, 1)}  # NO down (action 2)
    nav = Navigator(kin, avatar_color=9, bg=bg, all_actions=[1, 2, 3, 4])
    ok, pos = nav.navigate_to((2, 2), (7, 2), env.step, env.grid(), max_steps=400)
    assert ok and pos == (7, 2), "did not reach: ok=%s pos=%s" % (ok, pos)
    assert nav.kin.get(2) == (1, 0), "should have learned DOWN=(1,0): %s" % nav.kin


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
