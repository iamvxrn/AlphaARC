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

import json
import os
import random
from collections import deque
from typing import Dict, List, Optional, Tuple

from .mdl import PRIMITIVES, Grid, _components
from .clock import ClockTracker
from .perception import background_color, control_signature
from .policy import Policy
from .residual import object_targets, residual_targets


def _normalise(cells) -> frozenset:
    """A shape, independent of where it sits."""
    r0 = min(r for r, _ in cells)
    c0 = min(c for _, c in cells)
    return frozenset((r - r0, c - c0) for r, c in cells)


def _rigid_by_colour(before: Grid, after: Grid, bg: int) -> Optional[Tuple[int, int, int]]:
    """Exactly one COLOUR's cells moved as one rigid body. The original rule, kept
    first and unchanged, because it is what already finds ls20's and sp80's
    avatars and any replacement risks finding a different body than the one the
    route was built around."""
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


def _rigid_by_component(before: Grid, after: Grid, bg: int):
    """(colour, shape, drow, dcol) for the largest SHAPE that translated.

    Reached only where the colour test above is silent, and that is not a rare
    corner: probing every key of every movement game from a fresh board, g50t's
    avatar is a 24-cell colour-9 shape moving (+6,0) on ACTION2 and (0,+6) on
    ACTION4 -- perfectly rigid -- while colour 9 covers 119 cells, most of them
    scenery, so the colour is not a translation and the detector reported NOTHING.
    All five of g50t's directions were invisible.

    Ambiguity is still refused: two shapes of the SAME size moving in different
    directions is m0r0's mirrored pair of 25-cell markers, (0,-5) and (0,+5) on
    one key, where "which of them am I steering" has no answer yet.
    """
    a, b = _shapes(before, bg), _shapes(after, bg)
    moved = []
    for key in set(a) & set(b):
        pa, pb = sorted(a[key]), sorted(b[key])
        if pa == pb or len(pa) != len(pb):
            continue
        for off in {(q[0] - p[0], q[1] - p[1]) for p, q in zip(pa, pb)}:
            moved.append((len(key[1]), key[0], off, key[1]))
    if not moved:
        return None
    size = max(m[0] for m in moved)
    top = {m[2] for m in moved if m[0] == size}
    if len(top) != 1:
        return None
    colour, shape = next((m[1], m[3]) for m in moved if m[0] == size)
    (dr, dc), = top
    return (colour, shape, dr, dc)


def _shapes(grid: Grid, bg: int):
    """(colour, shape normalised to its own top-left) -> the anchors it occurs at."""
    out: Dict[Tuple[int, frozenset], List[Tuple[int, int]]] = {}
    for col, cells in _components(grid, bg):
        out.setdefault((col, _normalise(cells)), []).append(
            (min(r for r, _ in cells), min(c for _, c in cells)))
    return out


def rigid_body(before: Grid, after: Grid, bg: int):
    """(colour, shape or None, drow, dcol) for the body that translated.

    Two levels, and the order is the whole point. The colour test runs first and
    its answer stands, so every game where the avatar was already found behaves
    EXACTLY as before -- shape None means "locate it by colour", the old code
    path. Replacing that rule instead of extending it was measured: naming the
    largest moving shape re-identified ls20's avatar from colour 12 to colour 9
    and cost it its level (0.385 -> 0.000 at seed 2). The per-component rule is
    reached only where the colour rule found nothing at all.
    """
    got = _rigid_by_colour(before, after, bg)
    if got is not None:
        return (got[0], None, got[1], got[2])
    return _rigid_by_component(before, after, bg)


