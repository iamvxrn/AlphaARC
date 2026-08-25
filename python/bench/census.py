"""Phase 1 -- the mechanic census: what KIND of transition does each game have?

Phase 0 established that depth, not breadth, is where the score is, and that 17
of the 25 public games need non-click actions we have never handled. Before
building world-model transition classes we should know which classes actually
occur, and in how many games -- otherwise we are guessing at the coverage of each
piece of work (the mistake that cost us ft09).

So: probe each game with a fixed script and classify what its actions DO.

    make census                      # train split
    python/bench/census.py --games vc33,ls20 --verbose

Output per game: which simple actions move something (and how), how dense the
interactive clicks are, whether clicks act locally or at a distance (a button),
whether an effect toggles, and whether the same action's effect VARIES (a mode /
stateful protocol). Then a coverage ranking: implement class X, address N games.

The holdout is off limits here -- a census READS a game's mechanic, which is
exactly the knowledge the holdout exists to withhold.
"""
from __future__ import annotations

import argparse
import json
import os
import sys
from collections import Counter
from pathlib import Path
from typing import Optional

REPO = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO / "python"))

from alphaarc.mdl import _components  # noqa: E402
from alphaarc.perception import background_color  # noqa: E402

DEFAULT_KIT = Path.home() / "ARC-AGI-3-Kaggle-Starter" / "ARC-AGI-3-Kaggle-Starter"

Grid = list[list[int]]

# A click's effect counts as LOCAL if every changed cell is within this many
# cells of the click; anything further means the click acted at a distance
# (a button), which is a different transition class to model.
LOCAL_RADIUS = 3
# Fraction of the grid that must change for an effect to count as global
# (a repaint / level transition rather than an object-level edit).
GLOBAL_FRACTION = 0.20


# ---------------------------------------------------------------- classifying

def changed_cells(a: Grid, b: Grid) -> list[tuple[int, int]]:
    out = []
    for r in range(min(len(a), len(b))):
        for c in range(min(len(a[r]), len(b[r]))):
            if a[r][c] != b[r][c]:
                out.append((r, c))
    return out


def rigid_translation(a: Grid, b: Grid, bg: int) -> Optional[tuple[int, int, int]]:
    """(color, dr, dc) if exactly one colour's cells moved as a rigid body."""
    moved = None
    for color in {v for row in a for v in row} | {v for row in b for v in row}:
        if color == bg:
            continue
        ca = sorted((r, c) for r in range(len(a)) for c in range(len(a[r])) if a[r][c] == color)
        cb = sorted((r, c) for r in range(len(b)) for c in range(len(b[r])) if b[r][c] == color)
        if not ca or len(ca) != len(cb):
            continue
        if ca == cb:
            continue
        dr, dc = cb[0][0] - ca[0][0], cb[0][1] - ca[0][1]
        if all((r + dr, c + dc) == q for (r, c), q in zip(ca, cb)):
            if moved is not None:
                return None  # more than one body moved -> not a clean rigid translation
            moved = (color, dr, dc)
    return moved


def classify(before: Grid, after: Grid, bg: int, click: Optional[tuple[int, int]] = None) -> str:
    """One label for what a single action did. Deliberately coarse -- these are
    the classes a world model would need SEPARATE transition kinds for."""
    ch = changed_cells(before, after)
    if not ch:
        return "noop"
    total = sum(len(row) for row in before)
    if len(ch) >= GLOBAL_FRACTION * total:
        return "global"
    tr = rigid_translation(before, after, bg)
    if tr:
        return f"translate({tr[1]},{tr[2]})"
    if click is not None:
        cx, cy = click  # (col, row)
        if all(abs(r - cy) <= LOCAL_RADIUS and abs(c - cx) <= LOCAL_RADIUS for r, c in ch):
            return "local_edit"
        return "remote_edit"
    return "edit"


