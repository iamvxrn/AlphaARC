"""Value a RUN of presses, not a single click.

Five attempts at improving one-step credit returned zero or worse (see
python/bench/README.md, "Tried and rejected"), while the games we understand best
fail for one measured reason: their reward is not visible one step ahead.

  - vc33's scalar pays off on the THIRD press and the first press looks harmful:
    Reflect goes 308 -> 200 -> 256 -> 364.
  - ft09's tiles are involutive, so signed one-step deltas cancel to ~0.
  - tn36's controls are period-2 toggles of amplitude 3.
  - r11l's controls fire once and are then inert.

All four are served by the same idea: learn, for each control, what the board
becomes after n presses, then commit to the n that pays. That is the smallest
honest form of "learn a model, plan against it" -- the plans are runs of one
control rather than arbitrary programs, which is exactly what these games need.

Budget arithmetic, from `reference_arc_scoring_function`: a level's score is
(baseline/actions)^2 weighted by the level INDEX, so spending actions on level 1
(weight 1 of ~28) to learn the mechanic and then executing levels 2+ near
baseline is what the metric actually rewards.
"""

from __future__ import annotations

import random
from typing import Dict, List, Optional, Tuple

from .mdl import PRIMITIVES, Grid, _components
from .perception import background_color
from .policy import Policy
from .residual import object_targets, residual_targets


def _gain(now: List[float], later: List[float]) -> float:
    """Signed biggest-mover between two level vectors.

    Per primitive, never collapsed to the winner: two boards can both read 16
    because Reflect dominates each while Translate reads 16 against 8, and that
    difference is the whole effect of the control.
    """
    return max((b - a for a, b in zip(now, later)), key=abs, default=0.0)


