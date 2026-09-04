"""Telling a dead control from the move budget ticking.

Every policy here asks "did that click do anything?" the same way::

    if grid == self._prev_grid:      # nothing happened -> taboo it

and in most games that test can never be true, because every accepted action
ticks a move-budget strip drawn on the board: row 0 in vc33, column 0 in r11l,
row 63 in ft09, g50t, s5i5 and su15. So the boards always differ, inhibition of
return never fires, `RunPlanner.inert` is never set, and `HybridPolicy`'s
"switch after five dead clicks" is unreachable code -- only its 25-action clock
ever fires.

That is not theoretical. Traced on vc33 level 2, seed 4: **26 consecutive actions**
clicking the same four controls, every one returning a compression delta of -1 or
-2 -- the clock, and nothing else -- out of the 68 the level cost against a
baseline of 18. Level 1 had gone in 6 actions against a baseline of 7.

The same defect lived in `decode` and made every dead control read as live.

**This does not touch the compression measurement.** Cropping the HUD out of the
MEASUREMENT was tried and measured worse (see python/bench/README.md, Rejected
#3): the primitives are not invariant to grid dimensions, and the drift is
common-mode anyway. This is a different question -- "did anything happen?" rather
than "how compressible is it?" -- and the answer to it was simply wrong.

Finding the clock cannot rely on WHICH cells move: su15's budget bar is eaten two
cells at a time, so no cell repeats and an intersection over transitions is empty.
What is stable is the LINE it is drawn on. So count how often each row and each
column is touched, and treat a change confined to rows and columns that move on
almost every action as the clock. A game with no strip (tn36) simply never has a
line at that frequency, and the test falls back to exact equality -- the old
behaviour, which is correct there.
"""

from __future__ import annotations

from typing import List, Optional

from .mdl import Grid


class ClockTracker:
    """Learns which rows/columns are the move-budget strip, and ignores them.

    Never asked to identify the strip in the abstract: it only answers whether a
    particular transition was nothing but the clock.
    """

    #: a line must move on at least this fraction of transitions to be the clock
    SHARE = 0.8
    #: ...and we must have seen at least this many before believing any of it
    WARMUP = 8

    def __init__(self) -> None:
        self.transitions = 0
        self.row_hits: List[int] = []
        self.col_hits: List[int] = []

    def _fit(self, grid: Grid) -> None:
        h, w = len(grid), max((len(r) for r in grid), default=0)
        if len(self.row_hits) < h:
            self.row_hits.extend([0] * (h - len(self.row_hits)))
        if len(self.col_hits) < w:
            self.col_hits.extend([0] * (w - len(self.col_hits)))

    def strip(self):
        """Rows/cols currently believed to BE the move-budget strip.

        Exposed so a state can be hashed without it: the strip ticks on every
        action, so a raw board hash makes every state look new and any
        "am I revisiting?" measurement trivially answers no."""
        n = max(1, self.transitions)
        rows = {r for r in range(len(self.row_hits))
                if self.row_hits[r] >= self.SHARE * n}
        cols = {c for c in range(len(self.col_hits))
                if self.col_hits[c] >= self.SHARE * n}
        return rows, cols

    def clock_only(self, before: Optional[Grid], after: Grid) -> bool:
        """True if `after` differs from `before` only in clock lines (or not at all).

        Also records the transition, so call it exactly once per step.
        """
        if before is None:
            return False
        # A board that changed SHAPE certainly did something -- vc33 rescales its
        # whole scene on a press -- and comparing only the overlap would call that
        # nothing at all. (Caught by test_a_key_run_survives_a_redraw_by_construction.)
        if len(before) != len(after) or any(
                len(a) != len(b) for a, b in zip(before, after)):
            return False
        self._fit(after)
        changed = [(r, c)
                   for r in range(min(len(before), len(after)))
                   for c in range(min(len(before[r]), len(after[r])))
                   if before[r][c] != after[r][c]]
        # Judge with what we knew BEFORE this transition, then learn from it.
        known = self.transitions >= self.WARMUP
        rows = {r for r in range(len(self.row_hits))
                if self.row_hits[r] >= self.SHARE * self.transitions}
        cols = {c for c in range(len(self.col_hits))
                if self.col_hits[c] >= self.SHARE * self.transitions}
        self.transitions += 1
        for r in {p[0] for p in changed}:
            self.row_hits[r] += 1
        for c in {p[1] for p in changed}:
            self.col_hits[c] += 1
        if not changed:
            return True
        if not known:
            return False
        return all(r in rows or c in cols for r, c in changed)
