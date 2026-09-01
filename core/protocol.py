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

Milestone 2 adds a third rule, forced by measurement rather than by taste.

3. A held-out prediction over RANDOMLY masked cells is a WEAK instrument. Measured
   on the mirror board: a one-cell violation is caught only at the mask coverage
   rate -- 5.3% at 8 hidden cells, 17.3% at 24, 53.0% at 128 -- and it then FALLS to
   32.5% at 200, because masking hard enough to hit the damage also hides the cell
   the prediction would have been derived from. There is no mask size at which this
   instrument becomes reliable.

   So `UNIVERSAL` exists: a claim that a rule holds with no exception ANYWHERE. One
   counterexample refutes it, wherever it sits, so REFUTATION OF AN EMITTED UNIVERSAL
   IS NOT LIMITED TO HELD-OUT LOCATIONS -- measured 100% separation against 16.6%.

   That is the precise claim, and it is narrower than "universals are mask-independent",
   which would be false. The masking still governs the OTHER half of the loop: the
   hypothesis and its parameters -- which axis, which period -- are selected from the
   masked evidence, so what gets ASSERTED remains mask-dependent even though what
   REFUTES it does not. The two halves are visible separately in section D of
   `run_milestone2.py`: at a 1.00 commit threshold the engine still emitted 30
   universals that were refuted in full, because the mask had hidden the counterexample
   from the engine while leaving it in plain view of the verifier. Selection was fooled;
   refutation was not.

   The cost is real and is stated rather than hidden: to check a universal, the
   verifier must know what the rule MEANS, so it owns a small rule vocabulary
   (`_RULES`). It still cannot see which engine spoke or which task this is -- an
   engine names a schema and its parameters and cannot supply the checker, so it
   cannot grade itself -- but the verifier is no longer free of all content, and
   every rule added to that vocabulary is a claim the protocol makes on its own
   behalf. Keep the vocabulary small and general, and count it as a cost.
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
    UNIVERSAL = "universal"                  # "rule R holds with NO exception anywhere"


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


def _background(g: Grid) -> int:
    """Most frequent value. Reimplemented here, deliberately: the verifier must not
    share a code path with the engines it grades, or a bug agrees with itself."""
    counts: Dict[int, int] = {}
    for row in g:
        for v in row:
            if v != HIDDEN:
                counts[v] = counts.get(v, 0) + 1
    if not counts:
        return 0
    best = max(counts.values())
    return min(v for v, n in counts.items() if n == best)


def _rule_mirror(g: Grid, params: tuple) -> Tuple[int, int]:
    """cell(r,c) == cell(r, axis-c), everywhere. Returns (agreements, violations)."""
    (axis,) = params
    H, W = len(g), len(g[0])
    bg = _background(g)
    ok = bad = 0
    for r in range(H):
        for c in range(W):
            m = axis - c
            if not (0 <= m < W):
                continue
            if g[r][c] == HIDDEN or g[r][m] == HIDDEN:
                continue
            if g[r][c] == bg and g[r][m] == bg:
                # background agreeing with background is not evidence -- without this
                # a mostly-blank board satisfies every rule. The verifier owns this
                # judgement, which is exactly the content it was supposed not to have.
                continue
            ok += (g[r][c] == g[r][m])
            bad += (g[r][c] != g[r][m])
    return ok, bad


def _rule_period(g: Grid, params: tuple) -> Tuple[int, int]:
    """cell(r,c) == cell(r, c+period), everywhere."""
    (period,) = params
    H, W = len(g), len(g[0])
    bg = _background(g)
    ok = bad = 0
    for r in range(H):
        for c in range(W - period):
            if g[r][c] == HIDDEN or g[r][c + period] == HIDDEN:
                continue
            if g[r][c] == bg and g[r][c + period] == bg:
                continue
            ok += (g[r][c] == g[r][c + period])
            bad += (g[r][c] != g[r][c + period])
    return ok, bad


# The verifier's whole vocabulary. An engine names a key and supplies parameters; it
# cannot supply the checker, so it cannot grade itself.
_RULES = {"mirror": _rule_mirror, "period": _rule_period}


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

    if p.kind is Kind.UNIVERSAL:
        g = truth.grid
        if g is None:
            return None
        rule, params = p.value
        checker = _RULES.get(rule)
        if checker is None:
            return None                     # not in the vocabulary: untestable, not wrong
        ok, bad = checker(g, tuple(params))
        if ok + bad == 0:
            return None                     # the rule says nothing about this board
        return bad / (ok + bad)

    return None
