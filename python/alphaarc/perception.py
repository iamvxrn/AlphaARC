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

    Naming a control by the OBJECT under it -- colour, size as a FRACTION of the
    scene, and position in eighths of the frame -- is coarse on purpose: exact size
    and position are exactly what a redraw changes.
    """
    h, w = len(grid), len(grid[0]) if grid else 1
    colour = grid[y][x] if 0 <= y < h and 0 <= x < w else bg
    size, total = 0, 0
    for col, cells in _components(grid, bg):
        total += len(cells)
        if not size and col == colour and any(r == y and c == x for r, c in cells):
            size = len(cells)
    share = size / total if total else 0.0
    bucket = 0
    while bucket < 6 and share < 0.5 ** (bucket + 1):
        bucket += 1
    return "c%d/f%d/%d,%d" % (colour, bucket, (y * 8) // max(1, h), (x * 8) // max(1, w))
