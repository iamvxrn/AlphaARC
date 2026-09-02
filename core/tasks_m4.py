"""The MetaController benchmark task set. FROZEN BEFORE ANY POLICY CODE EXISTS.

This file is committed on its own, ahead of `metacontroller.py` and
`run_milestone4.py`, because a task set written while looking at a controller is a
task set shaped to flatter it. Nothing here imports a policy, and no policy is
allowed to import `Task.family` or any field marked JUDGE ONLY.

WHY NOT AN ABSTRACT EPISTEMIC MDP. The cheap version of this experiment defines a
task as "hidden epistemic state -> the one operation that advances it" and hands the
ORACLE arm that state. ORACLE then wins by construction and the number measures the
transition table its author wrote. That is a tautological benchmark and it was
rejected. Every epistemic state below is EMERGENT: it is computed at run time from
what the real M1-M3 mechanisms actually do (`MdlEngine` commits or abstains,
`resolve` returns UNIQUE or DEAD, `verify` refutes a universal or does not) on real
boards. Nothing in a task record says which operation is correct.

THE DESIGN PRINCIPLE, taken from what killed the first M3 decisive pair: do not vary
the intervention along the feature the mechanism keys on. So the set is built in
OFF-DIAGONAL PAIRS whose members are IDENTICAL in everything the agent can observe
at the decision point and differ only in ground truth:

    rule_intact   / rule_broken_hidden   same board, SAME MASK; they differ only in
                                        cells that are hidden at view 0. At view 0
                                        the two are literally the same evidence.
    ref_intact    / ref_replaced         `resolve()` answers UNIQUE on both. This is
                                        M3 PART 2's measured off-diagonal: the R0
                                        resolver cannot see the difference.

An arm that scores differently on the two members of a pair must be using something
other than the evidence -- which is exactly what ORACLE is allowed to do and HARD is
not. That gap is the quantity the experiment exists to measure, and it is
architectural, not injected noise.

THE FIVE CATEGORIES the benchmark is required to contain, and where they live:

  one operation is enough                  rule_intact       (over-sequencing costs)
  several needed, order barely matters     rule_loose
  a feedback-dependent transition          rule_broken_hidden, ref_replaced
  a harmful primitive is available         every rule task: TRACK discards a sound
                                           hypothesis; every ref task: EXPLORE has
                                           nothing left to reveal and only burns budget
  noise / control                          noise (no true hypothesis exists at all)

STATED LIMITATION, not to be discovered later in a summary: `ref_replaced` is solved
by TRACK and by nothing else, BY CONSTRUCTION. What it measures is therefore whether
an arm can tell that it needs TRACK -- not whether TRACK is discoverable. It is only
non-trivial because the other four families PUNISH reaching for TRACK, and because
`ref_intact` makes the need invisible to the observation channel.
"""
from __future__ import annotations

import copy
import random
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, List, Optional, Tuple

sys.path.insert(0, str(Path(__file__).resolve().parent))

from protocol import HIDDEN, Grid, Truth, Proposition, Kind, verify
from referent import Referent, components
import run_milestone3 as m3

BOARD_W = BOARD_H = 16
BREAK_ROW = 6                    # the row whose class-0 columns get overwritten


# --------------------------------------------------------------- the rule boards
def _tile(r: int, k: int) -> int:
    """A tile palindromic in `k`, so that BOTH universals hold on the intact board:
    mirror about axis 15 maps column class k to class (3-k)%4, so f(r,0)==f(r,3) and
    f(r,1)==f(r,2) makes the mirror exact, while g[r][c] = f(r, c%4) makes period 4
    exact. Two true rules, so refuting one still leaves something to find."""
    return ((r % 3) + 1) if k in (0, 3) else (((r + 1) % 3) + 1)


def rule_board(broken: bool) -> Grid:
    g = [[_tile(r, c % 4) for c in range(BOARD_W)] for r in range(BOARD_H)]
    if broken:
        # Overwrite EVERY class-0 column of one row together. Period 4 survives
        # untouched (all four periodic partners change as one); mirror about 15
        # breaks at exactly four pairs, because class 0 maps to class 3.
        v = (_tile(BREAK_ROW, 0) % 3) + 1
        assert v != _tile(BREAK_ROW, 0)
        for c in range(0, BOARD_W, 4):
            g[BREAK_ROW][c] = v
    return g


# The four cells that must be hidden at view 0 for the break to be invisible: one
# member of each broken mirror pair. Hiding the class-0 side also removes every
# period-4 comparison in that row, so neither rule sees a counterexample.
BREAK_CELLS: Tuple[Tuple[int, int], ...] = tuple((BREAK_ROW, c)
                                                 for c in range(0, BOARD_W, 4))

MIRROR_AXIS = BOARD_W - 1        # c + c' = 15
PERIOD = 4


def _mask(g: Grid, hide: List[Tuple[int, int]]) -> Grid:
    m = copy.deepcopy(g)
    for r, c in hide:
        m[r][c] = HIDDEN
    return m


