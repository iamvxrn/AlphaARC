"""MyAgent -- the adapter between the arc-agi engine and our decision core.

This is the ONLY module that touches the engine's API surface; everything above
it is engine-agnostic, unit-tested logic. The bundler emits this module last, so
in the built single-file submission `MyAgent` sits on top of the inlined core.

The API details here are VERIFIED against the installed `arc-agi` package (they
were wrong in the first, un-run port): `FrameData.frame` is a LIST of 2D grids
and the live board is the last non-empty layer; ACTION6 carries the click as
`set_data({"x": col, "y": row})`; state is a `GameState` enum.
"""

from __future__ import annotations

import os
import random
import time
import zlib
from typing import Any, List, Optional

from arcengine import FrameData, GameAction, GameState
from agents.agent import Agent

from .mdl import Grid
from .perception import background_color
from .policy import Policy


def _stable_seed(base: int, game_id: str) -> int:
    """Seed that survives across processes.

    `hash()` on a str is salted per interpreter (PYTHONHASHSEED), so deriving a
    seed from it makes every process a different experiment -- which silently
    defeated the bench's --seed. crc32 is stable.
    """
    return base + zlib.crc32(game_id.encode()) % 1_000_000


class MyAgent(Agent):
    # Exploration needs ~15+ actions to lock onto the solving click (live wins
    # cluster around action 12-95), so give it room. ~12s/game at the engine's
    # ~20fps -- comfortably inside the 6h total budget.
    MAX_ACTIONS = 250

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        # Exploration is random, so a run is only reproducible if the seed is.
        # ARC_AGENT_SEED pins it for the bench; on Kaggle it is unset and we seed
        # from the clock, which is what we want there (each play an independent
        # sample, and the scorecard keeps a game's BEST run).
        pinned = os.environ.get("ARC_AGENT_SEED")
        base = int(pinned) if pinned else int(time.time() * 1_000_000)
        self._policy = Policy(rng=random.Random(_stable_seed(base, self.game_id)))
        self._levels_done = 0

    @property
    def name(self) -> str:
        return f"{super().name}.{self.MAX_ACTIONS}"

    def is_done(self, frames: List[FrameData], latest_frame: FrameData) -> bool:
        return latest_frame.state is GameState.WIN

    @staticmethod
    def _current_grid(fr: FrameData) -> Optional[Grid]:
        layers = getattr(fr, "frame", None)
        if not layers:
            return None
        for layer in reversed(layers):  # last non-empty layer is the live board
            if layer and layer[0]:
                return layer
        return None

    def choose_action(self, frames: List[FrameData], latest_frame: FrameData) -> GameAction:
        if latest_frame.state in (GameState.NOT_PLAYED, GameState.GAME_OVER):
            self._levels_done = 0
            self._policy.board_replaced()
            return GameAction.RESET

        # Completing a level swaps the whole board. Tell the policy, or it credits
        # its last click with the compression delta ACROSS that seam -- a large
        # arbitrary number, plus a junk entry in the transition model, at exactly
        # the point where the score starts paying (level i is weighted by i).
        done = int(getattr(latest_frame, "levels_completed", 0) or 0)
        if done > self._levels_done:
            self._levels_done = done
            self._policy.board_replaced()

        grid = self._current_grid(latest_frame)
        avail = set(latest_frame.available_actions or [])

        if grid is not None and GameAction.ACTION6.value in avail:
            bg = background_color(grid)
            xy = self._policy.choose_click(grid, bg)
            if xy is not None:
                x, y = xy
                action = GameAction.ACTION6
                action.set_data({"x": int(x), "y": int(y)})
                action.reasoning = {"why": "compression-residual click"}
                return action

        # No click available or no candidate -> a random available simple action,
        # else RESET. (Pure-click games always offer ACTION6.)
        simple = [a for a in GameAction
                  if a is not GameAction.RESET and not a.is_complex() and a.value in avail]
        if simple:
            action = self._policy.rng.choice(simple)
            action.reasoning = "fallback simple action"
            return action
        return GameAction.RESET
