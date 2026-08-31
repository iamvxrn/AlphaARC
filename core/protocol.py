"""The protocol. No reasoning, no engine names, no task knowledge.

The bet:  claim -> explicit proposition about UNSEEN evidence -> generic verifier.

Two rules keep it honest.

1. A proposition must be about something the engine was not shown. Predicting the
   cells you were just handed measures nothing. So evidence is MASKED: engines see
   a grid with holes and are told where the holes are, never what is in them.

2. `verify` dispatches on the proposition's KIND alone. It cannot see which engine
   spoke, or which task this is. If it ever needs to, the protocol has failed.

`None` is a first-class result: not testable here. Silence and error are different
answers, and merging them is how an ensemble passes for reasoning.
"""
from __future__ import annotations

import hashlib
from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Dict, List, Optional, Protocol, Tuple

EvidenceId = str
Grid = List[List[int]]
HIDDEN = -1


class Kind(Enum):
    """What sort of proposition this is. The verifier knows only this."""
    HELD_OUT_CELLS = "held_out_cells"        # {(r,c): v} for cells the engine could not see
    RELATION_NOW = "relation_now"            # a relation asserted of the CURRENT frame
    RELATION_PERSISTS = "relation_persists"  # ...asserted to survive an action
    TRANSITION = "transition"                # (state_digest, action) -> state_digest


def digest(g: Grid, skip_rows: Tuple[int, ...] = (0,)) -> str:
    """Deterministic across processes. Python's hash() is salted; using it for a
    research fixture is a fine way to rediscover PYTHONHASHSEED as a cognitive
    phenomenon."""
    h = hashlib.sha256()
    for i, row in enumerate(g):
        if i in skip_rows:
            continue
        h.update(bytes(bytearray((v + 1) % 256 for v in row)))
    return h.hexdigest()[:16]


@dataclass(frozen=True)
class Evidence:
    """What an engine is allowed to see. `hidden` says WHERE the holes are."""
    id: EvidenceId
    kind: str                       # grid | transition
    payload: Dict[str, Any]
    hidden: Tuple[Tuple[int, int], ...] = ()


@dataclass(frozen=True)
class Proposition:
    kind: Kind
    target: str
    value: Any
    note: str = ""


class Claim(Protocol):
    def propose(self, ev: Evidence) -> List[Proposition]: ...
    def describe(self) -> str: ...


@dataclass
class Hypothesis:
    source: str
    claim: Claim
    scope: str
    confidence: float
    complexity: float
    evidence_ids: List[EvidenceId] = field(default_factory=list)

    def describe(self) -> str:
        return f"{self.claim.describe()}"


# ------------------------------------------------------------------- the verifier

@dataclass
class Truth:
    """What actually happened. Held by the runner, never inside Evidence."""
    grid: Optional[Grid] = None            # the un-masked frame the claim was about
    after: Optional[Grid] = None           # the frame following the action, if any


def _cells_at(g: Grid, top_left: Tuple[int, int]) -> Optional[frozenset]:
    """The connected component containing a point, or None if the point is a hole."""
    r0, c0 = top_left
    if not (0 <= r0 < len(g) and 0 <= c0 < len(g[0])):
        return None
    col = g[r0][c0]
    if col == HIDDEN:
        return None
    seen = {(r0, c0)}; st = [(r0, c0)]
    while st:
        r, c = st.pop()
        for dr in (-1, 0, 1):
            for dc in (-1, 0, 1):
                q = (r + dr, c + dc)
                if (0 <= q[0] < len(g) and 0 <= q[1] < len(g[0])
                        and q not in seen and g[q[0]][q[1]] == col):
                    seen.add(q); st.append(q)
    return frozenset(seen)


def _shape_at(g: Grid, top_left: Tuple[int, int]) -> Optional[tuple]:
    r0, c0 = top_left
    if not (0 <= r0 < len(g) and 0 <= c0 < len(g[0])):
        return None
    col = g[r0][c0]
    if col == HIDDEN:
        return None
    seen = {(r0, c0)}; st = [(r0, c0)]; cells = []
    while st:
        r, c = st.pop(); cells.append((r, c))
        for dr in (-1, 0, 1):
            for dc in (-1, 0, 1):
                q = (r + dr, c + dc)
                if (0 <= q[0] < len(g) and 0 <= q[1] < len(g[0])
                        and q not in seen and g[q[0]][q[1]] == col):
                    seen.add(q); st.append(q)
    ys = [x[0] for x in cells]; xs = [x[1] for x in cells]
    y0, x0 = min(ys), min(xs)
    m = [[0] * (max(xs) - x0 + 1) for _ in range(max(ys) - y0 + 1)]
    for r, c in cells:
        m[r - y0][c - x0] = 1
    return tuple(tuple(r) for r in m)


def verify(p: Proposition, truth: Truth) -> Optional[float]:
    """Error in [0,1], or None when this truth cannot test this proposition."""
    if p.kind is Kind.HELD_OUT_CELLS:
        g = truth.grid
        if g is None or not p.value:
            return None
        wrong = sum(1 for (r, c), v in p.value.items() if g[r][c] != v)
        return wrong / len(p.value)

    if p.kind in (Kind.RELATION_NOW, Kind.RELATION_PERSISTS):
        g = truth.grid if p.kind is Kind.RELATION_NOW else truth.after
        if g is None:
            return None
        a, b, rel = p.value
        ca, cb = _cells_at(g, a), _cells_at(g, b)
        if ca is None or cb is None:
            return None
        if ca == cb:
            # the two names resolve to ONE object; "X is identical to X" is not a
            # claim about the world. Untestable, not correct.
            return None
        ma, mb = _shape_at(g, a), _shape_at(g, b)
        if ma is None or mb is None:
            return None
        holds = (ma == mb) if rel == "identity" else None
        if holds is None:
            return None
        return 0.0 if holds else 1.0

    if p.kind is Kind.TRANSITION:
        if truth.after is None:
            return None
        return 0.0 if p.value == digest(truth.after) else 1.0

    return None
