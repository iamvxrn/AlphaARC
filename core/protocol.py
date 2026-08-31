"""The protocol, and nothing else. No reasoning lives here.

The bet of this branch is NOT that three engines share an internal language. It is
that each can turn what it believes into something testable, and that the error can
be scored at the level the claim was made:

    claim  ->  testable prediction  ->  error

So `PredictionSet` is deliberately heterogeneous. Forcing an MDL primitive, a
correspondence and a state-transition edge to all emit a next-State would kill the
idea with an artificial API long before the idea is tested.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Dict, List, Optional, Protocol

EvidenceId = str
Grid = List[List[int]]


class Level(Enum):
    """The level a claim speaks at. Error is scored HERE, not translated."""
    CELL = "cell"                 # concrete cell values
    RELATION = "relation"         # a relation between two structures
    TRANSITION = "transition"     # (state, action) -> state'
    ACTION_EFFECT = "effect"      # a scalar/vector consequence of an action
    CONSTRAINT = "constraint"     # something that must hold, not a value


@dataclass(frozen=True)
class Evidence:
    """One observed thing. Engines read these; they never read the environment."""
    id: EvidenceId
    kind: str                     # grid | grid_pair | transition
    payload: Dict[str, Any]


@dataclass(frozen=True)
class Prediction:
    level: Level
    target: str                   # what it is about, free-form but stable
    value: Any                    # meaning depends on level; the scorer knows
    note: str = ""


PredictionSet = List[Prediction]


class Claim(Protocol):
    """What an engine believes. It must be able to make itself testable."""
    def predict(self, ctx: Evidence) -> PredictionSet: ...
    def describe(self) -> str: ...


@dataclass
class Hypothesis:
    source: str                   # which engine
    claim: Claim
    scope: str                    # where the claim is meant to hold
    confidence: float
    complexity: float             # description length, in whatever unit the engine uses
    evidence_ids: List[EvidenceId] = field(default_factory=list)

    def describe(self) -> str:
        return f"[{self.source}] {self.claim.describe()}"


class Engine(Protocol):
    name: str
    def hypotheses(self, ev: Evidence) -> List[Hypothesis]: ...


# ---------------------------------------------------------------- error scoring

def _cell_error(pred_value, obs: Grid) -> Optional[float]:
    """pred_value: dict {(r,c): v} or a full grid. Fraction of predicted cells wrong."""
    if obs is None:
        return None
    if isinstance(pred_value, dict):
        items = pred_value.items()
    else:
        items = (((r, c), pred_value[r][c])
                 for r in range(len(pred_value)) for c in range(len(pred_value[r])))
    n = wrong = 0
    for (r, c), v in items:
        if not (0 <= r < len(obs) and 0 <= c < len(obs[r])):
            continue
        n += 1
        if obs[r][c] != v:
            wrong += 1
    return None if n == 0 else wrong / n


def score(pred: Prediction, observation: Any) -> Optional[float]:
    """Error in [0,1], or None when this observation cannot test this prediction.

    None is a first-class answer: an engine that could not be tested here is not
    the same as an engine that was wrong, and conflating them is how ensembles get
    mistaken for reasoning.
    """
    if pred.level is Level.CELL:
        return _cell_error(pred.value, observation)
    if pred.level is Level.RELATION:
        if observation is None:
            return None
        return 0.0 if pred.value == observation else 1.0
    if pred.level is Level.TRANSITION:
        if observation is None:
            return None
        return 0.0 if pred.value == observation else 1.0
    if pred.level is Level.ACTION_EFFECT:
        if observation is None:
            return None
        try:
            a, b = float(pred.value), float(observation)
        except (TypeError, ValueError):
            return 0.0 if pred.value == observation else 1.0
        denom = max(abs(a), abs(b), 1.0)
        return min(1.0, abs(a - b) / denom)
    if pred.level is Level.CONSTRAINT:
        return None if observation is None else (0.0 if bool(observation) else 1.0)
    return None
