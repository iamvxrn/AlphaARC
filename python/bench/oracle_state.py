"""Save-state for offline oracles. RESEARCH ONLY -- never in a submitted agent.

Why it exists: a systematic probe of a mid-game board needs to return to that board
thousands of times. `GameAction.RESET` cannot do it -- after a GAME_OVER the engine
runs `level_reset()` and keeps the score, but during normal play the same RESET was
measured to drop `levels_completed` to 0 and hand back level 1. So an oracle either
replays the whole approach for every probe, or it snapshots.

WHAT THIS TOUCHES, STATED PLAINLY:

    env._game            the simulator object behind LocalEnvironmentWrapper
    game._levels         its private level list
    game._current_level_index, game.set_level, game._state

`Level.clone()` is a normal engine method, but reaching it through `_game._levels`
is NOT the observation-action interface an agent is given. This is strictly more
access than a player has.

RULES:
  * research and diagnosis only, never imported by `alphaarc/`;
  * FORBIDDEN in the Kaggle submission agent, which sees frames and sends actions;
  * it does not read the puzzle's logic or its solution -- only saves and restores
    simulator state -- but it is still privileged access, so nothing measured with
    it may be quoted as something the agent could have done.
"""
from __future__ import annotations


class LevelSnapshot:
    """Save and restore the current level of a local arc-agi environment."""

    def __init__(self, env):
        self.game = env._game
        self.index = self.game._current_level_index
        self.level = self.game._levels[self.index].clone()

    def restore(self) -> None:
        from arcengine import GameState
        self.game._levels[self.index] = self.level.clone()
        self.game.set_level(self.index)
        self.game._state = GameState.NOT_FINISHED
