"""Three minimal experiences of DIFFERENT kinds. One example each is enough to
test the interface; more would only test the engines, which is not this milestone.
"""
from __future__ import annotations
import json, sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
from protocol import Evidence

HERE = Path(__file__).resolve().parent


def static_case() -> Evidence:
    """A grid with an exact mirror symmetry. ARC-1 shaped: no actions at all."""
    g = [[0] * 16 for _ in range(16)]
    for r in range(3, 8):
        for c in range(2, 6):
            if (r + c) % 3:
                g[r][c] = 4
    for r in range(16):
        for c in range(8):
            g[r][15 - c] = g[r][c]
    return Evidence("static-mirror", "grid", {"grid": g, "truth": "board is mirror-symmetric"})


def relational_case() -> Evidence:
    """Two identical shapes in different places: a correspondence, not a symmetry."""
    g = [[0] * 20 for _ in range(20)]
    shape = [(0, 0), (0, 1), (1, 0), (2, 0), (2, 1)]
    for r, c in shape:
        g[2 + r][2 + c] = 7
        g[12 + r][13 + c] = 7
    return Evidence("relational-copy", "grid",
                    {"grid": g, "truth": "the two colour-7 objects are identical"})


def interactive_case() -> Evidence:
    """A REAL vc33 level-3 transition, captured from the engine."""
    d = json.loads((HERE / "fixtures" / "vc33_l3_transition.json").read_text())
    return Evidence("vc33-l3-transfer", "transition",
                    {"before": d["before"], "after": d["after"],
                     "action": d["action"], "truth": d["note"]})


CASES = [static_case(), relational_case(), interactive_case()]
