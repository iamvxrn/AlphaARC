"""Offline benchmark over the whole public ARC-AGI-3 game set.

Phase 0 of the score push: replace "did it take L1?" with the *official*
competition number, per level, per game, plus a diff against a previous run.

The score the leaderboard reports (arc_agi/scorecard.py) is

    level_score = min(115, (baseline_actions / actions_taken)**2 * 100)  if completed else 0
    game_score  = sum(level_score_i * i) / sum(i over ALL levels of the game)
    aggregate   = mean(game_score over games)

so BOTH depth (deeper levels carry weight i) and efficiency (quadratic in the
action ratio) are first-class. This harness surfaces both.

Runs under the Kaggle starter kit's venv, which owns the `arc-agi` engine:

    KIT=~/ARC-AGI-3-Kaggle-Starter/ARC-AGI-3-Kaggle-Starter
    $KIT/.venv/bin/python python/bench/bench.py --split train
    $KIT/.venv/bin/python python/bench/bench.py --split all --out runs/base.json
    $KIT/.venv/bin/python python/bench/bench.py --split all --vs runs/base.json
"""
from __future__ import annotations

import argparse
import importlib.util
import json
import logging
import os
import random
import sys
import time
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
DEFAULT_KIT = Path.home() / "ARC-AGI-3-Kaggle-Starter" / "ARC-AGI-3-Kaggle-Starter"


def load_agent_class(agent_path: Path):
    spec = importlib.util.spec_from_file_location("bench_agent_module", agent_path)
    if spec is None or spec.loader is None:
        raise SystemExit(f"cannot load agent at {agent_path}")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    if not hasattr(mod, "MyAgent"):
        raise SystemExit(f"{agent_path} must define class MyAgent")
    return mod.MyAgent


def game_metadata(kit: Path) -> dict[str, dict]:
    """Short game id -> its metadata.json (tags, baseline_actions, title)."""
    out = {}
    for p in sorted((kit / "environment_files").glob("*/*/metadata.json")):
        out[p.parts[-3]] = json.loads(p.read_text())
    return out


def resolve_games(split: str, only: str | None, meta: dict) -> tuple[list[str], dict]:
    splits = json.loads((REPO / "python" / "bench" / "splits.json").read_text())
    if only:
        names = [g.strip() for g in only.split(",") if g.strip()]
    elif split == "all":
        names = sorted(meta)
    else:
        names = sorted(splits[split])
    unknown = [g for g in names if g not in meta]
    if unknown:
        raise SystemExit(f"unknown game id(s): {unknown}")
    return names, splits


def play(arc, AgentCls, game_id: str, max_steps: int, seed: int) -> dict:
    os.environ["ARC_AGENT_SEED"] = str(seed)
    env = arc.make(game_id)
    if env is None:
        return {"error": "env_none"}
    agent = AgentCls(
        card_id="bench",
        game_id=game_id,
        agent_name=f"bench.{game_id}",
        ROOT_URL="http://localhost",
        record=False,
        arc_env=env,
        tags=["bench"],
    )
    # Exploration is random, so without a pinned seed every run is a different
    # experiment and a diff means nothing. ARC_AGENT_SEED is read by our agents
    # in __init__ (hence set before construction, below); seeding the global
    # module too covers an agent that uses `random` directly.
    random.seed(seed)
    t0 = time.time()
    agent.main()
    final = agent.frames[-1]
    return {
        "state": str(final.state),
        "levels_completed": int(final.levels_completed),
        "actions": int(agent.action_counter),
        "seconds": round(time.time() - t0, 1),
    }


def per_game_detail(scorecard, game_id: str) -> dict:
    """Pull the best run's per-level breakdown out of the engine's scorecard."""
    for envlist in scorecard.environments:
        if envlist.id.split("-")[0] != game_id:
            continue
        best = max(envlist.runs, key=lambda r: (r.levels_completed, r.score))
        return {
            "score": round(best.score, 4),
            "level_scores": [round(s, 2) for s in (best.level_scores or [])],
            "level_actions": list(best.level_actions or []),
            "level_baselines": list(best.level_baseline_actions or []),
        }
    return {"score": 0.0, "level_scores": [], "level_actions": [], "level_baselines": []}


