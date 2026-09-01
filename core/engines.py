"""Adapters. Each turns an existing mechanism into propositions about UNSEEN cells.

An engine that cannot derive a consequence of its own claim returns []. Silence is
better than a 0.000 earned by predicting cells it was just shown.

Milestone 2: abstention vs falsifiability, resolved SEPARATELY for each family,
because the three abstain for different reasons and only one of them was a gate.

  MDL         -- abstained because `bad == 0` demanded a PERFECT symmetry, so damage
                 removed the hypothesis. Fixed: fit tolerantly, then commit to the
                 BOLD universal ("no exceptions anywhere") when agreement clears
                 COMMIT_AGREE. Bold because it is refutable by one cell; a hedged
                 "symmetric except at these two cells" is unfalsifiable and, in MDL's
                 own currency, longer. Measured: 100% separation, silent on noise.

  relational  -- did NOT abstain because of a gate, and loosening it is measured to
                 make things WORSE, not better. See RelationalEngine.

  graph       -- never abstained; it was already falsifiable at milestone 1.5.
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


class UniversalClaim:
    """"Rule R holds with NO exception anywhere on this board."

    The bold form. It is refuted by a single violating cell, including cells the
    engine never saw, so REFUTATION is not confined to the held-out locations --
    which is the whole reason it exists (see protocol.py, rule 3).

    Not the same as being mask-independent. `self.rule` and `self.params` were chosen
    by fitting the MASKED board, so what this claim asserts is still a function of
    what the mask happened to reveal; only what can refute it is not."""
    def __init__(self, rule: str, params: tuple, agree: float, ok: int, bad: int):
        self.rule, self.params = rule, params
        self.agree, self.ok, self.bad = agree, ok, bad

    def describe(self) -> str:
        return (f"{self.rule}{self.params} holds EVERYWHERE "
                f"(fit {self.ok}/{self.ok + self.bad})")

    def propose(self, ev: Evidence) -> List[Proposition]:
        return [Proposition(Kind.UNIVERSAL, self.rule, (self.rule, self.params),
                            note="one counterexample anywhere refutes this")]


# An engine commits to the bold universal above this in-sample agreement, and stays
# silent below it. Stated, not derived -- and the runner sweeps it, because an
# unearned constant is how this project has been burned before.
COMMIT_AGREE = 0.90
MIN_AGREE = 20          # too little evidence to be bold about


class MdlEngine:
    """Fits the two primitives that can generate consequences. The others -- Count,
    Correspondence as currently written -- produce a saving but no per-cell
    prediction, so they are not offered here at all rather than faked."""
    name = "mdl"

    def __init__(self, commit_agree: float = COMMIT_AGREE, min_agree: int = MIN_AGREE):
        self.commit_agree, self.min_agree = commit_agree, min_agree

    def hypotheses(self, ev: Evidence) -> List[Hypothesis]:
        g = ev.payload.get("grid") or ev.payload.get("before")
        if g is None:
            return []
        H, W = len(g), len(g[0])
        bg = background_color([[v for v in row] for row in g])
        out: List[Hypothesis] = []
        best = (0.0, None, 0, 0)
        for axis in range(W - 1, 2 * W - 2):          # axis = c1 + c2
            ok = bad = 0
            for r in range(H):
                for c in range(W):
                    m = axis - c
                    if _visible(g, r, c) and _visible(g, r, m):
                        if g[r][c] == bg and g[r][m] == bg:
                            continue          # background agreeing with background is not evidence
                        ok += (g[r][c] == g[r][m]); bad += (g[r][c] != g[r][m])
            if ok >= self.min_agree and ok + bad > 0 and ok / (ok + bad) > best[0]:
                best = (ok / (ok + bad), axis, ok, bad)
        if best[1] is not None and best[0] >= self.commit_agree:
            agree, axis, ok, bad = best
            # the weak instrument, kept so the two can be compared in one run
            out.append(Hypothesis(self.name, ReflectClaim(axis, ok),
                                  "whole board", agree, 1.0, [ev.id]))
            # the bold one
            out.append(Hypothesis(self.name, UniversalClaim("mirror", (axis,), agree, ok, bad),
                                  "whole board", agree, 1.0, [ev.id]))
        bestp = (0.0, None, 0, 0)
        for period in range(2, W // 2):
            ok = bad = 0
            for r in range(H):
                for c in range(W - period):
                    if _visible(g, r, c) and _visible(g, r, c + period):
                        if g[r][c] == bg and g[r][c + period] == bg:
                            continue
                        ok += (g[r][c] == g[r][c + period]); bad += (g[r][c] != g[r][c + period])
            if ok >= self.min_agree and ok + bad > 0 and ok / (ok + bad) > bestp[0]:
                bestp = (ok / (ok + bad), period, ok, bad)
        if bestp[1] is not None and bestp[0] >= self.commit_agree:
            agree, period, ok, bad = bestp
            out.append(Hypothesis(self.name, TranslateClaim(period, ok),
                                  "whole board", agree, 1.0, [ev.id]))
            out.append(Hypothesis(self.name, UniversalClaim("period", (period,), agree, ok, bad),
                                  "whole board", agree, 1.0, [ev.id]))
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
    """MEASURED, 2026-09-01: this engine's silence is NOT a gate that can be loosened,
    and the obvious fix is worse than the disease. Damage to a copy does not merely
    break the exact shape match -- it DISSOLVES the object. Deleting one cell of the
    5-cell copy leaves two 2-cell fragments that are no longer 8-connected, so the
    referent of the claim ceases to exist and there is nothing left to be wrong about.

    Lowering the size floor to 2 and pairing by nearest shape instead of exact match
    was tried and rejected on two independent counts:

      - it lets the claim DODGE refutation. On the broken board the engine re-aims at
        the two identical FRAGMENTS (similarity 1.000) instead of the original pair,
        and the verifier scores it CORRECT. An engine allowed to choose its own
        subject after seeing the evidence cannot be falsified by it.
      - it manufactures error from noise. On scattered same-colour blobs that are not
        copies of anything, it committed on 37 of 40 boards and was falsified on 23.

    So relational abstention stands, and it is a REPRESENTATIONAL limit, not a
    threshold: the family needs a notion of a damaged-but-still-identified object
    (an identity that survives its own cells changing) before it can be bold. That is
    the same wall as `Correspondence` in the Go stack, arriving from a new direction.
    """
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


# ------------------------------------------------------- D: control, not an engine
class LiarEngine:
    """Always commits to the best-looking axis, however bad the fit. Not a mechanism:
    the NOISE FLOOR. Error is only evidence of knowledge if an engine that knows
    nothing scores worse, so every run reports this alongside the real engines. If a
    real engine's error ever approaches the liar's, the difference between them was
    the thing being measured, and it is gone."""
    name = "liar"

    def hypotheses(self, ev: Evidence) -> List[Hypothesis]:
        g = ev.payload.get("grid") or ev.payload.get("before")
        if g is None:
            return []
        H, W = len(g), len(g[0])
        bg = background_color([[v for v in row] for row in g])
        best = (-1.0, None, 0, 0)
        for axis in range(W - 1, 2 * W - 2):
            ok = bad = 0
            for r in range(H):
                for c in range(W):
                    m = axis - c
                    if _visible(g, r, c) and _visible(g, r, m):
                        if g[r][c] == bg and g[r][m] == bg:
                            continue
                        ok += (g[r][c] == g[r][m]); bad += (g[r][c] != g[r][m])
            if ok + bad > 0 and ok / (ok + bad) > best[0]:
                best = (ok / (ok + bad), axis, ok, bad)
        if best[1] is None:
            return []
        agree, axis, ok, bad = best
        return [Hypothesis(self.name, UniversalClaim("mirror", (axis,), agree, ok, bad),
                           "whole board", agree, 1.0, [ev.id])]


def fresh_engines():
    return [MdlEngine(), RelationalEngine(), GraphEngine(), LiarEngine()]