def _random_cells(rng: random.Random, n: int, avoid: set) -> List[Tuple[int, int]]:
    pool = [(r, c) for r in range(BOARD_H) for c in range(BOARD_W)
            if (r, c) not in avoid]
    return rng.sample(pool, n)


def _noise_board(rng: random.Random) -> Grid:
    return [[(rng.randrange(1, 4) if rng.random() < 0.4 else 0)
             for _ in range(BOARD_W)] for _ in range(BOARD_H)]


# ------------------------------------------------------------------- the record
@dataclass(frozen=True)
class Task:
    tid: str
    family: str                      # JUDGE ONLY -- for per-family reporting
    kind: str                        # "rule" | "ref" | "noise"
    views: Tuple[Grid, ...]          # progressively revealed evidence
    frames: Tuple[Grid, ...]         # the observation history, for TRACK
    full: Optional[Grid] = None      # JUDGE ONLY -- the unmasked board
    true_rules: Tuple[Tuple[str, tuple], ...] = ()   # JUDGE ONLY
    ref_alive: Optional[bool] = None                 # JUDGE ONLY
    refs: Tuple[Referent, ...] = ()  # referents frozen on frames[0]


def _true_rules(g: Grid) -> Tuple[Tuple[str, tuple], ...]:
    """Ground truth, computed by the SAME verifier the agent is graded with, so the
    key cannot disagree with the grader."""
    out = []
    for rule, params in (("mirror", (MIRROR_AXIS,)), ("period", (PERIOD,))):
        p = Proposition(Kind.UNIVERSAL, rule, (rule, params))
        if verify(p, Truth(grid=g)) == 0.0:
            out.append((rule, params))
    return tuple(out)


def _ref_pair(frame0: Grid) -> Tuple[Referent, ...]:
    """A and B, frozen on the first frame, exactly as M3 froze them."""
    comps = {(*c["centroid"],): c for c in components(frame0)}
    by_pos = sorted(components(frame0), key=lambda c: (c["centroid"][0], c["centroid"][1]))
    return (Referent.from_component("A", by_pos[0]),
            Referent.from_component("B", by_pos[-1]))


def build(seed: int) -> List[Task]:
    """One instance of every family. The seed moves ONLY the random part of the
    masks; the off-diagonal pairs share their mask exactly, so the two members stay
    observationally identical at view 0 for every seed."""
    rng = random.Random(1000 + seed)
    out: List[Task] = []

    # --- the off-diagonal RULE pair -------------------------------------------
    # ONE mask, used for BOTH members. At view 0 the evidence is byte-identical.
    extra0 = _random_cells(rng, 40, avoid=set(BREAK_CELLS))
    hide0 = list(BREAK_CELLS) + extra0
    hide1 = extra0                      # view 1 reveals exactly the break cells
    for name, broken in (("rule_intact", False), ("rule_broken_hidden", True)):
        g = rule_board(broken)
        v0, v1 = _mask(g, hide0), _mask(g, hide1)
        assert _mask(rule_board(False), hide0) == _mask(rule_board(True), hide0), \
            "the off-diagonal pair must be indistinguishable at view 0"
        out.append(Task(f"{name}-s{seed}", name, "rule", (v0, v1), (v0,),
                        full=g, true_rules=_true_rules(g)))

    # --- order-insensitive: too little evidence to induce from at all ----------
    g = rule_board(False)
    heavy = _random_cells(rng, 215, avoid=set())
    light = _random_cells(rng, 30, avoid=set())
    out.append(Task(f"rule_loose-s{seed}", "rule_loose", "rule",
                    (_mask(g, heavy), _mask(g, light)), (_mask(g, heavy),),
                    full=g, true_rules=_true_rules(g)))

    # --- noise: no true hypothesis exists. Success is holding none. -----------
    ng = _noise_board(rng)
    nv = _mask(ng, _random_cells(rng, 30, avoid=set()))
    out.append(Task(f"noise-s{seed}", "noise", "noise", (nv, nv), (nv,),
                    full=ng, true_rules=_true_rules(ng)))

    # --- the off-diagonal REFERENT pair, straight from M3's fixtures -----------
    # `resolve()` answers UNIQUE on BOTH final frames -- that is M3 PART 2's
    # measured 0/3, and it is the whole reason HARD cannot route these apart.
    for name, hist, alive in (("ref_intact", m3.h_intact, True),
                              ("ref_replaced", m3.h_impostor, False)):
        frames = tuple(m3.frames_of(hist()))
        final = frames[-1]
        out.append(Task(f"{name}-s{seed}", name, "ref", (final,), frames,
                        full=final, ref_alive=alive, refs=_ref_pair(frames[0])))
    return out


FAMILIES = ("rule_intact", "rule_broken_hidden", "rule_loose", "noise",
            "ref_intact", "ref_replaced")
