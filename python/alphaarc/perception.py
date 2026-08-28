"""Minimal perception helpers for the Python submission core."""

from __future__ import annotations

from collections import Counter

from .mdl import Grid, _components


def background_color(grid: Grid) -> int:
    """Most frequent cell value = background (mirrors perception.BackgroundColor)."""
    c = Counter(v for row in grid for v in row)
    if not c:
        return 0
    # Counter.most_common ties by insertion; make it deterministic (lowest colour
    # among the max count) to match a stable choice.
    best_n = max(c.values())
    return min(v for v, n in c.items() if n == best_n)


def _size_bucket(size: int, total: int) -> int:
    """Size as a FRACTION of the scene, in halvings. Exact size is what a redraw
    changes; the fraction is what it preserves."""
    share = size / total if total else 0.0
    b = 0
    while b < 6 and share < 0.5 ** (b + 1):
        b += 1
    return b


def control_signature(grid: Grid, bg: int, x: int, y: int) -> str:
    """A name for a control that survives the board being REDRAWN or the level changing.

    Pixel coordinates are not a name. vc33 rescales its whole scene on every press,
    so instrumented over 30 clicks the planner hit 30 distinct positions and never
    repeated one; and across a level transition the layout changes wholesale, so a
    coordinate-keyed value is not merely useless but actively misleading -- the
    coordinate that meant "the shrink button" on level 1 is scenery on level 2.

    That matters more than anywhere else, because the score weights a level by its
    INDEX. Measured on vc33: level 1 falls in 6 actions against a baseline of 7 (a
    capped 115 points), and then level 2 takes 68 against a baseline of 18. The
    mechanic did not change between them; only our ability to carry what we learned.

    The name is therefore what the object IS -- its colour and its size as a
    fraction of the scene -- plus its RANK AMONG ITS PEERS: how many objects of the
    same colour and the same size bucket come before it in reading order.

    Rank rather than position, because position in eighths of the frame was
    measured NOT to survive the seam it was built for. vc33's two colour-9 buttons
    sit at eighth (3,7) and (4,7) on level 1 and at (3,0) and (4,0) on level 2 --
    same colour, same size bucket, same rows, MIRRORED column -- so an
    eighths-keyed name calls them four different controls and the mechanic is
    learned twice. Their rank among the colour-9 objects is 0 and 1 on both levels,
    because mirroring the pair does not reorder it.

    Rank does not collapse distinct objects the way dropping position entirely
    would: two identical boxes on the same board are peer 0 and peer 1, which is
    exactly the distinction the coordinate was there to make.
    """
    h, w = len(grid), len(grid[0]) if grid else 1
    colour = grid[y][x] if 0 <= y < h and 0 <= x < w else bg
    comps = list(_components(grid, bg))
    total = sum(len(cells) for _, cells in comps)
    mine, peers = None, []
    for col, cells in comps:
        if col != colour:
            continue
        # The top-left-most cell, not the centroid: an object that grows or is
        # partly repainted keeps its anchor, and only the ORDER of anchors is read.
        anchor = min(cells)
        entry = (_size_bucket(len(cells), total), anchor)
        peers.append(entry)
        if mine is None and any(r == y and c == x for r, c in cells):
            mine = entry
    if mine is None:
        # A click on background names no object. Kept distinct from every real
        # control rather than folded onto one, so it cannot absorb their values.
        return "c%d/bg" % colour
    bucket, anchor = mine
    rank = sum(1 for b, a in peers if b == bucket and a < anchor)
    return "c%d/f%d/#%d" % (colour, bucket, rank)