class RunPlanner:
    def __init__(
        self,
        run_length: int = 3,
        max_candidates: int = 8,
        explore: float = 0.1,
        rng: Optional[random.Random] = None,
    ):
        # vc33's payoff is at press 3, so a run shorter than that cannot see it.
        self.run_length = run_length
        self.max_candidates = max_candidates
        self.explore = explore
        self.rng = rng or random.Random()

        # token -> level vectors observed after 0, 1, 2, ... presses of that
        # control, measured from the board we started the run on.
        self.profiles: Dict[str, List[List[float]]] = {}
        self.inert: set[str] = set()
        self.ran: set[str] = set()       # controls already given a full run
        self.tries: Dict[str, int] = {}

        # A plan is "press control S, k more times" -- NOT a list of frozen
        # coordinates. On a board that redraws itself the coordinates go stale
        # between one press and the next, so the position is re-derived from the
        # current board every step.
        self._presses_left = 0
        self._run_token: Optional[str] = None
        self._prev_grid: Optional[Grid] = None
        self._prev_levels: Optional[List[float]] = None

    # ------------------------------------------------------------- bookkeeping

    @staticmethod
    def _tok(x: int, y: int) -> str:
        return "%d,%d" % (x, y)

    @staticmethod
    def _signature(grid: Grid, bg: int, x: int, y: int) -> str:
        """A name for a control that survives the board being redrawn.

        vc33 rescales its whole scene on every press, so a control's pixel
        coordinates land somewhere new each time: instrumented over 30 clicks the
        planner hit 30 DISTINCT positions, never repeated one, and therefore never
        exploited anything. Naming a control by the OBJECT under it -- its colour,
        its size bucket, and where it sits in the frame in eighths -- keeps the
        profile attached to the same button across a redraw.

        Coarse on purpose: exact size and position are what the redraw changes.
        """
        h, w = len(grid), len(grid[0]) if grid else 1
        colour = grid[y][x] if 0 <= y < h and 0 <= x < w else bg
        size, total = 0, 0
        for col, cells in _components(grid, bg):
            total += len(cells)
            if not size and col == colour and any(r == y and c == x for r, c in cells):
                size = len(cells)
        # Size as a FRACTION of the scene, not in pixels: a rescale multiplies every
        # area by the same factor, so absolute size is exactly what the redraw
        # changes (measured: the same box reads s4 at one scale and s6 at double).
        share = size / total if total else 0.0
        bucket = 0
        while bucket < 6 and share < 0.5 ** (bucket + 1):
            bucket += 1
        return "c%d/f%d/%d,%d" % (colour, bucket, (y * 8) // max(1, h), (x * 8) // max(1, w))

    @staticmethod
    def _levels(grid: Grid, bg: int) -> List[float]:
        return [float(p.savings(grid, bg)) for p in PRIMITIVES]

    def board_replaced(self) -> None:
        """A reset, a game over, or a level transition: the board we were reasoning
        about is gone. What we learned about controls survives -- the mechanic does
        not change with the scenery -- but the run in flight is abandoned, and the
        profiles are re-anchored because they are measured relative to a board."""
        self._presses_left = 0
        self._run_token = None
        self._prev_grid = None
        self._prev_levels = None
        self.profiles.clear()
        self.ran.clear()

    reset_episode = board_replaced

    def _observe(self, grid: Grid, levels: List[float]) -> None:
        """Extend the profile of the control whose run we are inside."""
        if self._run_token is None or self._prev_grid is None:
            return
        if grid == self._prev_grid:
            self.inert.add(self._run_token)
            self._presses_left = 0      # a dead control: stop spending on it
            return
        prof = self.profiles.setdefault(self._run_token, [self._prev_levels or levels])
        prof.append(levels)

    # ------------------------------------------------------------- deciding

    def _candidates(self, grid: Grid, bg: int):
        pts = list(residual_targets(grid, bg, self.max_candidates))
        seen = {(p.x, p.y) for p in pts}
        for p in object_targets(grid, bg, self.max_candidates):
            if (p.x, p.y) not in seen:
                seen.add((p.x, p.y))
                pts.append(p)
        return pts

    def _best_run(self, token: str) -> Tuple[float, int]:
        """(value, presses) of the best point along this control's known profile."""
        prof = self.profiles.get(token)
        if not prof or len(prof) < 2:
            return 0.0, 0
        start = prof[0]
        best_v, best_n = 0.0, 0
        for n in range(1, len(prof)):
            v = _gain(start, prof[n])
            if v > best_v:
                best_v, best_n = v, n
        return best_v, best_n

    def choose_click(self, grid: Grid, bg: Optional[int] = None) -> Optional[Tuple[int, int]]:
        if bg is None:
            bg = background_color(grid)
        levels = self._levels(grid, bg)
        self._observe(grid, levels)

        pts = self._candidates(grid, bg)
        sigs = {(p.x, p.y): self._signature(grid, bg, p.x, p.y) for p in pts}

        # Inside a run: find where that control is NOW and keep going. The whole
        # point is not to abandon a control because its first press looked bad --
        # and on a rescaling board, not to lose it because it moved.
        if self._presses_left > 0 and self._run_token is not None:
            here = [xy for xy, sig in sigs.items() if sig == self._run_token]
            if here:
                xy = here[0]
                self._presses_left -= 1
                self._prev_grid = [row[:] for row in grid]
                self._prev_levels = levels
                self.tries[self._run_token] = self.tries.get(self._run_token, 0) + 1
                return xy
            self._presses_left = 0      # the control is gone from the board
        if not pts:
            self._prev_grid = [row[:] for row in grid]
            self._run_token = None
            self._presses_left = 0
            return None

        # Budget policy, and it is the whole difference between this helping and
        # hurting. Probing every candidate with a full run costs run_length actions
        # each; vc33's level 1 has a baseline of SEVEN, so eight controls x three
        # presses spends the level's entire score before exploiting anything
        # (measured: vc33 4.34 -> 0.05, while lp85 -- whose control genuinely needs
        # repetition -- went 0.08 -> 0.78). So: try a control ONCE first, and pay
        # for a run only where a single press moved the board without paying off.
        # That is where a valley can hide; a control that does nothing, or that
        # already pays, needs no run.
        unprobed = [p for p in pts
                    if sigs[(p.x, p.y)] not in self.profiles
                    and sigs[(p.x, p.y)] not in self.inert]
        escalate = [p for p in pts
                    if sigs[(p.x, p.y)] in self.profiles
                    and sigs[(p.x, p.y)] not in self.inert
                    and sigs[(p.x, p.y)] not in self.ran
                    and self._best_run(sigs[(p.x, p.y)])[0] <= 0]
        if unprobed and self.rng.random() > self.explore:
            target = unprobed[0]                       # learn: one press, cheaply
            presses = 1
        elif escalate:
            target = escalate[0]                       # it moved but did not pay:
            presses = self.run_length                  # maybe the payoff is deeper
            self.ran.add(sigs[(target.x, target.y)])
        else:
            scored = [(self._best_run(sigs[(p.x, p.y)])[0], p) for p in pts
                      if sigs[(p.x, p.y)] not in self.inert]
            scored = [s for s in scored if s[0] > 0]
            if scored:
                scored.sort(key=lambda s: -s[0])        # exploit the best known run
                target = scored[0][1]
                presses = max(1, self._best_run(sigs[(target.x, target.y)])[1])
            else:
                target = self.rng.choice(pts)           # nothing known pays: sample
                presses = self.run_length

        self._run_token = sigs[(target.x, target.y)]
        self.profiles.pop(self._run_token, None)        # re-anchor from HERE
        self._presses_left = presses - 1
        self._prev_grid = [row[:] for row in grid]
        self._prev_levels = levels
        self.tries[self._run_token] = self.tries.get(self._run_token, 0) + 1
        return (target.x, target.y)


class HybridPolicy:
    """One-step credit first; run-planning only once the cheap agent has stalled.

    Measured on the quick split, the two agents win DIFFERENT games:

        vc33 (level-1 baseline 7)   one-step 4.343   planner 0.000
        lp85 (baseline 17)          one-step 0.082   planner 2.778
        tn36 (baseline 32)          one-step 0.000   planner 0.338

    and the split is not mysterious. Systematic exploration costs a press per
    candidate; on a level whose baseline is seven actions that is the whole level,
    and the score punishes wasted actions quadratically. So spend the first actions
    the cheap way, and only pay for runs once it is clear the cheap way is not
    working -- by which point the level is already long and the extra presses cost
    proportionally little.

    The threshold is in ACTIONS, not baselines: the agent is never told what the
    baseline is.
    """

    def __init__(self, switch_after: int = 25, dead_streak: int = 5,
                 rng: Optional[random.Random] = None, policy=None, planner=None):
        rng = rng or random.Random()
        # The engine adapter falls back to a random simple action when no click is
        # available -- keyboard-only games take that path on every step -- and it
        # reaches for `.rng`. Every policy the adapter can hold must expose one.
        self.rng = rng
        self.switch_after = switch_after
        self.policy = policy if policy is not None else Policy(rng=rng)
        self.planner = planner if planner is not None else RunPlanner(rng=rng)
        # Hand over early if the cheap agent is visibly getting nowhere. lp85 puts
        # a row of six small blocks at the top -- a progress indicator, not a
        # control -- and "smallest object first" ranks those AHEAD of the 4x4
        # palette swatches that actually do something. The planner marks them inert
        # in a few actions and moves on; the one-step policy spends its whole
        # opening on them. Dead clicks are the evidence, so switch on those rather
        # than only on a clock.
        self.dead_streak = dead_streak
        self.since_level = 0
        self.dead_run = 0
        self.switched = False
        self._prev_grid: Optional[Grid] = None

    @property
    def active(self):
        return self.planner if self.switched else self.policy

    def board_replaced(self) -> None:
        """A new level is a fresh chance for the cheap agent: reset the clock and
        hand control back, because the next level's opening may well be short."""
        self.since_level = 0
        self.dead_run = 0
        self.switched = False
        self._prev_grid = None
        self.policy.board_replaced()
        self.planner.board_replaced()

    reset_episode = board_replaced

    def choose_click(self, grid: Grid, bg: Optional[int] = None) -> Optional[Tuple[int, int]]:
        self.since_level += 1
        if self._prev_grid is not None:
            self.dead_run = self.dead_run + 1 if grid == self._prev_grid else 0
        self._prev_grid = [row[:] for row in grid]
        if not self.switched and (self.since_level > self.switch_after
                                  or self.dead_run >= self.dead_streak):
            self.switched = True
        return self.active.choose_click(grid, bg)
