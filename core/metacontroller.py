"""MetaController v0 -- operations, epistemic labels, classifiers, policies.

Written AFTER `tasks_m4.py` was committed. See that file for the benchmark and for
why an abstract epistemic MDP was rejected.

FOUR OPERATIONS, NOT FIVE. The frozen spec names induce / predict / falsify /
explore / refine. REFINE IS NOT HERE, and its absence is deliberate: in this
substrate re-induction on new evidence strictly dominates parameter refinement --
`MdlEngine` selects by agreement ratio, so whenever a true rule is available and
unbroken it is already the argmax, and a separate "re-fit the parameters" operation
never has work to do. Shipping it anyway would pad the denominator of the RANDOM
control (1-in-5 instead of 1-in-4) and make typed routing look better for a reason
that has nothing to do with routing. Dropping it makes the experiment HARDER to
pass. What REFINE was for -- not proposing a hypothesis already refuted -- survives
as the `excluded` set carried by INDUCE.

RE_REPRESENT IS NOT AN OPERATION, per the frozen spec. TRACK is not a rename of it:
TRACK is M3's `resolve_through`, a measured mechanism with a measured envelope and a
measured failure (occlusion). It re-establishes reference; it does not invent a new
ontology.

THE ONE AUTHORED COST, DECLARED. TRACK on a task with no live referent question
discards the current hypothesis. That implements the spec's stated semantics --
"RE-REPRESENT in place of REFINE discards a sound hypothesis and rebuilds the
ontology" -- but it is authored, not measured, and it is the obvious way to
manufacture a win for careful routing. So `track_destroys` is a run-level switch and
the whole experiment is reported BOTH WAYS. If the ORACLE-over-RANDOM gap survives
only with it on, the gap is this penalty and must be reported as such.

WHERE HARD IS FORCED TO BE WRONG -- and it is forced, not handicapped. Two blind
spots, both inherited from measurements already in the bank:

  * M2 section D. A universal that no visible cell contradicts is observationally
    identical whether it is true or merely unrefuted so far. `hard_label` must
    answer SOLVED for both, so it cannot separate `rule_intact` from
    `rule_broken_hidden` -- which share a mask precisely so that nothing can.
  * M3 PART 1 / PART 2. R0 answers UNIQUE for an impostor, so a refuted claim looks
    like MODEL_ERROR when the truth is REPRESENTATION_ERROR.

Everything else observable IS given to HARD, including the inference that repeated
abstention at the fullest view means there is nothing here (UNTESTABLE). Withholding
that would have handed the noise family to ORACLE by construction and measured a
handicap rather than a channel.
"""
from __future__ import annotations

import random
import sys
from dataclasses import dataclass, field, replace
from enum import Enum
from itertools import permutations
from pathlib import Path
from typing import List, Optional, Set, Tuple

sys.path.insert(0, str(Path(__file__).resolve().parent))

from protocol import HIDDEN, Evidence, Kind, Proposition, Truth, verify
from engines import MdlEngine
from referent import Resolution, resolve, resolve_through
from tasks_m4 import Task

BUDGET = 12
SHAPE_EQ = ("shape_eq", ())


class Op(Enum):
    INDUCE = "induce"
    EXPLORE = "explore"
    FALSIFY = "falsify"
    TRACK = "track"
    STOP = "stop"


REAL_OPS = (Op.INDUCE, Op.EXPLORE, Op.FALSIFY, Op.TRACK)


class Label(Enum):
    SOLVED = "solved"
    NO_MODEL = "no_model"
    UNTESTED = "untested"
    MODEL_ERROR = "model_error"
    INSUFFICIENT_EVIDENCE = "insufficient_evidence"
    REPRESENTATION_ERROR = "representation_error"
    UNTESTABLE = "untestable"
    UNKNOWN = "unknown"


@dataclass
class State:
    view: int = 0
    hyp: Optional[Tuple[str, tuple]] = None
    tested_view: Optional[int] = None
    refuted: bool = False
    excluded: Set[Tuple[str, tuple]] = field(default_factory=set)
    refs_dead: bool = False
    tracked: bool = False
    induce_empty: int = 0
    induced_view: Optional[int] = None
    ops: int = 0
    stopped: bool = False


_ENGINE = MdlEngine()


def _hidden_of(g):
    return tuple((r, c) for r in range(len(g)) for c in range(len(g[0]))
                 if g[r][c] == HIDDEN)


def _pick(g, excluded):
    best, bestkey = None, None
    for i, h in enumerate(_ENGINE.hypotheses(
            Evidence("v", "grid", {"grid": g}, _hidden_of(g)))):
        if h.claim.__class__.__name__ != "UniversalClaim":
            continue
        key = (h.claim.rule, tuple(h.claim.params))
        if key in excluded:
            continue
        rank = (-h.confidence, i)
        if bestkey is None or rank < bestkey:
            best, bestkey = key, rank
    return best


def _rule_error(key, g):
    rule, params = key
    return verify(Proposition(Kind.UNIVERSAL, rule, (rule, params)), Truth(grid=g))


def _refs_r0(task: Task, g):
    return [resolve(r, g) for r in task.refs]


