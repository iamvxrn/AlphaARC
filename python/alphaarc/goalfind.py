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
    _subgrid,
)

# A discovered task: the target subgrid + its incomplete peers as
# (peer_top_row, peer_top_col, peer_subgrid).
MatchTask = Tuple[Grid, List[Tuple[int, int, Grid]]]


def _non_bg(sub: Grid, bg: int) -> int:
    return sum(1 for row in sub for v in row if v != bg)


def _hole_cells(sub: Grid, ref: Grid, bg: int) -> List[Cell]:
    """Local cells where the peer is BACKGROUND but the exemplar has a colour --
    i.e. the peer is genuinely INCOMPLETE there (a hole to fill). Differing
    FOREGROUND colours are NOT holes: two same-size regions that merely differ in
    colour are variants (a palette/legend), not an incomplete copy -- treating
    them as a fill task is a false positive (seen on lp85: two colour-3 bands,
    diffs 3->5, no actuator possible). This guard keeps goalfind honest."""
    out: List[Cell] = []
    for r in range(min(len(sub), len(ref))):
        for c in range(min(len(sub[r]), len(ref[r]))):
            if sub[r][c] == bg and ref[r][c] != bg:
                out.append((r, c))
    return out


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
        # A peer is a fill-workspace only if it has HOLES (bg where ref has
        # colour); foreground-vs-foreground differences mean it's a variant, not
        # an incomplete copy -> skip (false-positive guard).
        peers = [(members[i][0], members[i][1], subs[i])
                 for i in range(len(members)) if i != ref_idx
                 and _hole_cells(subs[i], ref, bg)]
        if peers:
            tasks.append((ref, peers))
    return tasks


def next_fix(grid: Grid, bg: int) -> Optional[Tuple[int, int, int]]:
    """The next goal-directed edit: (row, col, target_colour) -- set this absolute
    cell to match its exemplar, reducing mismatch. None when every peer already
    matches its target (task complete)."""
    for ref, peers in discover_match_tasks(grid, bg):
        for (minr, minc, sub) in peers:
            holes = _hole_cells(sub, ref, bg)
            if holes:
                r, c = holes[0]
                return (minr + r, minc + c, ref[r][c])
    return None
