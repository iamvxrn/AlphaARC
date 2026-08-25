"""The submission is a BUILT file -- prove the build is faithful and importable.

Kaggle splices only `agent/my_agent.py`, so the bundled single file, not the
package, is what actually scores. A bundle that drifts from the package (or that
imports at all) is a silent zero, so pin it here: same functions, same numbers.
Run directly (no pytest).
"""

import ast
import os
import subprocess
import sys
import tempfile
import types

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.join(HERE, "..", "..")
sys.path.insert(0, os.path.join(HERE, ".."))

from alphaarc import mdl, residual, correspondence  # noqa: E402

SYM_BOX = [
    [2, 2, 2, 2, 2],
    [2, 4, 3, 4, 2],
    [2, 3, 4, 3, 2],
    [2, 4, 3, 4, 2],
    [2, 2, 2, 2, 2],
]


def _twin_boxes():
    """Two identical boxes, one cell knocked out -> the Correspondence fixture."""
    bg = 9
    g = [[bg] * 14 for _ in range(7)]
    for top, left in ((1, 1), (1, 8)):
        for r in range(5):
            for c in range(5):
                g[top + r][left + c] = SYM_BOX[r][c]
    g[3][10] = bg
    return g, bg


def _build_bundle() -> str:
    out = os.path.join(tempfile.mkdtemp(), "my_agent.py")
    subprocess.run([sys.executable, os.path.join(REPO, "python", "bench", "bundle.py"),
                    "--out", out], check=True, capture_output=True)
    return out


def _import_bundle(path: str):
    """Import the bundle with the engine API stubbed, as Kaggle would import it."""
    arcengine = types.ModuleType("arcengine")
    for name in ("FrameData", "GameAction", "GameState"):
        setattr(arcengine, name, type(name, (), {}))
    agents = types.ModuleType("agents")
    agents_agent = types.ModuleType("agents.agent")
    agents_agent.Agent = type("Agent", (), {})
    agents.agent = agents_agent
    sys.modules.update({"arcengine": arcengine, "agents": agents, "agents.agent": agents_agent})

    mod = types.ModuleType("bundled_agent")
    mod.__file__ = path
    with open(path) as fh:
        exec(compile(fh.read(), path, "exec"), mod.__dict__)
    return mod


def test_bundle_is_self_contained():
    """No package-relative import may survive: on Kaggle there is no package."""
    src = open(_build_bundle()).read()
    for node in ast.walk(ast.parse(src)):
        if isinstance(node, ast.ImportFrom):
            assert node.level == 0, f"relative import survived the bundle: {ast.dump(node)}"
            assert not (node.module or "").startswith("alphaarc"), \
                f"package import survived the bundle: {node.module}"


def test_bundle_defines_the_kaggle_contract():
    mod = _import_bundle(_build_bundle())
    assert hasattr(mod, "MyAgent"), "bundle must define MyAgent"
    for method in ("is_done", "choose_action"):
        assert callable(getattr(mod.MyAgent, method, None)), f"MyAgent.{method} missing"


def test_bundle_numbers_match_the_package():
    """Same inputs, same bits -- the bundled core is the tested core."""
    mod = _import_bundle(_build_bundle())
    g, bg = _twin_boxes()

    assert mod.correspondence_savings(g, bg) == correspondence.correspondence_savings(g, bg)
    assert mod.best_primitive(g, bg)[1] == mdl.best_primitive(g, bg)[1]
    assert [(t.x, t.y) for t in mod.residual_targets(g, bg, 8)] == \
           [(t.x, t.y) for t in residual.residual_targets(g, bg, 8)]


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