# ------------------------------------------------------------------- operations
def apply_op(op: Op, task: Task, st: State, track_destroys: bool = True) -> State:
    """Every operation costs exactly 1. None of them may consult ground truth."""
    st.ops += 1
    if op is Op.STOP:
        st.stopped = True
        return st

    if op is Op.EXPLORE:
        st.view = min(st.view + 1, len(task.views) - 1)
        return st

    if op is Op.INDUCE:
        if task.kind == "ref":
            if SHAPE_EQ in st.excluded:
                st.hyp = None
            else:
                rs = _refs_r0(task, task.views[st.view])
                ok = all(r.status is Resolution.UNIQUE for r in rs)
                st.hyp = SHAPE_EQ if ok else None
            st.tested_view, st.refuted = None, False
            st.induced_view = st.view
            if st.hyp is None:
                st.induce_empty += 1
            return st
        key = _pick(task.views[st.view], st.excluded)
        if key is None:
            st.induce_empty += 1
        st.hyp, st.tested_view, st.refuted = key, None, False
        st.induced_view = st.view
        return st

    if op is Op.FALSIFY:
        if st.hyp is None:
            return st
        if task.kind == "ref":
            rs = _refs_r0(task, task.views[st.view])
            if not all(r.status is Resolution.UNIQUE for r in rs):
                st.tested_view, st.refuted = st.view, True
                st.excluded.add(st.hyp)
                return st
            a, b = (r.comp["shape"] for r in rs)
            st.tested_view, st.refuted = st.view, (a != b)
            if st.refuted:
                st.excluded.add(st.hyp)
            return st
        e = _rule_error(st.hyp, task.views[st.view])
        st.tested_view = st.view
        st.refuted = bool(e is not None and e > 0.0)
        if st.refuted:
            st.excluded.add(st.hyp)
        return st

    if op is Op.TRACK:
        st.tracked = True
        if task.kind == "ref":
            r = resolve_through(task.refs[-1], list(task.frames))
            if r.status is not Resolution.UNIQUE:
                st.refs_dead = True
                st.hyp, st.tested_view, st.refuted = None, None, False
            return st
        # no referent question here: re-representation only discards the model
        if track_destroys:
            st.hyp, st.tested_view, st.refuted = None, None, False
        return st

    raise AssertionError(op)


# ------------------------------------------------------------------- the judge
def solved(task: Task, st: State) -> bool:
    """GROUND TRUTH. Used only for scoring, never reachable from a policy."""
    if task.kind == "noise":
        return st.hyp is None
    if task.kind == "ref":
        return (not st.refs_dead) if task.ref_alive else st.refs_dead
    if st.hyp is None:
        return False
    return _rule_error(st.hyp, task.full) == 0.0


# ---------------------------------------------------------------- the labellers
def _no_hypothesis_label(task: Task, st: State) -> Label:
    """Holding nothing is two different situations, and collapsing them was a real
    bug caught by tracing `rule_loose` before any measurement was taken: an engine
    that has not yet been asked here wants INDUCE, while one that has already
    abstained on this view wants different evidence, not a second identical attempt.
    Both facts are observable, so ORACLE and HARD share this and no arm is
    advantaged by the repair."""
    if st.induced_view != st.view:
        return Label.NO_MODEL
    if st.view < len(task.views) - 1:
        return Label.INSUFFICIENT_EVIDENCE
    return Label.NO_MODEL


def oracle_label(task: Task, st: State) -> Label:
    """Tier 0: the true epistemic state, computed from ground truth AND the agent's
    actual runtime state. Not a per-task answer key -- there is no field in `Task`
    that names an operation."""
    if solved(task, st):
        return Label.SOLVED
    if task.kind == "ref":
        if not task.ref_alive:
            return Label.REPRESENTATION_ERROR
        return Label.NO_MODEL
    if task.kind == "noise":
        return Label.UNTESTABLE
    if st.hyp is None:
        return _no_hypothesis_label(task, st)
    # the held hypothesis is false. Is it refutable from what is already visible?
    e = _rule_error(st.hyp, task.views[st.view])
    if e is not None and e > 0.0:
        return Label.MODEL_ERROR
    return Label.INSUFFICIENT_EVIDENCE


def hard_label(task: Task, st: State) -> Label:
    """Tier 1: everything the agent can actually observe, and nothing else."""
    if task.kind == "ref":
        if st.refs_dead:
            return Label.SOLVED
        if st.hyp is None:
            return Label.NO_MODEL
        if st.tested_view != st.view:
            return Label.UNTESTED
        return Label.MODEL_ERROR if st.refuted else Label.SOLVED
    at_max = st.view == len(task.views) - 1
    if st.hyp is None:
        # repeated abstention at the fullest view IS observable evidence that there
        # is nothing here. Withholding it would gift the noise family to ORACLE.
        if at_max and st.induce_empty >= 2:
            return Label.UNTESTABLE
        return _no_hypothesis_label(task, st)
    if st.tested_view != st.view:
        return Label.UNTESTED
    if st.refuted:
        return Label.MODEL_ERROR
    # M2 section D: unrefuted is not the same as true, and nothing visible separates
    # them. This is the forced error, and it is the point of the experiment.
    return Label.SOLVED