def probe_points(grid: Grid, bg: int, limit: int) -> list[tuple[int, int]]:
    """Click targets: object centroids INTERLEAVED with a coarse sweep.

    Objects first sounds right -- controls are usually small -- but a board with
    dozens of components then eats the whole budget and the sweep never runs. That
    is exactly how ft09 was recorded as "nothing responds to anything" when in
    fact its whole bottom-right panel is clickable: the live region held no small
    components, so no probe ever landed there. Interleave so neither source can
    starve the other.
    """
    objs: list[tuple[int, int]] = []
    for _, cells in sorted(_components(grid, bg), key=lambda kv: len(kv[1])):
        r = sum(c[0] for c in cells) // len(cells)
        c = sum(x[1] for x in cells) // len(cells)
        objs.append((c, r))

    # Coarse-to-fine, so that ANY prefix of the sweep still covers the whole board.
    # Row-major order truncated by the probe limit only ever samples the top of a
    # 64x64 grid -- the second half of the ft09 blind spot: its live region starts
    # at row 36 and no prefix of a row-major sweep ever reached it.
    sweep: list[tuple[int, int]] = []
    h, w = len(grid), len(grid[0]) if grid else 0
    stride = max(4, max(h, w) // 8)
    step = 1
    while stride * step < max(h, w) * 2:
        s = stride * step
        for r in range(s // 2, h, s):
            for c in range(s // 2, w, s):
                if (c, r) not in sweep:
                    sweep.append((c, r))
        step *= 2
    sweep.reverse()  # coarsest lattice first

    out: list[tuple[int, int]] = []
    seen: set[tuple[int, int]] = set()
    for i in range(max(len(objs), len(sweep))):
        for src in (objs, sweep):
            if i < len(src) and src[i] not in seen:
                seen.add(src[i])
                out.append(src[i])
                if len(out) >= limit:
                    return out
    return out


# ------------------------------------------------------------------- probing

def grid_of(frame) -> Optional[Grid]:
    layers = getattr(frame, "frame", None)
    if not layers:
        return None
    for layer in reversed(layers):
        if layer and layer[0]:
            return [row[:] for row in layer]
    return None


def census_game(agent_cls, arc, game_id: str, clicks: int, repeats: int, verbose: bool) -> dict:
    from arcengine import GameAction

    env = arc.make(game_id)
    agent = agent_cls(card_id="census", game_id=game_id, agent_name=f"census.{game_id}",
                      ROOT_URL="http://localhost", record=False, arc_env=env, tags=["census"])
    frame = agent.take_action(GameAction.RESET)
    grid = grid_of(frame)
    if grid is None:
        return {"error": "no grid after reset"}
    bg = background_color(grid)
    actions = 1
    levels_seen = frame.levels_completed

    def step(action: GameAction, data: Optional[dict] = None) -> tuple[Grid, object]:
        nonlocal actions, grid, levels_seen
        if data:
            action.set_data(data)
        action.reasoning = {"why": "census probe"}
        fr = agent.take_action(action)
        actions += 1
        levels_seen = max(levels_seen, fr.levels_completed)
        g = grid_of(fr)
        if g is not None:
            grid = g
        return grid, fr

    avail = list(frame.available_actions or [])
    simple = [a for a in GameAction
              if a is not GameAction.RESET and not a.is_complex() and a.value in avail]

    # (A) what does each simple action do, and does it do the SAME thing twice?
    simple_effects: dict[str, list[str]] = {}
    for act in simple:
        labels = []
        for _ in range(repeats):
            before = [row[:] for row in grid]
            after, _ = step(act)
            labels.append(classify(before, after, bg))
        simple_effects[act.name] = labels

    # (B) how interactive are clicks, and do they act locally or at a distance?
    click_labels: list[str] = []
    live_points: list[tuple[int, int]] = []
    if GameAction.ACTION6.value in avail:
        for (cx, cy) in probe_points(grid, bg, clicks):
            before = [row[:] for row in grid]
            after, _ = step(GameAction.ACTION6, {"x": int(cx), "y": int(cy)})
            label = classify(before, after, bg, click=(cx, cy))
            click_labels.append(label)
            if label != "noop":
                live_points.append((cx, cy))

    # (C) does an effective click TOGGLE (period 2) or cycle?
    toggle = None
    if live_points:
        cx, cy = live_points[0]
        s0 = [row[:] for row in grid]
        step(GameAction.ACTION6, {"x": int(cx), "y": int(cy)})
        s1 = [row[:] for row in grid]
        step(GameAction.ACTION6, {"x": int(cx), "y": int(cy)})
        s2 = [row[:] for row in grid]
        toggle = "toggle" if (s2 == s0 and s1 != s0) else ("cycle" if s1 != s0 else "sticky")

    live = sum(1 for l in click_labels if l != "noop")
    # Guard against over-reading density: if EVERY click "worked" and repeating one
    # click keeps changing the board, the board is probably animating on its own
    # and the clicks are not what moved it. A world model has to treat that as
    # exogenous dynamics, not as an effect it caused.
    animated = bool(click_labels) and live == len(click_labels) and toggle == "cycle"
    return {
        "animated_suspect": animated,
        "actions_spent": actions,
        "levels_completed_during_probe": levels_seen,
        "simple_actions": simple_effects,
        "click_probes": len(click_labels),
        "click_live": live,
        "click_density": round(live / len(click_labels), 3) if click_labels else None,
        "click_labels": dict(Counter(click_labels)),
        "toggle": toggle,
        "verbose_clicks": click_labels if verbose else None,
    }


# -------------------------------------------------------------------- summary

def summarize(game: dict) -> dict:
    """Reduce one game's probe to the transition CLASSES a model must support."""
    classes = set()
    varies = False
    for labels in (game.get("simple_actions") or {}).values():
        kinds = set(labels)
        if kinds == {"noop"}:
            continue
        if len({k for k in kinds if k != "noop"}) > 1:
            varies = True
        for k in kinds:
            if k.startswith("translate"):
                classes.add("object-translation")
            elif k == "global":
                classes.add("global-repaint")
            elif k != "noop":
                classes.add("keyed-edit")
    labels = game.get("click_labels") or {}
    for k in labels:
        if k.startswith("translate"):
            classes.add("object-translation")
        elif k == "local_edit":
            classes.add("click-local-edit")
        elif k == "remote_edit":
            classes.add("click-remote-effect")
        elif k == "global":
            classes.add("global-repaint")
    if game.get("toggle") in ("toggle", "cycle"):
        classes.add("cell-" + game["toggle"])
    if game.get("animated_suspect"):
        classes.add("exogenous-dynamics?")
    if varies:
        classes.add("stateful-mode")
    return {"classes": sorted(classes), "varies": varies}


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--kit", type=Path, default=Path(os.environ.get("ARC_KIT", DEFAULT_KIT)))
    ap.add_argument("--split", default="train", choices=["train", "holdout", "all"])
    ap.add_argument("--games", default=None, help="comma-separated ids, overrides --split")
    ap.add_argument("--clicks", type=int, default=40, help="click probe points per game")
    ap.add_argument("--repeats", type=int, default=3, help="repeats per simple action")
    ap.add_argument("--out", type=Path, default=None)
    ap.add_argument("--verbose", action="store_true")
    args = ap.parse_args()

    if args.split in ("holdout", "all") and not os.environ.get("ARC_HOLDOUT_OK"):
        raise SystemExit(
            "Refusing to census the holdout. A census READS the mechanic -- that is\n"
            "precisely the knowledge the holdout withholds. Use --split train."
        )

    kit = args.kit.expanduser()
    args.out = args.out.resolve() if args.out else None
    sys.path.insert(0, str(kit))
    sys.path.insert(0, str(kit / "vendor" / "ARC-AGI-3-Agents"))
    os.chdir(kit)

    import logging
    logging.basicConfig(level=logging.WARNING)
    import arc_agi
    from arc_agi import OperationMode
    from agents.agent import Agent

    class Probe(Agent):
        def is_done(self, frames, latest_frame):  # never used: we drive by hand
            return True

        def choose_action(self, frames, latest_frame):
            raise NotImplementedError

    splits = json.loads((REPO / "python" / "bench" / "splits.json").read_text())
    meta = {p.parts[-3]: json.loads(p.read_text())
            for p in sorted((kit / "environment_files").glob("*/*/metadata.json"))}
    if args.games:
        games = [g.strip() for g in args.games.split(",") if g.strip()]
    elif args.split == "all":
        games = sorted(meta)
    else:
        games = sorted(splits[args.split])

    arc = arc_agi.Arcade(operation_mode=OperationMode.NORMAL)
    results = {}
    for i, g in enumerate(games, 1):
        print(f"[{i}/{len(games)}] {g} ...", end="", flush=True)
        r = census_game(Probe, arc, g, args.clicks, args.repeats, args.verbose)
        r["tags"] = meta.get(g, {}).get("tags") or []
        r.update(summarize(r))
        results[g] = r
        print(f" {r['actions_spent']} actions, classes={','.join(r['classes']) or '-'}",
              flush=True)

    print("\n" + "=" * 96)
    print(f"{'game':6} {'tag':15} {'clicks live':>11} {'toggle':>7}  {'simple actions that do something':32} classes")
    print("-" * 96)
    for g, r in results.items():
        movers = [a for a, ls in (r.get("simple_actions") or {}).items() if set(ls) != {"noop"}]
        dens = f"{r['click_live']}/{r['click_probes']}" if r.get("click_probes") else "-"
        dens += "*" if r.get("animated_suspect") else ""
        print(f"{g:6} {','.join(r['tags']) or '-':15} {dens:>11} {str(r.get('toggle') or '-'):>7}  "
              f"{','.join(movers) or '-':32} {','.join(r['classes']) or '-'}")
    print("-" * 96)
    print("* = every probe 'worked' AND a repeated click kept changing things: the board may be\n"
          "    animating by itself, so read that game's density as unproven, not as interactivity.")

    cov = Counter()
    for r in results.values():
        for c in r["classes"]:
            cov[c] += 1
    print("COVERAGE -- implement this transition class, address this many games:")
    for c, n in cov.most_common():
        print(f"  {n:2}/{len(results)}  {c}")

    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(results, indent=1))
        print(f"\nwrote {args.out}")


if __name__ == "__main__":
    main()