def rigid_move(before: Grid, after: Grid, bg: int) -> Optional[Tuple[int, int, int]]:
    """The avatar's colour and offset -- `rigid_body` without the shape."""
    got = rigid_body(before, after, bg)
    return None if got is None else (got[0], got[2], got[3])


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
        self._trace_path = os.environ.get("ARC_TRACE") or None
        self._writeoffs = 0
        self.ran: set[str] = set()       # controls already given a full run
        # What each key does to the avatar: key -> (drow, dcol). Learned from the
        # board, never assumed. Compression rewards a tidier board, not ARRIVAL, so
        # a movement game needs a goal and a route -- decoding ls20 showed its keys
        # and its credit are both fine and it still scores zero.
        self.moves: Dict[str, Tuple[int, int]] = {}
        self.avatar: Optional[int] = None
        # ...and the avatar's SHAPE, when the colour alone does not locate it:
        # g50t's avatar is a 24-cell shape inside 119 cells of colour 9, so a
        # centroid over the colour sits in the scenery. None = locate by colour.
        self.avatar_shape: Optional[frozenset] = None
        # The destination the avatar is currently steering at, held across steps
        # instead of re-derived, and the ones already reached or given up on.
        self._dest: Optional[Tuple[int, int]] = None
        self._dest_best: Optional[float] = None
        self._dest_stale = 0
        self.crossed_off: set = set()
        # A wall is an EDGE THAT DOES NOT EXIST, learned by trying: a routed key
        # that leaves the avatar where it was marks (anchor, key) impassable.
        self.blocked: set = set()
        self._routed_from: Optional[Tuple[Tuple[int, int], str]] = None
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

    def board_replaced(self, reason: str = "board") -> None:
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
        # A destination is a place on THIS board; the next level's board is a
        # different place. What survives is the key->offset map, which is the
        # mechanic.
        self._dest = None
        self._dest_best = None
        self._dest_stale = 0
        self.crossed_off.clear()
        self.blocked.clear()          # walls are a property of THIS board
        self._routed_from = None

    reset_episode = board_replaced

    def _observe(self, grid: Grid, levels: List[float]) -> None:
        """Extend the profile of the control whose run we are inside."""
        if self._run_token is None or self._prev_grid is None:
            return
        # The budget strip ticks on every action, so an exact-equality test can
        # never be true and `inert` was never once set in vc33, r11l, ft09, g50t,
        # s5i5 or su15. See alphaarc/clock.py.
        _co = self.clock.clock_only(self._prev_grid, grid)
        if self._trace_path:
            # Is POSITION the variable that would make "this control does nothing"
            # a conditional fact instead of a flat one? Log where the avatar was
            # when the control fired, and what the control did. Nothing reads this.
            try:
                cells = self._avatar_cells(self._prev_grid, self._bg) if self._bg is not None else []
            except Exception:
                cells = []
            pos = (sum(c[0] for c in cells)//len(cells), sum(c[1] for c in cells)//len(cells)) if cells else None
            # `_avatar_cells` returns EVERY cell of the avatar's colour when the
            # shape is not uniquely placed -- on m0r0 that blends two mirrored
            # bodies into one meaningless centroid, and on g50t it mixes the legend
            # and the budget strip in. So also record the bodies SEPARATELY: the
            # configuration, not an average of it.
            bodies = None
            try:
                if self.avatar is not None and self._bg is not None:
                    cs = [cl for col, cl in _components(self._prev_grid, self._bg)
                          if col == self.avatar]
                    if cs and len(cs) <= 8:
                        bodies = sorted((sum(c[0] for c in cl)//len(cl),
                                         sum(c[1] for c in cl)//len(cl)) for cl in cs)
            except Exception:
                bodies = None
            _rows, _cols = self.clock.strip()
            _st = hash(tuple(tuple(v for c, v in enumerate(row) if c not in _cols)
                             for r, row in enumerate(grid) if r not in _rows))
            with open(self._trace_path, "a") as fh:
                fh.write(json.dumps({"event": "fired", "tok": self._run_token,
                                     "pos": pos, "bodies": bodies, "n": len(cells),
                                     "shape": self.avatar_shape is not None,
                                     "s": _st, "nothing": bool(_co)}) + "\n")
        if _co:
            if self._trace_path and self._run_token not in self.inert:
                self._writeoffs += 1
                with open(self._trace_path, "a") as fh:
                    fh.write(json.dumps({"event": "inert", "tok": self._run_token,
                                         "n": self._writeoffs}) + "\n")
            self.inert.add(self._run_token)
            self._presses_left = 0      # a dead control: stop spending on it
            return
        prof = self.profiles.setdefault(self._run_token, [self._prev_levels or levels])
        prof.append(levels)
        if self._run_token.startswith("k") and self._bg is not None:
            got = rigid_body(self._prev_grid, grid, self._bg)
            if got:
                self.avatar, self.avatar_shape = got[0], got[1]
                self.moves[self._run_token] = (got[2], got[3])

    # ------------------------------------------------------------- deciding

    def _candidates(self, grid: Grid, bg: int):
        pts = list(residual_targets(grid, bg, self.max_candidates))
        seen = {(p.x, p.y) for p in pts}
        for p in object_targets(grid, bg, self.max_candidates):
            if (p.x, p.y) not in seen:
                seen.add((p.x, p.y))
                pts.append(p)
        return pts

    def _plan(self, pos, dest, tol: float) -> Optional[str]:
        """First key of a shortest path from `pos` to within `tol` of `dest`.

        Greedy distance cannot go around a wall -- that is the whole difficulty of
        a maze, and ls20 is a maze -- because getting past one means moving AWAY
        from the target for a step. Breadth-first over the lattice the avatar's own
        offsets define, with the edges it has found to be impassable removed, will
        do that; untried edges are assumed open, which is what makes it explore.
        """
        if abs(pos[0] - dest[0]) + abs(pos[1] - dest[1]) <= tol:
            return None
        seen = {pos}
        q = deque([(pos, None)])
        while q:
            cur, first = q.popleft()
            for key, (dr, dc) in sorted(self.moves.items()):
                if (cur, key) in self.blocked:
                    continue
                nxt = (cur[0] + dr, cur[1] + dc)
                if nxt in seen or not (0 <= nxt[0] < self._h and 0 <= nxt[1] < self._w):
                    continue
                step = first if first is not None else key
                if abs(nxt[0] - dest[0]) + abs(nxt[1] - dest[1]) <= tol:
                    return step
                seen.add(nxt)
                q.append((nxt, step))
        return None

    def _avatar_cells(self, grid: Grid, bg: int) -> List[Tuple[int, int]]:
        """Where the avatar is NOW. By shape when one is known and occurs exactly
        once; otherwise every cell of its colour -- the original behaviour, which
        is right for a one-body scene and merely imprecise, whereas guessing
        between two identical bodies is not."""
        if self.avatar is None:
            return []
        if self.avatar_shape is not None:
            hits = [cells for col, cells in _components(grid, bg)
                    if col == self.avatar and _normalise(cells) == self.avatar_shape]
            if len(hits) == 1:
                return hits[0]
        return [(r, c) for r in range(len(grid))
                for c in range(len(grid[r])) if grid[r][c] == self.avatar]

    def _route(self, grid: Grid, bg: int) -> Optional[str]:
        """Steer the avatar toward a COMMITTED destination.

        The previous version re-chose "the nearest candidate" on every single step,
        which is not a goal at all -- it is a gradient that reverses the moment the
        avatar moves, because the nearest anomaly is usually whatever fragment the
        avatar is standing next to. Instrumented on ls20, seed 1: 249 route
        decisions, the avatar's position changed on 223 of them, and it spent the
        whole run oscillating between about six positions -- (40,36) -> (36,36) ->
        (40,36) -- with the key choice alternating k3 and k4, which are opposite
        directions. It moves constantly and arrives nowhere.

        So a destination is chosen once and held until it is REACHED or PROVES
        UNREACHABLE, and then crossed off so the next one is tried. "Proves
        unreachable" needs no tuned constant: once every direction we know has been
        spent without the distance improving, greedy steering has nothing left to
        offer for that destination.

        This does not solve a maze -- greedy distance cannot go around a wall, and
        ls20 is a maze. What it removes is the jitter that prevented the agent from
        ever committing to anything long enough to find that out.
        """
        if self.avatar is None or len(self.moves) < 2:
            return None
        cells = self._avatar_cells(grid, bg)
        if not cells:
            return None
        self._h, self._w = len(grid), len(grid[0]) if grid else 1
        ar = sum(r for r, _ in cells) / len(cells)
        ac = sum(c for _, c in cells) / len(cells)
        anchor = (min(r for r, _ in cells), min(c for _, c in cells))
        # Did the last routed key move us? If not, that edge is a wall.
        if self._routed_from is not None and self._routed_from[0] == anchor:
            self.blocked.add(self._routed_from)
        self._routed_from = None
        occupied = set(cells)
        targets = [t for t in self._candidates(grid, bg)
                   if (t.y, t.x) not in occupied and (t.y, t.x) not in self.crossed_off]

        if self._dest is not None and self._dest in self.crossed_off:
            self._dest = None
        if self._dest is None:
            if not targets:
                self.crossed_off.clear()      # nothing left: start the sweep again
                return None
            t = min(targets, key=lambda p: abs(p.y - ar) + abs(p.x - ac))
            self._dest = (t.y, t.x)
            self._dest_best = None
            self._dest_stale = 0

        ty, tx = self._dest
        here = abs(ty - ar) + abs(tx - ac)
        # Arrived: the avatar covers the destination, or is within one step of it.
        step = min((abs(dr) + abs(dc) for dr, dc in self.moves.values()), default=1)
        if here <= step / 2:
            self.crossed_off.add(self._dest)
            self._dest = None
            return None

        # Search, not gradient. The destination is expressed in ANCHOR space so
        # the lattice and the goal test speak the same coordinates.
        goal = (ty - (ar - anchor[0]), tx - (ac - anchor[1]))
        best = self._plan(anchor, goal, step / 2)
        if best is None:
            # Nothing reachable through what we have learned: cross it off rather
            # than stand still, and let the sweep move to the next candidate.
            self.crossed_off.add(self._dest)
            self._dest = None
            return None
        self._routed_from = (anchor, best)

        if self._dest_best is None or here < self._dest_best:
            self._dest_best, self._dest_stale = here, 0
        else:
            self._dest_stale += 1
            # Every known direction spent without getting closer: greedy steering
            # has nothing more to offer here, so stop paying for it.
            if self._dest_stale >= len(self.moves):
                self.crossed_off.add(self._dest)
                self._dest = None
                return None
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

    def __init__(self, switch_after: int = 25, dead_streak: int = 20,
                 rng: Optional[random.Random] = None, policy=None, planner=None,
                 clock_dead_run: bool = True, dead_decay: float = 0.75):
        rng = rng or random.Random()
        # The engine adapter falls back to a random simple action when no click is
        # available -- keyboard-only games take that path on every step -- and it
        # reaches for `.rng`. Every policy the adapter can hold must expose one.
        self.rng = rng
        self.switch_after = switch_after
        # NOTE the shared `rng`: the one-step policy and the planner draw from the
        # SAME stream, so a change that only alters a constant keeps the whole run
        # comparable seed for seed. Handing the policy its own Random would move
        # every subsequent draw and show up as a difference that is purely the RNG.
        self.policy = policy if policy is not None else Policy(rng=rng, dead_decay=dead_decay)
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

    def board_replaced(self, reason: str = "board") -> None:
        """A new level is a fresh chance for the cheap agent: reset the clock and
        hand control back, because the next level's opening may well be short."""
        self.since_level = 0
        self.dead_run = 0
        self.switched = False
        self._prev_grid = None
        self.policy.board_replaced(reason)
        self.planner.board_replaced(reason)

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
            # "Was that nothing but the move budget ticking?" -- see clock.py.
            # Whole-board equality is never true in a game with a budget strip, so
            # before that fix this counter was permanently zero and the hand-over
            # below could only ever fire on its 25-action clock.
            #
            # `dead_streak` was 5, chosen while the counter could not increment, so
            # it had never been calibrated. Swept once it worked, paired over 4
            # seeds against HEAD (each on top of the clock fix):
            #
            #     5   +0.177 +/- 0.200   r11l -0.482, ALL seeds agree -- it hurts
            #    10   +0.138 +/- 0.199
            #    20   +0.089 +/- 0.052   no game harmed
            #
            # At 5 the hybrid gives vc33 and r11l to the planner around action 13,
            # and this class's own docstring records the planner scoring 0.000 on
            # vc33 against the one-step policy's 4.343. 20 against switch_after=25
            # means the streak only pre-empts the clock when a level is nearly all
            # dead clicks, which is exactly when the cheap policy has nothing left
            # to offer. Confirmed at 8 seeds: +0.0666, sem 0.0302 -- it clears
            # 2*sem, though one seed of eight moves -0.0034, so seeds.py withholds
            # the REAL verdict and this is reported as a small positive, not a win.
            #
            # ARC_DEAD_STREAK / ARC_CLOCK_DEADRUN=0 re-run the sweep from the bench.
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
