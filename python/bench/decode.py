"""Reverse-engineer one game: what are its controls, and is its reward visible one step ahead?

The census (census.py) says what CLASS of transition a game has. This says
something sharper and game-specific: for each candidate control, press it
repeatedly from a fresh board and watch the compression profile along it. That is
how vc33 gave up its secret -- its scalar pays off only on the third press, and
the first press looks harmful, so one-step credit rejects it.

    make decode GAME=r11l
    python/bench/decode.py --game r11l --presses 6

Reverse-engineering a game is NOT hardcoding it. The output is a specification for
the general mechanism -- "controls here need N steps of lookahead across a worse
intermediate state" -- and the frozen holdout is what judges whether the
generalization was real.

Refuses holdout games: this reads a mechanic, which is what the holdout withholds.
"""
from __future__ import annotations

import argparse
import json
import logging
import os
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO / "python"))
DEFAULT_KIT = Path.home() / "ARC-AGI-3-Kaggle-Starter" / "ARC-AGI-3-Kaggle-Starter"

GLYPH = ".123456789ABCDEF"


def render(grid, title, max_side=64):
    print(f"--- {title} ({len(grid)}x{len(grid[0])}) ---")
    print("    " + "".join(str(c % 10) for c in range(min(len(grid[0]), max_side))))
    for r, row in enumerate(grid[:max_side]):
        print(f"{r:3} " + "".join(GLYPH[v % 16] for v in row[:max_side]))


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--game", required=True)
    ap.add_argument("--kit", type=Path, default=Path(os.environ.get("ARC_KIT", DEFAULT_KIT)))
    ap.add_argument("--presses", type=int, default=6,
                    help="how far to follow each control (the vc33 payoff was at 3)")
    ap.add_argument("--controls", type=int, default=8, help="candidate controls to try")
    ap.add_argument("--render", action="store_true", help="print the opening board")
    args = ap.parse_args()

    splits = json.loads((REPO / "python" / "bench" / "splits.json").read_text())
    if args.game in splits["holdout"] and not os.environ.get("ARC_HOLDOUT_OK"):
        raise SystemExit(f"{args.game} is in the frozen holdout. Decoding it spends the "
                         f"evidence it exists to provide.")

    kit = args.kit.expanduser()
    sys.path.insert(0, str(kit))
    sys.path.insert(0, str(kit / "vendor" / "ARC-AGI-3-Agents"))
    os.chdir(kit)
    logging.basicConfig(level=logging.WARNING)

    import arc_agi
    from arc_agi import OperationMode
    from arcengine import GameAction
    from agents.agent import Agent
    from alphaarc.mdl import PRIMITIVES, _components
    from alphaarc.perception import background_color

    class Probe(Agent):
        def is_done(self, f, l):
            return True

        def choose_action(self, f, l):
            raise NotImplementedError

    arc = arc_agi.Arcade(operation_mode=OperationMode.NORMAL)

    def fresh():
        agent = Probe(card_id="decode", game_id=args.game, agent_name=f"decode.{args.game}",
                      ROOT_URL="http://localhost", record=False,
                      arc_env=arc.make(args.game), tags=["decode"])
        fr = agent.take_action(GameAction.RESET)
        return agent, fr

    def grid_of(fr):
        return [row[:] for row in fr.frame[-1]] if (fr.frame and fr.frame[-1]) else None

    agent, fr = fresh()
    g0 = grid_of(fr)
    if g0 is None:
        raise SystemExit("no grid after reset")
    bg = background_color(g0)
    if args.render:
        render(g0, f"{args.game} opening board")

    comps = sorted(_components(g0, bg), key=lambda kv: len(kv[1]))
    print(f"\nbackground={bg}, available={fr.available_actions}, {len(comps)} components")
    cands = []
    for col, cells in comps[:args.controls]:
        rs = [c[0] for c in cells]
        cs = [c[1] for c in cells]
        cy, cx = sum(rs) // len(rs), sum(cs) // len(cs)
        cands.append((cx, cy, col, len(cells)))
        print(f"   colour {col:2}  {len(cells):4} cells  rows {min(rs)}-{max(rs)} "
              f"cols {min(cs)}-{max(cs)}  -> click ({cx},{cy})")

    # A move budget is usually a strip that ticks on every action; measuring
    # compression over it measures the clock, so drop the outermost rows if they
    # are uniform-but-changing. Cheap heuristic: report both.
    def levels(g, skip_first_row):
        body = g[1:] if skip_first_row else g
        return [p.savings(body, bg) for p in PRIMITIVES]

    names = [p.name for p in PRIMITIVES]
    print(f"\nFollowing each control for {args.presses} presses from a FRESH board.")
    print("A control whose profile dips before it climbs is invisible to one-step credit.\n")
    for (cx, cy, col, size) in cands:
        agent, fr = fresh()
        g = grid_of(fr)
        prof = [levels(g, True)]
        deltas = []
        cleared = None
        for i in range(args.presses):
            act = GameAction.ACTION6
            act.set_data({"x": int(cx), "y": int(cy)})
            act.reasoning = {"why": "decode"}
            fr = agent.take_action(act)
            n = grid_of(fr)
            if n is None:
                deltas.append("GAME_OVER")
                break
            deltas.append(sum(1 for r in range(min(len(g), len(n)))
                              for c in range(min(len(g[r]), len(n[r])))
                              if g[r][c] != n[r][c]))
            g = n
            prof.append(levels(g, True))
            if fr.levels_completed:
                cleared = i + 1
                break
        moved = [d for d in deltas if d not in (0, "GAME_OVER")]
        tag = "INERT" if not moved else ("CLEARS L1" if cleared else "live")
        print(f"click ({cx:2},{cy:2}) colour {col:2} size {size:4}  {tag}"
              + (f" in {cleared}" if cleared else ""))
        if moved:
            print(f"    cells changed per press: {deltas}")
            for k, name in enumerate(names):
                series = [p[k] for p in prof]
                if len(set(series)) > 1:
                    dip = any(series[i] < series[0] for i in range(1, len(series)))
                    best_at = max(range(len(series)), key=lambda i: series[i])
                    note = ""
                    if dip and best_at > 1:
                        note = f"   <-- DIPS then peaks at press {best_at}: needs lookahead"
                    print(f"    {name:>14}: {series}{note}")


if __name__ == "__main__":
    main()
