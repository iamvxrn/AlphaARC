"""Movement / motor layer for engine games (actions 1-5). Pure-logic core.

The agent doesn't know the avatar or which action moves which way -- it DISCOVERS
both by acting and watching. Two pure functions here (unit-testable on synthetic
frames):
  - detect_avatar_move(before, after): the object that kept its shape+colour but
    TRANSLATED = the avatar; returns its (dr, dc) displacement.
  - astar(...): shortest action sequence from the avatar cell to a target over
    walkable cells, using the discovered action->direction map.

The interactive glue (press each action, build the action->vector map) lives in
calibrate_kinematics; it's validated against a real game separately.
"""

from __future__ import annotations

import heapq
from collections import defaultdict
from typing import Callable, Dict, List, Optional, Tuple

from .mdl import Grid, _components, _shape_key

Cell = Tuple[int, int]  # (row, col)


class AvatarMove:
    __slots__ = ("dr", "dc", "color")

    def __init__(self, dr: int, dc: int, color: int):
        self.dr = dr
        self.dc = dc
        self.color = color

    def __repr__(self):
        return "AvatarMove(dr=%d,dc=%d,color=%d)" % (self.dr, self.dc, self.color)

    def __eq__(self, other):
        return isinstance(other, AvatarMove) and (self.dr, self.dc, self.color) == (other.dr, other.dc, other.color)


def _component_index(grid: Grid, bg: int):
    """Map (color, shape_key) -> list of (minr, minc, size) for each component."""
    idx = defaultdict(list)
    for color, cells in _components(grid, bg):
        key = _shape_key(color, cells)
        minr = min(c[0] for c in cells)
        minc = min(c[1] for c in cells)
        idx[(color, key)].append((minr, minc, len(cells)))
    return idx


def detect_avatar_move(before: Grid, after: Grid, bg: int, axis_aligned: bool = True) -> Optional[AvatarMove]:
    """The avatar = the component that kept its exact shape+colour but moved.

    Real frames are noisy (repeated 1-cell marks that coincidentally match a
    shifted copy; per-move health/counter displays that repaint cells every
    step). So this is deliberately picky:
      - axis_aligned: only pure horizontal/vertical translations count (a real
        step is not diagonal) -- kills the (1,1) coincidences;
      - among survivors, prefer the LARGEST moved component (the avatar block,
        not a stray mark), then the smallest translation.
    Returns None if nothing qualifies (blocked / non-motor / transformed)."""
    idx_after = _component_index(after, bg)
    best: Optional[AvatarMove] = None
    best_key = None  # (size, -dist) for comparison: larger size wins, then smaller dist
    for color, cells in _components(before, bg):
        key = _shape_key(color, cells)
        minr = min(c[0] for c in cells)
        minc = min(c[1] for c in cells)
        size = len(cells)
        for (ar, ac, _sz) in idx_after.get((color, key), []):
            dr, dc = ar - minr, ac - minc
            if dr == 0 and dc == 0:
                continue
            if axis_aligned and dr != 0 and dc != 0:
                continue
            cand_key = (size, -(abs(dr) + abs(dc)))
            if best_key is None or cand_key > best_key:
                best_key = cand_key
                best = AvatarMove(dr, dc, color)
    return best


def astar(start: Cell, goal: Cell, moves: Dict[int, Tuple[int, int]],
          walkable: Callable[[int, int], bool], max_nodes: int = 20000) -> Optional[List[int]]:
    """Shortest action sequence from start to goal.

    moves: action_id -> (dr, dc) unit step (the discovered kinematics map).
    walkable(r, c): may the avatar occupy this cell. Returns the list of action
    ids to execute, or None if unreachable within max_nodes."""
    if start == goal:
        return []

    def h(cell: Cell) -> int:
        return abs(cell[0] - goal[0]) + abs(cell[1] - goal[1])

    openq: List[Tuple[int, int, Cell]] = [(h(start), 0, start)]
    came: Dict[Cell, Tuple[Cell, int]] = {}
    gscore: Dict[Cell, int] = {start: 0}
    seen = 0
    while openq and seen < max_nodes:
        _, g, cur = heapq.heappop(openq)
        seen += 1
        if cur == goal:
            # reconstruct action list
            acts: List[int] = []
            node = cur
            while node in came:
                prev, act = came[node]
                acts.append(act)
                node = prev
            acts.reverse()
            return acts
        for aid, (dr, dc) in moves.items():
            if dr == 0 and dc == 0:
                continue
            nxt = (cur[0] + dr, cur[1] + dc)
            if not walkable(nxt[0], nxt[1]):
                continue
            ng = g + 1
            if ng < gscore.get(nxt, 1 << 30):
                gscore[nxt] = ng
                came[nxt] = (cur, aid)
                heapq.heappush(openq, (ng + h(nxt), ng, nxt))
    return None


def avatar_anchors(grid: Grid, bg: int, color: int) -> List[Cell]:
    """Top-left of every component of the avatar colour (there may be decoys --
    e.g. ls20's reference figures are also colour 9)."""
    out: List[Cell] = []
    for c, cells in _components(grid, bg):
        if c != color:
            continue
        out.append((min(x[0] for x in cells), min(x[1] for x in cells)))
    return out


