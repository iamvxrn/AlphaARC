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
