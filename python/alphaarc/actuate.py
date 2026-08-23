"""Actuation layer (Affordance Model), rung 1 -- Python port of Go pkg/actuate.

Bridges goalfind ("cell X should become colour k") and DOING it when controls
are indirect (the actuator is a button/tile elsewhere, not a click on X). See
the Go package for the canonical spec; this is the submission-side twin so the
whole pipeline composes in one language.
"""

from __future__ import annotations

from typing import List, Optional, Tuple

from .mdl import Grid


class Control:
    __slots__ = ("kind", "x", "y", "action_id")

    def __init__(self, kind: str = "click", x: int = 0, y: int = 0, action_id: int = 0):
        self.kind = kind
        self.x = x            # col (for click)
        self.y = y            # row (for click)
        self.action_id = action_id

    def __eq__(self, o):
        return isinstance(o, Control) and (self.kind, self.x, self.y, self.action_id) == (o.kind, o.x, o.y, o.action_id)

    def __hash__(self):
        return hash((self.kind, self.x, self.y, self.action_id))

    def __repr__(self):
        if self.kind == "click":
            return "Control(click %d,%d)" % (self.x, self.y)
        return "Control(action %d)" % self.action_id


class CellChange:
    __slots__ = ("r", "c", "frm", "to")

    def __init__(self, r: int, c: int, frm: int, to: int):
        self.r = r
        self.c = c
        self.frm = frm
        self.to = to

    def __repr__(self):
        return "CellChange(%d,%d %d->%d)" % (self.r, self.c, self.frm, self.to)


def diff(before: Grid, after: Grid) -> List[CellChange]:
    out: List[CellChange] = []
    for r in range(min(len(before), len(after))):
        for c in range(min(len(before[r]), len(after[r]))):
            if before[r][c] != after[r][c]:
                out.append(CellChange(r, c, before[r][c], after[r][c]))
    return out


class CausalMapper:
    def __init__(self):
        # observations: list of (Control, [CellChange])
        self.obs: List[Tuple[Control, List[CellChange]]] = []

    def explore(self, env, controls: List[Control]) -> None:
        """Try each control once from a fresh reset; record its effect."""
        for ctrl in controls:
            before = env.reset()
            after = env.step(ctrl)
            self.obs.append((ctrl, diff(before, after)))

    def observe(self, before: Grid, after: Grid, ctrl: Control) -> None:
        self.obs.append((ctrl, diff(before, after)))

    def control_for_cell(self, r: int, c: int, to: int) -> Optional[Control]:
        """Which control makes cell (r,c) become colour `to`? Smallest side-effect
        footprint wins (most precise actuator)."""
        best, best_fp = None, 1 << 30
        for ctrl, changes in self.obs:
            for ch in changes:
                if ch.r == r and ch.c == c and ch.to == to:
                    if len(changes) < best_fp:
                        best, best_fp = ctrl, len(changes)
                    break
        return best

    def control_for_color(self, to: int) -> Optional[Control]:
        """Which control sets SOME cell to colour `to` (attribute-ish changer)?"""
        for ctrl, changes in self.obs:
            for ch in changes:
                if ch.to == to:
                    return ctrl
        return None

    def plan(self, desired: List[CellChange]) -> Tuple[List[Control], List[CellChange]]:
        """Turn goalfind's desired cell changes into the controls that actuate
        them (deduped, in order). Unmet changes are returned for further
        exploration."""
        plan: List[Control] = []
        unmet: List[CellChange] = []
        for d in desired:
            ctrl = self.control_for_cell(d.r, d.c, d.to)
            if ctrl is None:
                unmet.append(d)
                continue
            if ctrl not in plan:
                plan.append(ctrl)
        return plan, unmet
