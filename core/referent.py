"""Diagnostic only. NOT part of the protocol, and deliberately not imported by it.

The question this exists to ask:

    prediction failure  !=  representation failure ?

Right now both arrive as the same thing -- silence -- so an evaluator cannot tell
"my hypothesis was wrong" (revise the hypothesis) from "the thing my hypothesis was
about no longer exists" (revise the representation). These are opposite instructions.

Four reasons an engine can fail to produce a testable claim. They are labels attached
to an observation, not a new dispatch axis in `verify`:

    NO_MODEL              no hypothesis FORM applies to this evidence at all
    INSUFFICIENT_EVIDENCE the form applies, support is below the engine's floor
    REFERENT_LOST         a claim had a subject and the subject can no longer be found
    UNTESTABLE            a proposition was emitted, the verifier returned None


THE RULE THAT MAKES THIS HONEST
-------------------------------
A referent is created BEFORE the evidence under test and is then either re-resolved
or pronounced dead. The engine is NOT allowed to look at the new frame and nominate a
fresh B'. That is precisely the cheat measured in milestone 2 section C: allowed to
re-aim, the relational engine picked two identical fragments and scored "correct".

So `Referent` is frozen at construction and `resolve` applies a fixed rule. The rule
may inspect the new frame; it may not be chosen in response to it.


THE CIRCULARITY THIS MUST AVOID
-------------------------------
The claim under test is `identity(A.shape, B.shape)`. If B's referent were resolved
BY SHAPE, then any change of shape would kill the referent, and every MODEL_ERROR
would be misreported as REFERENT_LOST -- the diagnostic would answer its own question.

So resolution descriptors must be ORTHOGONAL to the claimed predicate. Here they are
`(colour, size, centroid)`: two 5-cell components can have different shapes, so size
does not determine shape. `shape` is recorded on the referent but is used ONLY to test
the claim afterwards, never to find the object.

    BUT `check_orthogonality` DOES NOT ESTABLISH THIS. It checks that the string
    "shape" is absent from RESOLVE_BY. That is SYNTACTIC orthogonality and nothing
    more. `size` and `centroid` are both functions of the object's geometry, as
    `shape` is; they are not independent of it, they merely fail to determine it.
    An intervention that damages shape very often damages size too, and then a
    resolver keyed on size will look like it is tracking identity while it is
    tracking size.

    This is not hypothetical -- it invalidated the first version of this experiment,
    whose two decisive cases differed precisely along `size`, so the verdicts were
    fixed by construction. The defence cannot live in this function. It has to live
    in the CASE DESIGN, by dissociating "identity survived" from "descriptors
    survived" and reading only the rows where the two disagree. See PART 2 of
    `run_milestone3.py`; PART 1 sidesteps descriptors altogether.
"""
from __future__ import annotations

from dataclasses import dataclass
from enum import Enum
from typing import List, Optional, Tuple

HIDDEN = -1
Grid = List[List[int]]


class Abstention(Enum):
    NO_MODEL = "no_model"
    INSUFFICIENT_EVIDENCE = "insufficient_evidence"
    REFERENT_LOST = "referent_lost"
    UNTESTABLE = "untestable"


class Resolution(Enum):
    UNIQUE = "unique"        # exactly one candidate; the referent survived
    AMBIGUOUS = "ambiguous"  # several equally good; identity is not determined
    DEAD = "dead"            # nothing matches the frozen descriptors


# --------------------------------------------------------------------- perception
def _background(g: Grid) -> int:
    counts = {}
    for row in g:
        for v in row:
            if v != HIDDEN:
                counts[v] = counts.get(v, 0) + 1
    if not counts:
        return 0
    best = max(counts.values())
    return min(v for v, n in counts.items() if n == best)