def _evidence_left(task: Task, st: State) -> bool:
    return st.view < len(task.views) - 1 or not st.tracked


def abstain_label(task: Task, st: State) -> Label:
    """Tier 1 evidence, plus permission to answer UNKNOWN where the discriminating
    evidence is PROVABLY absent. The two triggers are the two measured blind spots,
    not a list tuned to this benchmark -- and note they fire on `rule_intact` and
    `ref_intact` exactly as hard as on their broken twins, because the twins are
    observationally identical. Abstention cannot be aimed."""
    h = hard_label(task, st)
    if h is Label.SOLVED and not (task.kind == "ref" and st.refs_dead):
        if _evidence_left(task, st):
            return Label.UNKNOWN
    if h is Label.MODEL_ERROR and task.kind == "ref" and not st.tracked:
        return Label.UNKNOWN
    return h


_CONFUSABLE = (Label.MODEL_ERROR, Label.REPRESENTATION_ERROR)


def corrupt(label: Label, rng: random.Random, c: float) -> Label:
    """C_classifier, ABOVE the observability floor -- and confusion-matrix-aware, not
    iid over the label set: it swaps only the pair M3 identified as non-identifiable.
    c = 0 must reproduce HARD exactly, which the runner asserts."""
    if c > 0.0 and label in _CONFUSABLE and rng.random() < c:
        return _CONFUSABLE[1 - _CONFUSABLE.index(label)]
    return label


# ------------------------------------------------------------------ the policies
ROUTE = {
    Label.NO_MODEL: Op.INDUCE,
    Label.UNTESTED: Op.FALSIFY,
    Label.MODEL_ERROR: Op.INDUCE,          # with the refuted key excluded
    Label.INSUFFICIENT_EVIDENCE: Op.EXPLORE,
    Label.REPRESENTATION_ERROR: Op.TRACK,
    Label.SOLVED: Op.STOP,
    Label.UNTESTABLE: Op.STOP,
}


def typed_policy(label: Label, task: Task, st: State) -> Op:
    """Deterministic table lookup. No scalar objective, per the frozen spec."""
    if label is Label.UNKNOWN:
        # uniform "acquire the evidence you do not have", in a fixed precedence.
        # NOT typed by family -- that would smuggle the oracle answer in.
        if st.view < len(task.views) - 1:
            return Op.EXPLORE
        if not st.tracked:
            return Op.TRACK
        return Op.STOP
    return ROUTE[label]


# ------------------------------------------------------------------- the episode
@dataclass
class Episode:
    tid: str
    family: str
    arm: str
    solved: bool
    ops: int
    ops_to_solve: int          # budget spent before first reaching a solved state
    irreversible: bool         # held the answer, then lost it
    trace: List[Tuple[str, str]] = field(default_factory=list)


def run_episode(task: Task, arm: str, rng: random.Random, *, c: float = 0.0,
                fixed: Optional[Tuple[Op, ...]] = None,
                track_destroys: bool = True, budget: int = BUDGET,
                forced: Tuple[Op, ...] = ()) -> Episode:
    """`forced` replays a prefix of operations before the policy takes over. It is
    how the counterfactual replay substitutes one decision and lets the controller
    carry on, which is the difference between a regret measurement and a story."""
    st = State()
    if task.kind == "ref":
        st.hyp = SHAPE_EQ                 # frozen on frames[0], as M3 froze it
    ever, ops_to_solve = solved(task, st), (0 if solved(task, st) else -1)
    trace = []
    while st.ops < budget and not st.stopped:
        if st.ops < len(forced):
            op, lab = forced[st.ops], "forced"
            trace.append((lab, op.value))
            apply_op(op, task, st, track_destroys=track_destroys)
            if solved(task, st):
                ever = True
                if ops_to_solve < 0:
                    ops_to_solve = st.ops
            continue
        if arm == "random":
            op = rng.choice(REAL_OPS + (Op.STOP,))
            lab = "-"
        elif arm == "fixed":
            op = fixed[st.ops % len(fixed)]
            lab = "-"
        else:
            if arm == "oracle":
                label = oracle_label(task, st)
            elif arm == "hard":
                label = hard_label(task, st)
            elif arm == "corrupted":
                label = corrupt(hard_label(task, st), rng, c)
            elif arm == "abstain":
                label = abstain_label(task, st)
            elif arm == "abstain_corrupted":
                label = corrupt(abstain_label(task, st), rng, c)
            else:
                raise AssertionError(arm)
            op = typed_policy(label, task, st)
            lab = label.value
        trace.append((lab, op.value))
        apply_op(op, task, st, track_destroys=track_destroys)
        if solved(task, st):
            ever = True
            if ops_to_solve < 0:
                ops_to_solve = st.ops
    fin = solved(task, st)
    return Episode(task.tid, task.family, arm, fin, st.ops,
                   ops_to_solve if ops_to_solve >= 0 else budget,
                   irreversible=(ever and not fin), trace=trace)


FIXED_ORDERS = tuple(permutations(REAL_OPS))
