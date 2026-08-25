"""Model-free compression-drive policy -- the essential decision loop from
cmd/alphaarc-play, ported lean.

The affordance MODEL can't learn the big/absolute/colour-colliding effects these
games have, so instead of predicting we REINFORCE on the observed compression
delta: each clicked token's value is an EMA of how much it actually moved the
drive (best_primitive_delta -- the signed biggest-mover, this session's fix).
Candidates come from residual_cells' clustered targets (click where compression
fails). This is env-agnostic pure logic; my_agent.py adapts it to arc-agi.
"""

from __future__ import annotations

import random
from typing import Dict, List, Optional, Tuple

from .mdl import PRIMITIVES, Grid
from .perception import background_color
from .residual import object_targets, residual_targets


class Policy:
    def __init__(
        self,
        drive_gain_weight: float = 0.02,
        residual_bonus: float = 0.15,
        explore: float = 0.2,
        max_candidates: int = 8,
        rng: Optional[random.Random] = None,
    ):
        self.w = drive_gain_weight
        self.residual_bonus = residual_bonus
        self.explore = explore
        self.max_candidates = max_candidates
        self.rng = rng or random.Random()
        self.drive_gain: Dict[str, float] = {}   # EMA of observed compression delta per token
        # A one-step world model: what the compression levels BECAME, last time this
        # token was used from a board with these levels. Keyed by (token, levels).
        self.succ: Dict[Tuple[str, Tuple[float, ...]], List[float]] = {}
        self.tries: Dict[str, int] = {}          # optimism for the under-tried
        self.dead: Dict[str, float] = {}         # decaying "did nothing" count (inhibition of return)
        self._prev_grid: Optional[Grid] = None
        self._prev_levels: Optional[List[float]] = None
        self._last_token: Optional[str] = None

    def board_replaced(self) -> None:
        """The board was swapped out: a RESET, a GAME_OVER, or a LEVEL TRANSITION.

        What the policy LEARNED about clicks (drive_gain, succ, tries) survives --
        the mechanic does not change when the board does. Only the transition trace
        is dropped, so the next click is not credited with a compression delta
        measured across the seam between two unrelated boards. Without this the
        first click of a new level receives a large arbitrary credit and writes a
        junk entry into the transition model, precisely at the moment that matters
        most: level 2 onward is where the score actually is.
        """
        self._prev_grid = None
        self._prev_levels = None
        self._last_token = None

    # A level change and a reset are the same event from the policy's point of
    # view: the board it was reasoning about is gone. Kept as an alias so the
    # engine adapter can name whichever it actually observed.
    reset_episode = board_replaced

    @staticmethod
    def _tok(x: int, y: int) -> str:
        return "%d,%d" % (x, y)

    @staticmethod
    def _levels(grid: Grid, bg: int) -> List[float]:
        """Savings of EVERY primitive, not just the winner.

        Collapsing to the argmax hides exactly the differences that matter: two
        boards can both score 16 because Reflect dominates both, while Translate
        reads 16 vs 8 -- the real effect of the control. This is the same
        argmax-of-argmax trap `best_primitive_delta` was written to avoid.
        """
        return [float(p.savings(grid, bg)) for p in PRIMITIVES]

    def _credit_last(self, grid: Grid, bg: int, level: List[float]) -> None:
        """Credit the previous click, and remember WHAT IT DID from where it was.

        The EMA of signed ΔL is kept -- it is fine for a control that pushes the
        board one way -- but it is blind to an INVOLUTIVE control: a toggle scores
        +24 then -24 on the same token, so its average decays to ~0 and the control
        never accumulates value even though one of its two states is genuinely
        better. That is measured, on ft09, where every control is a toggle.

        Averaging cannot fix this, because the same token is right from one state
        and wrong from the other. What distinguishes them is the STATE, so record
        the transition: (token, levels here) -> levels there. One step of a world
        model, which is also the direction the whole architecture is going.
        """
        if self._prev_grid is None or self._last_token is None:
            return
        # The signed biggest-mover delta, derived from the level vectors we already
        # have instead of re-evaluating every primitive on both grids. This is
        # exactly what mdl.best_primitive_delta computes, so deriving it here HALVES
        # the per-step primitive work rather than adding to it.
        prev = self._prev_levels or [0.0] * len(level)
        d = max((n - o for n, o in zip(level, prev)), key=abs, default=0.0)
        self.drive_gain[self._last_token] = 0.6 * self.drive_gain.get(self._last_token, 0.0) + 0.4 * d
        self.succ[(self._last_token, tuple(prev))] = list(level)
        if grid == self._prev_grid:  # the click changed nothing -> taboo it a while
            self.dead[self._last_token] = self.dead.get(self._last_token, 0.0) + 1.0
        # decay all taboos slightly (inhibition of return fades)
        for k in list(self.dead):
            self.dead[k] *= 0.75
            if self.dead[k] < 0.05:
                del self.dead[k]

    def choose_click(self, grid: Grid, bg: Optional[int] = None) -> Optional[Tuple[int, int]]:
        """Return the (x=col, y=row) to click, or None if no candidate exists."""
        if bg is None:
            bg = background_color(grid)
        level = self._levels(grid, bg)
        self._credit_last(grid, bg, level)
        self._prev_levels = level

        # Candidates = residual anomaly centroids UNION object centroids, deduped
        # (residual first). Object centroids surface small interactive controls
        # (buttons) the residual misses -- without them the ported agent scored 0
        # on vc33; with them it solves L1. driveGain then learns which pay off.
        pts = list(residual_targets(grid, bg, self.max_candidates))
        seen = {(p.x, p.y) for p in pts}
        for p in object_targets(grid, bg, self.max_candidates):
            if (p.x, p.y) not in seen:
                seen.add((p.x, p.y))
                pts.append(p)
        if not pts:
            self._prev_grid = [row[:] for row in grid]
            self._last_token = None
            return None

        best_tok, best_xy, best_v = None, None, -1e18
        for i, p in enumerate(pts):
            tok = self._tok(p.x, p.y)
            # How much better than NOW is the best state this control has reached?
            # Zero once we are already there, so a toggle is pursued to its good
            # state and then left alone instead of being flipped back and forth.
            # What this control did LAST TIME FROM HERE. Signed on purpose: a
            # toggle is worth clicking from its bad state and worth avoiding from
            # its good one, and only a state-keyed prediction can tell them apart.
            nxt = self.succ.get((tok, tuple(level)))
            predicted = 0.0 if nxt is None else max(
                (n - c for n, c in zip(nxt, level)), key=abs, default=0.0
            )
            v = (
                self.w * self.drive_gain.get(tok, 0.0)
                + self.w * 4.0 * predicted               # state-keyed prediction (survives involution)
                + self.residual_bonus / (i + 1)          # larger clusters (earlier) rank higher
                + 0.05 / (self.tries.get(tok, 0) + 1)    # optimism for the under-tried
                - self.dead.get(tok, 0.0)                # inhibition of return
            )
            if v > best_v:
                best_v, best_tok, best_xy = v, tok, (p.x, p.y)

        # epsilon-exploration over the candidate set
        if self.explore > 0 and self.rng.random() < self.explore:
            p = self.rng.choice(pts)
            best_tok, best_xy = self._tok(p.x, p.y), (p.x, p.y)

        self.tries[best_tok] = self.tries.get(best_tok, 0) + 1
        self._prev_grid = [row[:] for row in grid]
        self._last_token = best_tok
        return best_xy
