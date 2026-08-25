"""Control agent: uniformly random legal actions. The floor of the benchmark.

Without this number "our agent scores 0.0097" is uninterpretable -- some ARC-AGI-3
levels fall to button-mashing, and any mechanism we build has to beat the mash,
not zero. Self-contained (same contract as a submission file) so the bench can
run it with --agent.
"""

from __future__ import annotations

import os
import random
import zlib
from typing import Any, List

from arcengine import FrameData, GameAction, GameState
from agents.agent import Agent


class MyAgent(Agent):
    MAX_ACTIONS = 250

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        base = int(os.environ.get("ARC_AGENT_SEED", "0"))
        self.rng = random.Random(base + zlib.crc32(self.game_id.encode()) % 1_000_000)

    def is_done(self, frames: List[FrameData], latest_frame: FrameData) -> bool:
        return latest_frame.state is GameState.WIN

    @staticmethod
    def _dims(fr: FrameData) -> tuple[int, int]:
        layers = getattr(fr, "frame", None) or []
        for layer in reversed(layers):
            if layer and layer[0]:
                return len(layer), len(layer[0])
        return 64, 64

    def choose_action(self, frames: List[FrameData], latest_frame: FrameData) -> GameAction:
        if latest_frame.state in (GameState.NOT_PLAYED, GameState.GAME_OVER):
            return GameAction.RESET

        avail = set(latest_frame.available_actions or [])
        choices = [a for a in GameAction if a is not GameAction.RESET and a.value in avail]
        if not choices:
            return GameAction.RESET

        action = self.rng.choice(choices)
        if action.is_complex():
            h, w = self._dims(latest_frame)
            action.set_data({"x": self.rng.randrange(w), "y": self.rng.randrange(h)})
        action.reasoning = "random control"
        return action
