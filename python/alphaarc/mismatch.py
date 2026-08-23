"""Mismatch drive (rung 1) -- the goal signal for "make it look like the shown target".

The compression drive rewards SELF-regularity (symmetry/period/repeat). It is
blind to games whose goal is relational: "here is a target/legend drawn on the
board; act so the workspace MATCHES it" (tn36's field->legend, ls20's key->door
spec). Filling a workspace toward an arbitrary target usually does NOT raise
self-compression, so the compression drive gives no gradient there.

This module adds the complementary drive: measure the MISMATCH between a
workspace region and a target region (up to the same simple transforms
Correspondence allows), and reward actions that REDUCE it. mismatch_cells are
the "act here" attention (the cells that still disagree).

Rung 1 = the drive mechanic + a proof it guides a task the compression drive
cannot. Rung 2 (later) = DISCOVER the (workspace, target) pairing generally
instead of being handed it -- the hard, deferred part.
"""

from __future__ import annotations

from typing import List

from .mdl import Grid
from .correspondence import Cell, _min_residual, _min_residual_cells


def region_mismatch(workspace: Grid, target: Grid, bg: int) -> int:
    """Fewest cells the workspace must change to equal the target, under the best
    of {identity, colour-swap, reflect-H, reflect-V}. 0 == already matches."""
    return _min_residual(workspace, target, bg)


def mismatch_cells(workspace: Grid, target: Grid, bg: int) -> List[Cell]:
    """The LOCAL (row, col) cells in the workspace that still disagree with the
    target under the best transform -- the drive's attention / act-here list."""
    return _min_residual_cells(workspace, target, bg)


def mismatch_delta(before: Grid, after: Grid, target: Grid, bg: int) -> int:
    """Reward signal for model-free use: how much closer to the target the action
    got the workspace (positive = mismatch fell). The analogue of the compression
    drive's best_primitive_delta, but toward an external target."""
    return region_mismatch(before, target, bg) - region_mismatch(after, target, bg)