def nearest_anchor(grid: Grid, bg: int, color: int, ref: Cell) -> Optional[Cell]:
    """The avatar-colour component whose top-left is nearest `ref` -- tracks the
    moving avatar across frames despite same-colour decoys."""
    best, bestd = None, 1 << 30
    for a in avatar_anchors(grid, bg, color):
        d = abs(a[0] - ref[0]) + abs(a[1] - ref[1])
        if d < bestd:
            best, bestd = a, d
    return best


class Navigator:
    """Drives the avatar to a target anchor cell over an UNKNOWN map, discovering
    walls as it goes (optimistic A* + replanning): assume unknown cells walkable,
    execute the first step, and if the avatar didn't move (zero-delta) mark that
    cell a wall and replan. If it gets stuck with no path, it opportunistically
    probes any not-yet-calibrated action (a direction that was blocked at the
    start cell) -- once the avatar is elsewhere that action may reveal its vector."""

    def __init__(self, kin: Dict[int, Tuple[int, int]], avatar_color: int, bg: int,
                 all_actions: Optional[List[int]] = None, bounds: Optional[Tuple[int, int]] = None):
        self.kin = dict(kin)
        self.color = avatar_color
        self.bg = bg
        self.all_actions = list(all_actions) if all_actions else list(kin.keys())
        self.bounds = bounds  # (H, W); if None, inferred from the grid each step
        self.blocked = set()  # anchor cells known to be non-standable

    def _walkable(self, H: int, W: int) -> Callable[[int, int], bool]:
        def w(r: int, c: int) -> bool:
            return 0 <= r < H and 0 <= c < W and (r, c) not in self.blocked
        return w

    def _dims(self, grid: Grid) -> Tuple[int, int]:
        return self.bounds if self.bounds else (len(grid), len(grid[0]))

    def navigate_to(self, start: Cell, target: Cell, step_fn: Callable[[int], Grid],
                    grid: Grid, max_steps: int = 400) -> Tuple[bool, Cell]:
        pos = start
        for _ in range(max_steps):
            if pos == target:
                return True, pos
            H, W = self._dims(grid)
            path = astar(pos, target, self.kin, self._walkable(H, W))
            if not path:
                # Stuck: try to learn an uncalibrated direction from here.
                if not self._probe_unknown(step_fn, grid, pos):
                    return False, pos
                # _probe_unknown updated kin/pos/grid via return; refetch
                grid = self._last_grid
                pos = self._last_pos
                continue
            a = path[0]
            dr, dc = self.kin[a]
            expected = (pos[0] + dr, pos[1] + dc)
            grid = step_fn(a)
            newpos = nearest_anchor(grid, self.bg, self.color, expected) or pos
            if newpos == pos:  # zero-delta -> the cell we tried to enter is a wall
                self.blocked.add(expected)
            else:
                pos = newpos
        return pos == target, pos

    def _probe_unknown(self, step_fn: Callable[[int], Grid], grid: Grid, pos: Cell) -> bool:
        """Try each not-yet-calibrated action once; if one moves the avatar, learn
        its vector. Returns True if it made progress (learned a move or moved)."""
        for a in self.all_actions:
            if a in self.kin:
                continue
            g = step_fn(a)
            newpos = nearest_anchor(g, self.bg, self.color, pos) or pos
            if newpos != pos:
                self.kin[a] = (newpos[0] - pos[0], newpos[1] - pos[1])
                self._last_grid, self._last_pos = g, newpos
                return True
        return False


def calibrate_kinematics(reset_fn: Callable[[], Grid], step_fn: Callable[[int], Grid],
                         action_ids: List[int], bg: int) -> Tuple[Dict[int, Tuple[int, int]], Optional[int]]:
    """Discover the avatar colour and the action->direction map, by acting.

    reset_fn() -> grid at start; step_fn(action_id) -> grid after that action.
    Real frames have confounders (per-move counters/health displays), so a single
    detection can be a false positive on a blocked action. We therefore VOTE: the
    avatar is the colour whose move is seen for the most actions, and we keep only
    moves of THAT colour. Actions with no avatar-coloured move (blocked at the
    start cell, or non-motor) are omitted -- their direction stays unknown until
    the avatar is somewhere that action isn't blocked.

    Returns (kinematics map action_id -> (dr, dc), avatar_color or None)."""
    raw: Dict[int, AvatarMove] = {}
    votes: Dict[int, int] = {}
    for aid in action_ids:
        before = reset_fn()
        after = step_fn(aid)
        mv = detect_avatar_move(before, after, bg)
        if mv is not None:
            raw[aid] = mv
            votes[mv.color] = votes.get(mv.color, 0) + 1
    if not votes:
        return {}, None
    avatar_color = max(votes, key=lambda c: votes[c])
    kin = {aid: (mv.dr, mv.dc) for aid, mv in raw.items() if mv.color == avatar_color}
    return kin, avatar_color
