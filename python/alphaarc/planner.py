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
from .clock import ClockTracker
from .perception import background_color, control_signature
from .policy import Policy
from .residual import object_targets, residual_targets


def rigid_move(before: Grid, after: Grid, bg: int) -> Optional[Tuple[int, int, int]]:
    """(colour, drow, dcol) if exactly one colour's cells moved as a rigid body.

    This is how an avatar announces itself. Everything else on the board -- a
    budget strip ticking, a counter, scenery being repainted -- fails the test,
    because a rigid translation preserves the cell COUNT and every offset.
    """
    moved = None
    colours = {v for row in before for v in row} | {v for row in after for v in row}
    for colour in colours:
        if colour == bg:
            continue
        a = sorted((r, c) for r in range(len(before))
                   for c in range(len(before[r])) if before[r][c] == colour)
        b = sorted((r, c) for r in range(len(after))
                   for c in range(len(after[r])) if after[r][c] == colour)
        if not a or len(a) != len(b) or a == b:
            continue
        dr, dc = b[0][0] - a[0][0], b[0][1] - a[0][1]
        if all((r + dr, c + dc) == q for (r, c), q in zip(a, b)):
            if moved is not None:
                return None          # two bodies moved: not one avatar
            moved = (colour, dr, dc)
    return moved


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
        # What each key does to the avatar: key -> (drow, dcol). Learned from the
        # board, never assumed. Compression rewards a tidier board, not ARRIVAL, so
        # a movement game needs a goal and a route -- decoding ls20 showed its keys
        # and its credit are both fine and it still scores zero.
        self.moves: Dict[str, Tuple[int, int]] = {}
        self.avatar: Optional[int] = None
        self._bg: Optional[int] = None
        self.tries: Dict[str, int] = {}

        # A plan is "press control S, k more times" -- NOT a list of frozen
        # coordinates. On a board that redraws itself the coordinates go stale
        # between one press and the next, so the position is re-derived from the
        # current board every step.
        self._presses_left = 0
        self._run_token: Optional[str] = None
        self._prev_grid: Optional[Grid] = None
        self._prev_levels: Optional[List[float]] = None
        self.clock = ClockTracker()

    # ------------------------------------------------------------- bookkeeping

    @staticmethod
    def _tok(x: int, y: int) -> str:
        return "%d,%d" % (x, y)

    @staticmethod
    def _signature(grid: Grid, bg: int, x: int, y: int) -> str:
        """A name for a control that survives a redraw -- see perception."""
        return control_signature(grid, bg, x, y)

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
        # The budget strip ticks on every action, so an exact-equality test can
        # never be true and `inert` was never once set in vc33, r11l, ft09, g50t,
        # s5i5 or su15. See alphaarc/clock.py.
        if self.clock.clock_only(self._prev_grid, grid):
            self.inert.add(self._run_token)
            self._presses_left = 0      # a dead control: stop spending on it
            return
        prof = self.profiles.setdefault(self._run_token, [self._prev_levels or levels])
        prof.append(levels)
        if self._run_token.startswith("k") and self._bg is not None:
            got = rigid_move(self._prev_grid, grid, self._bg)
            if got:
                self.avatar = got[0]
                self.moves[self._run_token] = (got[1], got[2])

    # ------------------------------------------------------------- deciding

    def _candidates(self, grid: Grid, bg: int):
        pts = list(residual_targets(grid, bg, self.max_candidates))
        seen = {(p.x, p.y) for p in pts}
        for p in object_targets(grid, bg, self.max_candidates):
            if (p.x, p.y) not in seen:
                seen.add((p.x, p.y))
                pts.append(p)
        return pts

    def _route(self, grid: Grid, bg: int) -> Optional[str]:
        """The key that most reduces the avatar's distance to the anomaly, or None.

        Greedy on Manhattan distance rather than a full path: a step that closes
        the gap is worth taking now, and the model is re-checked every step, so a
        wall simply shows up as a key that stopped working.
        """
        if self.avatar is None or len(self.moves) < 2:
            return None
        cells = [(r, c) for r in range(len(grid))
                 for c in range(len(grid[r])) if grid[r][c] == self.avatar]
        if not cells:
            return None
        ar = sum(r for r, _ in cells) / len(cells)
        ac = sum(c for _, c in cells) / len(cells)
        targets = self._candidates(grid, bg)
        targets = [t for t in targets if grid[t.y][t.x] != self.avatar]
        if not targets:
            return None
        t = min(targets, key=lambda p: abs(p.y - ar) + abs(p.x - ac))
        here = abs(t.y - ar) + abs(t.x - ac)
        best, best_d = None, here
        for key, (dr, dc) in self.moves.items():
            d = abs(t.y - (ar + dr)) + abs(t.x - (ac + dc))
            if d < best_d:
                best, best_d = key, d
        return best

    @staticmethod
    def _as_choice(target):
        return ("key", target[1]) if target[0] == "key" else ("click", target)

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
        got = self.choose(grid, bg, keys=())
        return got[1] if got and got[0] == "click" else None

    def choose(self, grid: Grid, bg: Optional[int] = None, keys: Tuple[int, ...] = (),
               clickable: bool = True):
        """Pick a control: ("click", (x, y)) or ("key", action_value), or None.

        A key IS a control -- something you can press N times and watch -- so it
        goes through the same profile machinery as a click. Eight of the seventeen
        train games are keyboard-driven and none of them has ever scored, because
        neither this nor the one-step policy could express a key at all and the
        adapter fell through to a random one. Keys also make the BEST controls for
        this machinery: their names never move when the board redraws.
        """
        if bg is None:
            bg = background_color(grid)
        self._bg = bg
        levels = self._levels(grid, bg)
        self._observe(grid, levels)

        # Once the avatar and at least two directions are known, steering it at the
        # anomaly beats pressing whichever key most tidied the board: the residual
        # already says WHERE compression fails, and moving there is what a movement
        # game asks for.
        route = self._route(grid, bg) if (keys and self.moves) else None
        if route is not None:
            self._presses_left = 0
            self._run_token = route
            self._prev_grid = [row[:] for row in grid]
            self._prev_levels = levels
            return ("key", int(route[1:]))

        pts = self._candidates(grid, bg) if clickable else []
        sigs = {(p.x, p.y): self._signature(grid, bg, p.x, p.y) for p in pts}
        for k in keys:
            sigs[("key", k)] = "k%d" % k

        # Inside a run: find where that control is NOW and keep going. The whole
        # point is not to abandon a control because its first press looked bad --
        # and on a rescaling board, not to lose it because it moved.
        if self._presses_left > 0 and self._run_token is not None:
            here = [t for t, sig in sigs.items() if sig == self._run_token]
            if here:
                self._presses_left -= 1
                self._prev_grid = [row[:] for row in grid]
                self._prev_levels = levels
                self.tries[self._run_token] = self.tries.get(self._run_token, 0) + 1
                return self._as_choice(here[0])
            self._presses_left = 0      # the control is gone from the board
        targets = list(sigs)
        if not targets:
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
        unprobed = [t for t in targets
                    if sigs[t] not in self.profiles and sigs[t] not in self.inert]
        escalate = [t for t in targets
                    if sigs[t] in self.profiles
                    and sigs[t] not in self.inert
                    and sigs[t] not in self.ran
                    and self._best_run(sigs[t])[0] <= 0]
        if unprobed and self.rng.random() > self.explore:
            target = unprobed[0]                       # learn: one press, cheaply
            presses = 1
        elif escalate:
            target = escalate[0]                       # it moved but did not pay:
            presses = self.run_length                  # maybe the payoff is deeper
            self.ran.add(sigs[target])
        else:
            scored = [(self._best_run(sigs[t])[0], t) for t in targets
                      if sigs[t] not in self.inert]
            scored = [x for x in scored if x[0] > 0]
            if scored:
                scored.sort(key=lambda x: -x[0])        # exploit the best known run
                target = scored[0][1]
                presses = max(1, self._best_run(sigs[target])[1])
            else:
                target = self.rng.choice(targets)       # nothing known pays: sample
                presses = self.run_length

        self._run_token = sigs[target]
        self.profiles.pop(self._run_token, None)        # re-anchor from HERE
        self._presses_left = presses - 1
        self._prev_grid = [row[:] for row in grid]
        self._prev_levels = levels
        self.tries[self._run_token] = self.tries.get(self._run_token, 0) + 1
        return self._as_choice(target)


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
                 rng: Optional[random.Random] = None, policy=None, planner=None,
                 clock_dead_run: bool = False):
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
        self.clock_dead_run = clock_dead_run
        self.since_level = 0
        self.dead_run = 0
        self.switched = False
        self._prev_grid: Optional[Grid] = None
        self.clock = ClockTracker()

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
        got = self.choose(grid, bg, keys=())
        return got[1] if got and got[0] == "click" else None

    def choose(self, grid: Grid, bg: Optional[int] = None, keys: Tuple[int, ...] = (),
               clickable: bool = True):
        """Same contract as RunPlanner.choose.

        One asymmetry decides the routing: the one-step policy cannot express a
        key at all. So when the board offers keys and no click, the planner takes
        it regardless of the clock -- otherwise the adapter falls through to a
        RANDOM key, which is what eight of seventeen games have been getting.
        """
        if keys and not clickable:
            return self.planner.choose(grid, bg, keys, clickable=False)
        self.since_level += 1
        if self._prev_grid is not None:
            # DELIBERATELY still whole-board equality, which in a game with a move
            # budget is never true -- so this counter stays zero and the hand-over
            # below fires only on its 25-action clock.
            #
            # Fixing it is a one-line change and it is NOT a bug fix: `dead_streak`
            # was set to 5 while this counter could not increment, so the threshold
            # was never calibrated against a counter that works. Turning it on is
            # NEW BEHAVIOUR and belongs to its own measurement. Measured with it on,
            # paired over 4 seeds: tn36 +0.750 with all seeds agreeing, r11l -0.432,
            # aggregate +0.10 +/- 0.19 -- not measurable, because it hands vc33 and
            # r11l to the planner at action ~13 instead of 25, and this class's own
            # docstring records the planner scoring 0.000 on vc33 against the
            # one-step policy's 4.343.
            #
            # So: `clock_dead_run=True` enables it, and `dead_streak` has to be
            # re-derived when it does. See alphaarc/clock.py.
            dead = (self.clock.clock_only(self._prev_grid, grid) if self.clock_dead_run
                    else grid == self._prev_grid)
            self.dead_run = self.dead_run + 1 if dead else 0
        self._prev_grid = [row[:] for row in grid]
        if not self.switched and (self.since_level > self.switch_after
                                  or self.dead_run >= self.dead_streak):
            self.switched = True
        if self.switched:
            return self.planner.choose(grid, bg, keys, clickable=clickable)
        xy = self.policy.choose_click(grid, bg)
        return ("click", xy) if xy is not None else None