def components(g: Grid, lo: int = 1) -> List[dict]:
    """8-connected same-colour components, background excluded. `lo` is a floor on
    size; milestone 2 section C showed the floor itself is load-bearing, so it is a
    parameter here and is reported rather than buried."""
    bg = _background(g)
    seen = set()
    out = []
    for r in range(len(g)):
        for c in range(len(g[0])):
            if (r, c) in seen or g[r][c] in (bg, HIDDEN):
                continue
            col = g[r][c]
            st = [(r, c)]
            seen.add((r, c))
            cells = []
            while st:
                y, x = st.pop()
                cells.append((y, x))
                for dy in (-1, 0, 1):
                    for dx in (-1, 0, 1):
                        q = (y + dy, x + dx)
                        if (0 <= q[0] < len(g) and 0 <= q[1] < len(g[0])
                                and q not in seen and g[q[0]][q[1]] == col):
                            seen.add(q)
                            st.append(q)
            if len(cells) >= lo:
                ys = [p[0] for p in cells]
                xs = [p[1] for p in cells]
                y0, x0 = min(ys), min(xs)
                m = [[0] * (max(xs) - x0 + 1) for _ in range(max(ys) - y0 + 1)]
                for y, x in cells:
                    m[y - y0][x - x0] = 1
                out.append({
                    "colour": col,
                    "cells": frozenset(cells),
                    "size": len(cells),
                    "shape": tuple(tuple(row) for row in m),
                    "centroid": (sum(ys) / len(ys), sum(xs) / len(xs)),
                    "top_left": (y0, x0),
                })
    return out


# ----------------------------------------------------------------- the referent
@dataclass(frozen=True)
class Referent:
    """A name for an object, fixed BEFORE the evidence under test.

    `shape` is carried so the claim can be checked once the object is found. It takes
    no part in finding it -- see the module docstring on circularity."""
    name: str
    colour: int
    size: int
    centroid: Tuple[float, float]
    shape: tuple

    # descriptors that resolution is allowed to use
    RESOLVE_BY = ("colour", "size", "centroid")
    # ...and the property the claim is about, which it may not use
    CLAIMS_ABOUT = "shape"

    @staticmethod
    def from_component(name: str, comp: dict) -> "Referent":
        return Referent(name, comp["colour"], comp["size"],
                        comp["centroid"], comp["shape"])


def check_orthogonality() -> bool:
    """SYNTACTIC check only: the claimed property is not literally among the
    descriptors. Necessary, nowhere near sufficient -- `size` and `centroid` are
    geometry too. Passing this says only that the circularity is not the crude kind.
    See the module docstring; the real defence is in the case design."""
    return Referent.CLAIMS_ABOUT not in Referent.RESOLVE_BY


@dataclass
class ResolveResult:
    status: Resolution
    comp: Optional[dict] = None
    n_candidates: int = 0
    note: str = ""


def resolve(ref: Referent, g: Grid, size_tol: float = 0.25,
            radius: float = 6.0, lo: int = 1) -> ResolveResult:
    """Re-find `ref` in a new frame using ONLY its frozen orthogonal descriptors.

    Fixed rule, chosen before any frame is seen:
      1. candidates = same colour, size within `size_tol`, AND centroid within `radius`
      2. none         -> DEAD
      3. exactly one  -> UNIQUE
      4. several      -> AMBIGUOUS. Nearest-of-many is NOT allowed to break the tie;
                         that would let position quietly become the whole identity.

    Locality is a REQUIREMENT, not a tiebreaker, and that is load-bearing. Traced by
    hand on the dissolution case: once B has crumbled, the colour+size filter leaves A
    as the only survivor, so a "sole candidate wins" shortcut binds B's referent to
    object A fifteen cells away, compares A with itself, and reports UNTESTABLE. The
    referent would have drifted onto a different object and the diagnostic would have
    missed exactly the case it exists to catch.

    The cost is stated: an object that teleports further than `radius` is pronounced
    dead even though it is intact. This diagnostic cannot tell "gone" from "moved a
    long way", and it does not pretend to.
    """
    cands = [c for c in components(g, lo=lo)
             if c["colour"] == ref.colour
             and abs(c["size"] - ref.size) <= size_tol * ref.size
             and ((c["centroid"][0] - ref.centroid[0]) ** 2
                  + (c["centroid"][1] - ref.centroid[1]) ** 2) ** 0.5 <= radius]
    if not cands:
        return ResolveResult(Resolution.DEAD, None, 0,
                             "nothing of this colour and size near the remembered place")
    if len(cands) == 1:
        return ResolveResult(Resolution.UNIQUE, cands[0], 1,
                             "exactly one candidate matches the frozen descriptors")
    return ResolveResult(Resolution.AMBIGUOUS, None, len(cands),
                         f"{len(cands)} equally good candidates; identity undetermined")


