"""Adapters. Each wraps an EXISTING mechanism and makes it say something testable.

No new reasoning is added here on purpose: the milestone is whether the three can
speak through one protocol at all, not whether they are good.
"""
from __future__ import annotations

import sys
from pathlib import Path
from typing import Any, Dict, List, Tuple

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "python"))

from protocol import (Claim, Engine, Evidence, Grid, Hypothesis, Level,
                      Prediction, PredictionSet)

from alphaarc.mdl import PRIMITIVES, _components
from alphaarc.perception import background_color


# --------------------------------------------------------------- A: MDL / rules
class MdlClaim:
    """"this board is compressible by primitive P, which predicts cells"."""
    def __init__(self, prim, savings: int, grid: Grid, bg: int):
        self.prim, self.savings, self.grid, self.bg = prim, savings, grid, bg

    def describe(self) -> str:
        return f"board is regular under {self.prim.name} (saves {self.savings} bits)"

    def predict(self, ctx: Evidence) -> PredictionSet:
        # A regularity claim predicts that the SAME regularity survives whatever
        # comes next: the cells it explains stay as they are.
        g = ctx.payload.get("before") or ctx.payload.get("grid")
        if g is None:
            return []
        cells = {(r, c): g[r][c] for r in range(0, len(g), 4) for c in range(0, len(g[r]), 4)}
        return [Prediction(Level.CELL, "board", cells,
                           note=f"{self.prim.name} regularity persists")]


class MdlEngine:
    name = "mdl"

    def hypotheses(self, ev: Evidence) -> List[Hypothesis]:
        g = ev.payload.get("before") or ev.payload.get("grid")
        if g is None:
            return []
        bg = background_color(g)
        out = []
        for p in PRIMITIVES:
            try:
                s = int(p.savings(g, bg))
            except Exception:
                continue
            if s <= 0:
                continue
            out.append(Hypothesis(self.name, MdlClaim(p, s, g, bg), scope="whole board",
                                  confidence=min(0.99, s / 200.0),
                                  complexity=1.0, evidence_ids=[ev.id]))
        return sorted(out, key=lambda h: -h.confidence)[:2]


# ------------------------------------------------------- B: relational / objects
class CorrespondenceClaim:
    def __init__(self, a, b, rel: str):
        self.a, self.b, self.rel = a, b, rel

    def describe(self) -> str:
        return f"object {self.a} corresponds to {self.b} under {self.rel}"

    def predict(self, ctx: Evidence) -> PredictionSet:
        return [Prediction(Level.RELATION, f"{self.a}~{self.b}", self.rel,
                           note="relation holds in the observed board")]


class RelationalEngine:
    """Groups same-colour components by shape and claims a relation between pairs."""
    name = "relational"

    @staticmethod
    def _mask(cells) -> Tuple[Tuple[int, ...], ...]:
        ys = [c[0] for c in cells]; xs = [c[1] for c in cells]
        y0, x0 = min(ys), min(xs)
        h, w = max(ys) - y0 + 1, max(xs) - x0 + 1
        m = [[0] * w for _ in range(h)]
        for r, c in cells:
            m[r - y0][c - x0] = 1
        return tuple(tuple(r) for r in m)

    def hypotheses(self, ev: Evidence) -> List[Hypothesis]:
        g = ev.payload.get("before") or ev.payload.get("grid")
        if g is None:
            return []
        bg = background_color(g)
        comps = [(col, cells) for col, cells in _components(g, bg) if 4 <= len(cells) <= 400]
        by: Dict[Any, List[Any]] = {}
        for col, cells in comps:
            m = self._mask(cells)
            by.setdefault((col, m), []).append(cells)
        out = []
        for (col, m), group in by.items():
            if len(group) < 2:
                continue
            def box(cs):
                ys = [p[0] for p in cs]; xs = [p[1] for p in cs]
                return (min(ys), min(xs))
            a, b = box(group[0]), box(group[1])
            out.append(Hypothesis(self.name, CorrespondenceClaim(a, b, "identity"),
                                  scope=f"colour {col}", confidence=0.6,
                                  complexity=2.0, evidence_ids=[ev.id]))
        return out[:2]


# ------------------------------------------------------------- C: graph explorer
class TransitionClaim:
    def __init__(self, s: int, action: Any, s2: int):
        self.s, self.action, self.s2 = s, action, s2

    def describe(self) -> str:
        return f"(state {self.s}, {self.action}) -> state {self.s2}"

    def predict(self, ctx: Evidence) -> PredictionSet:
        return [Prediction(Level.TRANSITION, f"{self.s}|{self.action}", self.s2,
                           note="edge already observed")]


class GraphEngine:
    """Remembers (state, action) -> state'. Understands nothing, predicts edges."""
    name = "graph"

    def __init__(self):
        self.edges: Dict[Tuple[int, str], int] = {}

    @staticmethod
    def _h(g: Grid) -> int:
        return hash(tuple(tuple(r) for i, r in enumerate(g) if i != 0))

    def observe(self, before: Grid, action: Any, after: Grid) -> None:
        self.edges[(self._h(before), str(action))] = self._h(after)

    def hypotheses(self, ev: Evidence) -> List[Hypothesis]:
        b = ev.payload.get("before"); act = ev.payload.get("action")
        if b is None or act is None:
            return []
        k = (self._h(b), str(act))
        if k not in self.edges:
            return []          # honest silence: this engine has nothing to say
        return [Hypothesis(self.name, TransitionClaim(k[0], act, self.edges[k]),
                           scope="this board", confidence=0.95, complexity=1.0,
                           evidence_ids=[ev.id])]


ENGINES = [MdlEngine(), RelationalEngine(), GraphEngine()]
