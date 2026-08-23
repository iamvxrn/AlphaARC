"""Target discovery (rung 2) -- find the (target, workspace) pairing on a board,
so the mismatch drive has something to aim at.

Rung 1 (mismatch.py) proved the drive works when HANDED a target. The hard part
is finding the target on a real board. This first rung handles the most concrete
discoverable structure: repeated framed regions where one is COMPLETE (the
exemplar/target) and its peers are INCOMPLETE (the workspace) -- e.g. a legend
box vs the field boxes being filled to match it.

discover_match_tasks groups regions by the same (bbox, frame-colour) signature
(reusing Correspondence's anchor grouping), picks the fullest member as the
target, and returns the incomplete peers. next_fix turns that into one concrete
goal-directed edit: "cell (r,c) should become colour k" -- the action the agent
should take (a click that sets that cell), chosen to reduce mismatch.

Later rungs: non-framed legends, colour-swap/reflect/scaled targets, and object->
drawn-spec (ls20). This rung is identity-matching of framed exemplars.
"""

from __future__ import annotations

from typing import List, Optional, Tuple

from .mdl import Grid
from .correspondence import (
    Cell,
    _distinct_colors,
    _groups,
    _residual_identity_cells,
    _subgrid,
)

# A discovered task: the target subgrid + its incomplete peers as
# (peer_top_row, peer_top_col, peer_subgrid).
MatchTask = Tuple[Grid, List[Tuple[int, int, Grid]]]


def _non_bg(sub: Grid, bg: int) -> int:
    return sum(1 for row in sub for v in row if v != bg)


def discover_match_tasks(grid: Grid, bg: int) -> List[MatchTask]:
    """Repeated same-signature regions where one is the fullest (target) and the
    others are incomplete peers (workspace). Empty if no such structure."""
    tasks: List[MatchTask] = []
    for members in _groups(grid, bg).values():
        if len(members) < 2:
            continue
        subs = [_subgrid(grid, *m) for m in members]
        ref_idx = max(range(len(subs)), key=lambda i: _non_bg(subs[i], bg))
        ref = subs[ref_idx]
        if _distinct_colors(ref) < 2:
            continue  # a solid block is not a real exemplar (cheat guard)
        peers = [(members[i][0], members[i][1], subs[i])
                 for i in range(len(members)) if i != ref_idx
                 and _residual_identity_cells(subs[i], ref)]  # only peers that differ
        if peers:
            tasks.append((ref, peers))
    return tasks


def next_fix(grid: Grid, bg: int) -> Optional[Tuple[int, int, int]]:
    """The next goal-directed edit: (row, col, target_colour) -- set this absolute
    cell to match its exemplar, reducing mismatch. None when every peer already
    matches its target (task complete)."""
    for ref, peers in discover_match_tasks(grid, bg):
        for (minr, minc, sub) in peers:
            diffs = _residual_identity_cells(sub, ref)
            if diffs:
                r, c = diffs[0]
                return (minr + r, minc + c, ref[r][c])
    return None