# ------------------------------------------------------- the diagnostic verdict
def resolve_through(ref: Referent, frames: List[Grid], **kw) -> ResolveResult:
    """R1 -- the same rule as R0, chained across a history, updating the referent at
    every step. `frames[0]` is the frame the referent was frozen on.

    WHY A CHANGE-MASK WOULD NOT DO. The diff between the first and last frame is a
    FUNCTION of that pair, so it carries no information the pair does not already
    have, and PART 1's construction defeats it exactly as it defeats R0. Only genuine
    INTERMEDIATE states are new information. That is the whole content of "history"
    here, and it is why R1 is chained rather than merely diff-aware.

    The mechanism, minimal and entirely reused: identity survives iff it survives
    every single step. Two consequences fall out, and neither is free:

      - tolerance now applies PER STEP. An object that drifts far from its original
        descriptors while never changing much between consecutive frames stays
        tracked, which endpoint comparison cannot do.
      - a referent that dies at any step is dead even if something matching it
        reappears later. Reappearance is not resurrection.

    The cost: R1 needs the intermediate frames. Where the history is coarse it
    degrades toward R0, which `stride` in the runner measures rather than assumes.
    """
    if not frames:
        return ResolveResult(Resolution.DEAD, None, 0, "no frames")
    cur = ref
    for i, g in enumerate(frames):
        r = resolve(cur, g, **kw)
        if r.status is not Resolution.UNIQUE:
            return ResolveResult(r.status, None, r.n_candidates,
                                 f"lost at step {i}/{len(frames)-1}: {r.note}")
        cur = Referent.from_component(ref.name, r.comp)
    return ResolveResult(Resolution.UNIQUE, r.comp, r.n_candidates,
                         f"tracked through all {len(frames)-1} steps")


@dataclass
class Verdict:
    label: str                       # MODEL_ERROR | REPRESENTATION_ERROR | OK | ...
    abstention: Optional[Abstention]
    error: Optional[float]
    detail: str


def diagnose(ref_a: Referent, ref_b: Referent, g: Grid, **kw) -> Verdict:
    """Test `identity(A.shape, B.shape)` on a new frame, and when it cannot be tested,
    say WHY in a way that separates a wrong hypothesis from a lost subject.

    No ground truth about which object "is" B is consulted -- only the descriptors
    frozen before this frame existed.
    """
    ra, rb = resolve(ref_a, g, **kw), resolve(ref_b, g, **kw)
    for r, ref in ((ra, ref_a), (rb, ref_b)):
        if r.status is not Resolution.UNIQUE:
            return Verdict("REPRESENTATION_ERROR", Abstention.REFERENT_LOST, None,
                           f"{ref.name}: {r.status.value} -- {r.note}")
    if ra.comp["cells"] == rb.comp["cells"]:
        return Verdict("UNTESTABLE", Abstention.UNTESTABLE, None,
                       "both referents resolved to ONE object; 'X is identical to X' "
                       "is not a claim about the world")
    holds = ra.comp["shape"] == rb.comp["shape"]
    return Verdict("OK" if holds else "MODEL_ERROR", None, 0.0 if holds else 1.0,
                   "both referents tracked; the claim itself was "
                   + ("upheld" if holds else "REFUTED"))


def diagnose_through(ref_a: Referent, ref_b: Referent, frames: List[Grid],
                     **kw) -> Verdict:
    """Same verdict logic as `diagnose`, with R1 in place of R0. The claim is tested
    on the LAST frame, so R0 and R1 answer about the same board and differ only in
    what they were allowed to see on the way there."""
    ra = resolve_through(ref_a, frames, **kw)
    rb = resolve_through(ref_b, frames, **kw)
    for r, ref in ((ra, ref_a), (rb, ref_b)):
        if r.status is not Resolution.UNIQUE:
            return Verdict("REPRESENTATION_ERROR", Abstention.REFERENT_LOST, None,
                           f"{ref.name}: {r.status.value} -- {r.note}")
    if ra.comp["cells"] == rb.comp["cells"]:
        return Verdict("UNTESTABLE", Abstention.UNTESTABLE, None,
                       "both referents tracked to ONE object")
    holds = ra.comp["shape"] == rb.comp["shape"]
    return Verdict("OK" if holds else "MODEL_ERROR", None, 0.0 if holds else 1.0,
                   "both referents tracked through the history; the claim was "
                   + ("upheld" if holds else "REFUTED"))
