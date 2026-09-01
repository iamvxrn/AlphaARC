"""Cases in POSITIVE and CORRUPTED form. Without the corrupted twin, a suite only
shows that engines are right where they were built to be right.

The runner masks cells before handing evidence over, so a prediction is always
about something the engine did not see.
"""
from __future__ import annotations
import copy, json, random, sys
from pathlib import Path
from typing import List, Tuple
sys.path.insert(0, str(Path(__file__).resolve().parent))
from protocol import HIDDEN, Evidence, Grid, Truth

HERE = Path(__file__).resolve().parent


def _mirror_grid() -> Grid:
    g = [[0] * 16 for _ in range(16)]
    for r in range(3, 12):
        for c in range(2, 8):
            g[r][c] = (r * 3 + c * 5) % 4
    for r in range(16):
        for c in range(8):
            g[r][15 - c] = g[r][c]
    return g


def _copies_grid() -> Grid:
    g = [[0] * 20 for _ in range(20)]
    shape = [(0, 0), (0, 1), (1, 0), (2, 0), (2, 1)]
    for r, c in shape:
        g[2 + r][2 + c] = 7
        g[12 + r][13 + c] = 7
    return g


def _noise_grid(seed: int, density: float = 0.35) -> Grid:
    """No structure of any kind. A real engine must stay SILENT here."""
    rng = random.Random(seed)
    return [[(rng.randrange(1, 5) if rng.random() < density else 0) for _ in range(16)]
            for _ in range(16)]


def _shuffled_mirror(seed: int) -> Grid:
    """The mirror board's exact colour histogram, with the structure destroyed. The
    harder control: an engine cannot pass it by noticing the colour distribution."""
    g = _mirror_grid()
    flat = [v for row in g for v in row]
    random.Random(seed).shuffle(flat)
    return [flat[i * 16:(i + 1) * 16] for i in range(16)]


def _transition() -> Tuple[Grid, dict, Grid, str]:
    d = json.loads((HERE / "fixtures" / "vc33_l3_transition.json").read_text())
    return d["before"], d["action"], d["after"], d["note"]


def _mask(g: Grid, n: int, seed: int, only_bg: bool = False) -> Tuple[Grid, Tuple[Tuple[int, int], ...]]:
    """Hide n cells. `only_bg` keeps the holes off the objects, which matters when
    the claim under test is about object shape rather than about cell values."""
    rng = random.Random(seed)
    cells = [(r, c) for r in range(len(g)) for c in range(len(g[0]))
             if not only_bg or g[r][c] == 0]
    hid = tuple(sorted(rng.sample(cells, n)))
    m = copy.deepcopy(g)
    for r, c in hid:
        m[r][c] = HIDDEN
    return m, hid


def build() -> List[Tuple[str, Evidence, Truth, str]]:
    """(label, evidence the engines see, truth the verifier uses, expectation)"""
    out = []

    for label, corrupt in (("static-mirror", False), ("static-mirror-BROKEN", True)):
        g = _mirror_grid()
        if corrupt:
            g[5][12] = (g[5][12] + 1) % 5          # one mirrored cell violated
        m, hid = _mask(g, 24, seed=1)
        out.append((label, Evidence(label, "grid", {"grid": m}, hid), Truth(grid=g),
                    "high error" if corrupt else "low error"))

    for label, corrupt in (("relational-copy", False), ("relational-copy-BROKEN", True)):
        g = _copies_grid()
        if corrupt:
            g[13][13] = 0                          # one object cell removed
        m, hid = _mask(g, 8, seed=2, only_bg=True)
        out.append((label, Evidence(label, "grid", {"grid": m}, hid), Truth(grid=g),
                    "high error" if corrupt else "low error"))

    # --- negative controls: structureless boards. Expectation is SILENCE from the
    # real engines and commitment from the liar; that gap is the noise floor.
    for label, gen in (("noise-no-structure", _noise_grid),
                       ("shuffled-mirror", _shuffled_mirror)):
        g = gen(7)
        m, hid = _mask(g, 24, seed=3)
        out.append((label, Evidence(label, "grid", {"grid": m}, hid), Truth(grid=g),
                    "silence from the real engines; the liar speaks"))

    before, act, after, note = _transition()
    out.append(("vc33-transition",
                Evidence("vc33-transition", "transition",
                         {"before": before, "after": after, "action": act, "note": note}, ()),
                Truth(grid=before, after=after), "graph 0.000 once it has observed"))
    wrong = copy.deepcopy(after); wrong[10][10] = (wrong[10][10] + 1) % 9
    out.append(("vc33-transition-BROKEN",
                Evidence("vc33-transition-BROKEN", "transition",
                         {"before": before, "after": wrong, "action": act}, ()),
                Truth(grid=before, after=wrong), "graph 1.000 -- a different outcome"))
    return out