def ratio_cells(actions: list[int], baselines: list[int], scores: list[float]) -> str:
    """Compact per-level view: 'x1.4' when solved, '·' when not reached/failed."""
    out = []
    for i, base in enumerate(baselines):
        if i < len(scores) and scores[i] > 0 and i < len(actions) and base:
            out.append(f"x{actions[i] / base:.1f}")
        else:
            out.append("·")
    return " ".join(out)


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--kit", type=Path, default=Path(os.environ.get("ARC_KIT", DEFAULT_KIT)),
                    help="Kaggle starter-kit checkout (owns .venv + environment_files)")
    ap.add_argument("--agent", type=Path, default=None,
                    help="agent file defining MyAgent (default: <kit>/agent/my_agent.py)")
    ap.add_argument("--split", default="train", choices=["train", "holdout", "all"],
                    help="which frozen split to run (default: train)")
    ap.add_argument("--games", default=None, help="comma-separated ids, overrides --split")
    ap.add_argument("--max-steps", type=int, default=250, help="per-game action cap")
    ap.add_argument("--repeats", type=int, default=1,
                    help="plays per game; the engine scores a game by its BEST run")
    ap.add_argument("--seed", type=int, default=1,
                    help="pinned RNG seed (the reference agent otherwise seeds from the clock)")
    ap.add_argument("--out", type=Path, default=None, help="write results JSON here")
    ap.add_argument("--vs", type=Path, default=None, help="diff against a results JSON")
    args = ap.parse_args()

    if args.split == "holdout" and not os.environ.get("ARC_HOLDOUT_OK"):
        print("Refusing to run the HOLDOUT split without ARC_HOLDOUT_OK=1.\n"
              "The holdout is the local stand-in for the hidden Kaggle set: every look\n"
              "at it spends generalization evidence. Run it to MEASURE, never to tune.",
              file=sys.stderr)
        raise SystemExit(2)

    kit = args.kit.expanduser()
    agent_path = args.agent or (kit / "agent" / "my_agent.py")
    for p, what in ((kit, "kit"), (agent_path, "agent")):
        if not p.exists():
            raise SystemExit(f"{what} not found: {p}")

    # Absolutize before the chdir below, so every path stays relative to the
    # caller's directory rather than the kit's.
    args.out = args.out.resolve() if args.out else None
    args.vs = args.vs.resolve() if args.vs else None
    agent_path = agent_path.resolve()

    sys.path.insert(0, str(kit))
    sys.path.insert(0, str(kit / "vendor" / "ARC-AGI-3-Agents"))
    # The engine caches game sources in ./environment_files relative to the CWD;
    # run from the kit so every bench shares its already-downloaded cache instead
    # of littering (and re-downloading into) whatever directory we were called from.
    os.chdir(kit)
    logging.basicConfig(level=logging.WARNING, format="%(message)s")

    import arc_agi
    from arc_agi import OperationMode

    meta = game_metadata(kit)
    games, splits = resolve_games(args.split, args.games, meta)
    AgentCls = load_agent_class(agent_path)
    if hasattr(AgentCls, "MAX_ACTIONS"):
        AgentCls.MAX_ACTIONS = args.max_steps

    arc = arc_agi.Arcade(operation_mode=OperationMode.NORMAL)
    results: dict[str, dict] = {}
    t0 = time.time()
    for i, g in enumerate(games, 1):
        print(f"[{i}/{len(games)}] {g} ...", end="", flush=True)
        runs = []
        for rep in range(args.repeats):
            runs.append(play(arc, AgentCls, g, args.max_steps, args.seed + 1000 * rep))
            print(f" {runs[-1].get('levels_completed', '?')}", end="", flush=True)
        best = max(runs, key=lambda r: (r.get("levels_completed", 0), -r.get("actions", 0)))
        results[g] = dict(best)
        results[g]["runs"] = runs
        results[g]["levels_per_run"] = [r.get("levels_completed", 0) for r in runs]
        print(f"  -> best levels={best.get('levels_completed', '?')} "
              f"actions={best.get('actions', '?')} "
              f"({sum(r.get('seconds', 0) for r in runs):.1f}s)", flush=True)

    scorecard = arc.get_scorecard()
    for g in games:
        results[g].update(per_game_detail(scorecard, g))
        results[g]["tags"] = meta[g].get("tags") or []
        results[g]["n_levels"] = len(meta[g].get("baseline_actions") or [])

    holdout = set(splits["holdout"])
    print("\n" + "=" * 78)
    print(f"{'game':6} {'tag':15} {'set':4} {'lvl':>5} {'score':>7} {'runs':>9}   per-level actions/baseline")
    print("-" * 78)
    for g in games:
        r = results[g]
        tag = ",".join(r["tags"]) or "-"
        cells = ratio_cells(r["level_actions"], r["level_baselines"], r["level_scores"])
        per_run = ",".join(str(x) for x in r.get("levels_per_run", []))
        print(f"{g:6} {tag:15} {'HOLD' if g in holdout else 'trn':4} "
              f"{r['levels_completed']:>2}/{r['n_levels']:<2} {r['score']:>7.3f} {per_run:>9}   {cells}")
    print("-" * 78)

    agg = sum(r["score"] for r in results.values()) / max(1, len(results))
    solved = sum(1 for r in results.values() if r["levels_completed"] > 0)
    print(f"AGGREGATE (mean game score) = {agg:.4f}   "
          f"games reaching L1+: {solved}/{len(results)}   "
          f"engine card score: {scorecard.score:.4f}")
    print(f"ceiling reminder: L1 of every game at exact baseline = 3.52; "
          f">10 needs depth (first 2 levels everywhere) or ~3 full games")
    print(f"wall clock: {time.time() - t0:.0f}s  (cap {args.max_steps} actions/game)")

    payload = {
        "split": args.games or args.split,
        "max_steps": args.max_steps,
        "repeats": args.repeats,
        "seed": args.seed,
        "agent": str(agent_path),
        "aggregate": agg,
        "games": results,
    }
    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(payload, indent=1))
        print(f"wrote {args.out}")

    if args.vs:
        old = json.loads(args.vs.read_text())
        shared = sorted(set(old["games"]) & set(results))
        if not shared:
            print(f"\nDIFF vs {args.vs}: no games in common -- nothing to compare.")
            return
        # Compare on the INTERSECTION. The aggregate is a mean over games, so
        # comparing a 17-game run against a 25-game one reads as a large gain that
        # is purely the different denominator -- which is exactly how this diff
        # first reported +0.10 for a change that was worth -0.0003.
        old_agg = sum(old["games"][g]["score"] for g in shared) / len(shared)
        new_agg = sum(results[g]["score"] for g in shared) / len(shared)
        note = ""
        if set(old["games"]) != set(results):
            note = (f"  [compared on the {len(shared)} shared games; "
                    f"{args.vs.name} has {len(old['games'])}, this run {len(results)}]")
        print(f"\nDIFF vs {args.vs} ({old_agg:.4f} -> {new_agg:.4f}, "
              f"{new_agg - old_agg:+.4f}){note}")
        moved = False
        for g in shared:
            o = old["games"][g]
            dl = results[g]["levels_completed"] - o["levels_completed"]
            ds = results[g]["score"] - o["score"]
            if dl or abs(ds) > 1e-6:
                moved = True
                print(f"  {g:6} levels {o['levels_completed']}->{results[g]['levels_completed']} "
                      f"({dl:+d})   score {o['score']:.3f}->{results[g]['score']:.3f} ({ds:+.3f})")
        if not moved:
            print("  no game moved")


if __name__ == "__main__":
    main()
