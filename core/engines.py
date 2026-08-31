"""Adapters. Each turns an existing mechanism into propositions about UNSEEN cells.

An engine that cannot derive a consequence of its own claim returns []. Silence is
better than a 0.000 earned by predicting cells it was just shown.
"""
from __future__ import annotations

import sys
from pathlib import Path
from typing import Dict, List, Optional, Tuple

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "python"))
sys.path.insert(0, str(Path(__file__).resolve().parent))

from protocol import (HIDDEN, Claim, Evidence, Grid, Hypothesis, Kind,
                      Proposition, digest)
from alphaarc.perception import background_color


def _visible(g: Grid, r: int, c: int) -> bool:
    return 0 <= r < len(g) and 0 <= c < len(g[0]) and g[r][c] != HIDDEN


# ------------------------------------------------------------ A: MDL / regularity
class ReflectClaim:
    """cell(r,c) = cell(r, axis-c). A real consequence: it fills the holes."""
    def __init__(self, axis: int, agree: int):
        self.axis, self.agree = axis, agree

    def describe(self) -> str:
        return f"board is mirror-symmetric about column axis {self.axis/2:.1f}"

    def propose(self, ev: Evidence) -> List[Proposition]:
        g = ev.payload.get("grid") or ev.payload.get("before")
        if g is None:
            return []
        out: Dict[Tuple[int, int], int] = {}
        for (r, c) in ev.hidden:
            m = self.axis - c
            if _visible(g, r, m):
                out[(r, c)] = g[r][m]
        return [Proposition(Kind.HELD_OUT_CELLS, "mirror", out,
                            note="value of each hidden cell equals its mirror")] if out else []


class TranslateClaim:
    """cell(r,c) = cell(r, c-period). Also a real consequence."""
    def __init__(self, period: int, agree: int):
        self.period, self.agree = period, agree

    def describe(self) -> str:
        return f"board repeats with column period {self.period}"

    def propose(self, ev: Evidence) -> List[Proposition]:
        g = ev.payload.get("grid") or ev.payload.get("before")
        if g is None:
            return []
        out: Dict[Tuple[int, int], int] = {}
        for (r, c) in ev.hidden:
            for src in (c - self.period, c + self.period):
                if _visible(g, r, src):
                    out[(r, c)] = g[r][src]
                    break
        return [Proposition(Kind.HELD_OUT_CELLS, "period", out,
                            note="value of each hidden cell equals its translate")] if out else []


