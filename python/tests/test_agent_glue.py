"""The engine adapter is the code that actually plays, so test IT, not a mock of it.

Only this module touches the arc-agi API, so the API is what gets stubbed here;
everything below it is the real policy. Run directly (no pytest).
"""

import os
import sys
import types

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, ".."))


def _install_engine_stubs():
    """Minimal stand-ins for the arc-agi surface agent_glue imports."""
    class GameState:
        NOT_PLAYED, GAME_OVER, WIN, NOT_FINISHED = "NOT_PLAYED", "GAME_OVER", "WIN", "NOT_FINISHED"

    class _Action:
        def __init__(self, name, value, complex_=False):
            self.name, self.value, self._complex = name, value, complex_
            self.data, self.reasoning = None, None

        def set_data(self, d):
            self.data = d

        def is_complex(self):
            return self._complex

    class GameActionMeta(type):
        def __iter__(cls):
            return iter((cls.RESET, cls.ACTION1, cls.ACTION6))

    class GameAction(metaclass=GameActionMeta):
        RESET = _Action("RESET", 0)
        ACTION1 = _Action("ACTION1", 1)
        ACTION6 = _Action("ACTION6", 6, complex_=True)

    arcengine = types.ModuleType("arcengine")
    arcengine.FrameData = type("FrameData", (), {})
    arcengine.GameAction = GameAction
    arcengine.GameState = GameState

    agents = types.ModuleType("agents")
    agents_agent = types.ModuleType("agents.agent")

    class Agent:
        def __init__(self, **kw):
            self.game_id = kw.get("game_id", "test")

        @property
        def name(self):
            return "stub"

    agents_agent.Agent = Agent
    agents.agent = agents_agent
    sys.modules.update({"arcengine": arcengine, "agents": agents, "agents.agent": agents_agent})
    return GameState, GameAction


GameState, GameAction = _install_engine_stubs()
from alphaarc.agent_glue import MyAgent  # noqa: E402

SYM_BOX = [
    [2, 2, 2, 2, 2],
    [2, 4, 3, 4, 2],
    [2, 3, 4, 3, 2],
    [2, 4, 3, 4, 2],
    [2, 2, 2, 2, 2],
]


class Frame:
    def __init__(self, grid, state=GameState.NOT_FINISHED, levels_completed=0):
        self.frame = [grid]
        self.state = state
        self.levels_completed = levels_completed
        self.available_actions = [GameAction.ACTION6.value]


def _board(seed_col):
    g = [[9] * 14 for _ in range(7)]
    for top, left in ((1, 1), (1, 8)):
        for r in range(5):
            for c in range(5):
                g[top + r][left + c] = SYM_BOX[r][c]
    g[3][seed_col] = 9          # a residual, in a different place per board
    return g


def _agent():
    os.environ["ARC_AGENT_SEED"] = "1"
    return MyAgent(card_id="t", game_id="t", agent_name="t", ROOT_URL="http://x",
                   record=False, arc_env=None, tags=[])


def test_completing_a_level_drops_the_stale_trace():
    """The seam between two levels must not be credited as if it were a click."""
    a = _agent()
    a.choose_action([], Frame(_board(10), levels_completed=0))
    assert a._policy._last_token is not None, "fixture: a click should have left a trace"

    a.choose_action([], Frame(_board(3), levels_completed=1))
    # The level-up is seen BEFORE the new click, so no transition spans the seam.
    assert a._policy.succ == {}, f"a transition was learned across the seam: {a._policy.succ}"


def test_a_level_is_only_counted_once():
    """levels_completed stays high on later frames; that must not keep resetting."""
    a = _agent()
    a.choose_action([], Frame(_board(10), levels_completed=1))
    a.choose_action([], Frame(_board(10), levels_completed=1))
    a.choose_action([], Frame(_board(10), levels_completed=1))
    assert a._policy.succ, "the trace was dropped on every frame, so nothing is ever learned"


def test_game_over_resets_the_level_counter():
    """After a reset the engine counts from zero again; so must we, or the next
    level-up looks like no change and the seam is credited."""
    a = _agent()
    a.choose_action([], Frame(_board(10), levels_completed=2))
    assert a._levels_done == 2
    a.choose_action([], Frame(_board(10), state=GameState.GAME_OVER, levels_completed=2))
    assert a._levels_done == 0


def test_the_agent_still_returns_a_click():
    a = _agent()
    action = a.choose_action([], Frame(_board(10)))
    assert action is GameAction.ACTION6 and action.data is not None, action



def test_every_policy_the_adapter_can_hold_exposes_an_rng():
    """The adapter's fallback path -- no click available, pick a random simple
    action -- reaches for `.rng`. Keyboard-only games take that path on EVERY step,
    and the quick split is all click games, so only the full split caught this:
    a train run died on g50t with AttributeError after five games."""
    import random as _random
    from alphaarc.planner import HybridPolicy, RunPlanner
    from alphaarc.policy import Policy
    for cls in (Policy, RunPlanner, HybridPolicy):
        agent = cls(rng=_random.Random(0))
        assert hasattr(agent, "rng"), f"{cls.__name__} has no .rng"
        assert agent.rng.choice([1, 2, 3]) in (1, 2, 3)


if __name__ == "__main__":
    failures = 0
    for name, fn in sorted(globals().items()):
        if name.startswith("test_") and callable(fn):
            try:
                fn()
                print(f"ok   {name}")
            except AssertionError as e:
                failures += 1
                print(f"FAIL {name}: {e}")
    print("PASS" if not failures else f"{failures} FAILURE(S)")
    sys.exit(1 if failures else 0)