class MdlEngine:
    """Fits the two primitives that can generate consequences. The others -- Count,
    Correspondence as currently written -- produce a saving but no per-cell
    prediction, so they are not offered here at all rather than faked."""
    name = "mdl"

    def hypotheses(self, ev: Evidence) -> List[Hypothesis]:
        g = ev.payload.get("grid") or ev.payload.get("before")
        if g is None:
            return []
        H, W = len(g), len(g[0])
        bg = background_color([[v for v in row] for row in g])
        out: List[Hypothesis] = []
        best = (0, None)
        for axis in range(W - 1, 2 * W - 2):          # axis = c1 + c2
            ok = bad = 0
            for r in range(H):
                for c in range(W):
                    m = axis - c
                    if _visible(g, r, c) and _visible(g, r, m):
                        if g[r][c] == bg and g[r][m] == bg:
                            continue          # background agreeing with background is not evidence
                        ok += (g[r][c] == g[r][m]); bad += (g[r][c] != g[r][m])
            if ok >= 20 and bad == 0 and ok > best[0]:
                best = (ok, axis)
        if best[1] is not None:
            out.append(Hypothesis(self.name, ReflectClaim(best[1], best[0]),
                                  "whole board", min(0.99, best[0] / 400), 1.0, [ev.id]))
        bestp = (0, None)
        for period in range(2, W // 2):
            ok = bad = 0
            for r in range(H):
                for c in range(W - period):
                    if _visible(g, r, c) and _visible(g, r, c + period):
                        if g[r][c] == bg and g[r][c + period] == bg:
                            continue
                        ok += (g[r][c] == g[r][c + period]); bad += (g[r][c] != g[r][c + period])
            if ok >= 20 and bad == 0 and ok > bestp[0]:
                bestp = (ok, period)
        if bestp[1] is not None:
            out.append(Hypothesis(self.name, TranslateClaim(bestp[1], bestp[0]),
                                  "whole board", min(0.99, bestp[0] / 400), 1.0, [ev.id]))
        return out


# --------------------------------------------------------- B: relational / objects
class RelationNowClaim:
    def __init__(self, a, b):
        self.a, self.b = a, b

    def describe(self) -> str:
        return f"objects at {self.a} and {self.b} have identical shape (now)"

    def propose(self, ev: Evidence) -> List[Proposition]:
        return [Proposition(Kind.RELATION_NOW, f"{self.a}~{self.b}",
                            (self.a, self.b, "identity"),
                            note="statement about the current frame, not a forecast")]


class RelationPersistsClaim:
    def __init__(self, a, b):
        self.a, self.b = a, b

    def describe(self) -> str:
        return f"objects at {self.a} and {self.b} stay identical across the action"

    def propose(self, ev: Evidence) -> List[Proposition]:
        if ev.kind != "transition":
            return []
        return [Proposition(Kind.RELATION_PERSISTS, f"{self.a}~{self.b}",
                            (self.a, self.b, "identity"),
                            note="falsifiable by the transition")]


class RelationalEngine:
    name = "relational"

    @staticmethod
    def _mask(cells):
        ys = [c[0] for c in cells]; xs = [c[1] for c in cells]
        y0, x0 = min(ys), min(xs)
        m = [[0] * (max(xs) - x0 + 1) for _ in range(max(ys) - y0 + 1)]
        for r, c in cells:
            m[r - y0][c - x0] = 1
        return tuple(tuple(r) for r in m)

    def hypotheses(self, ev: Evidence) -> List[Hypothesis]:
        g = ev.payload.get("grid") or ev.payload.get("before")
        if g is None:
            return []
        bg = background_color([[v for v in row] for row in g])
        seen = set(); comps = []
        for r in range(len(g)):
            for c in range(len(g[0])):
                if (r, c) in seen or g[r][c] in (bg, HIDDEN):
                    continue
                col = g[r][c]; st = [(r, c)]; seen.add((r, c)); cells = []
                while st:
                    y, x = st.pop(); cells.append((y, x))
                    for dy in (-1, 0, 1):
                        for dx in (-1, 0, 1):
                            q = (y + dy, x + dx)
                            if (0 <= q[0] < len(g) and 0 <= q[1] < len(g[0])
                                    and q not in seen and g[q[0]][q[1]] == col):
                                seen.add(q); st.append(q)
                if 4 <= len(cells) <= 400:
                    comps.append((col, cells))
        by = {}
        for col, cells in comps:
            by.setdefault((col, self._mask(cells)), []).append(cells)
        out = []
        for (col, _), group in by.items():
            if len(group) < 2:
                continue
            tl = lambda cs: (min(p[0] for p in cs), min(p[1] for p in cs))
            a, b = tl(group[0]), tl(group[1])
            out.append(Hypothesis(self.name, RelationNowClaim(a, b),
                                  f"colour {col}", 0.7, 2.0, [ev.id]))
            out.append(Hypothesis(self.name, RelationPersistsClaim(a, b),
                                  f"colour {col}", 0.4, 2.0, [ev.id]))
        return out[:4]


# --------------------------------------------------------------- C: graph explorer
class TransitionClaim:
    def __init__(self, s: str, action, s2: str):
        self.s, self.action, self.s2 = s, action, s2

    def describe(self) -> str:
        return f"({self.s[:8]}.., {self.action}) -> {self.s2[:8]}.."

    def propose(self, ev: Evidence) -> List[Proposition]:
        return [Proposition(Kind.TRANSITION, f"{self.s}|{self.action}", self.s2,
                            note="edge seen before, asserted to repeat")]


class GraphEngine:
    name = "graph"

    def __init__(self):
        self.edges: Dict[Tuple[str, str], str] = {}

    def observe(self, before: Grid, action, after: Grid) -> None:
        self.edges[(digest(before), str(action))] = digest(after)

    def hypotheses(self, ev: Evidence) -> List[Hypothesis]:
        b = ev.payload.get("before"); act = ev.payload.get("action")
        if b is None or act is None:
            return []
        k = (digest(b), str(act))
        if k not in self.edges:
            return []
        return [Hypothesis(self.name, TransitionClaim(k[0], act, self.edges[k]),
                           "this board", 0.95, 1.0, [ev.id])]


def fresh_engines():
    return [MdlEngine(), RelationalEngine(), GraphEngine()]
