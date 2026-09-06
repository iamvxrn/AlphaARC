"""The compressed loop: one command, one number, five minutes.

Why this exists. `make quick` runs the four games the agent already wins, so it
measures what works and is blind to the open problem. The open problem is the
destination rule on the keyboard games -- see python/bench/README.md, "Rejected
#9" and the two position refutations after it. An idea about that has to be
tested on g50t/ls20/sp80/m0r0 or it is not tested at all.

The click pair is not decoration. The `inert` fix looked neutral on keyboard and
cost lp85 -0.284 and vc33 -0.277 with all 16 seeds negative. A keyboard change
that quietly breaks the scoring games is a loss, and nothing in a keyboard-only
run would show it.

    KEYBOARD   g50t ls20 sp80 m0r0    where the question is
    CLICK      vc33 lp85               the regression guard

Paired comparison, because of rule 7. The scoring split carries sd ~0.83, so two
independent means cannot see an effect below about one point. The same seed runs
the same exploration path in both arms, so the per-seed DIFFERENCE cancels that
path and its spread is far smaller than the spread of either arm. Always --vs a
saved baseline; a bare number here is a starting point, never a result.

    python3 python/bench/loop.py --seeds 8 --tag idea-x
    python3 python/bench/loop.py --seeds 8 --tag idea-x --vs runs/loop-base
"""

import argparse
import concurrent.futures as cf
import json
import os
import pathlib
import statistics as st
import subprocess
import sys

KEYBOARD = ["g50t", "ls20", "sp80", "m0r0"]
CLICK = ["vc33", "lp85"]
GAMES = KEYBOARD + CLICK

HERE = pathlib.Path(__file__).resolve().parent
RUNS = HERE / "runs"
DEFAULT_KIT = os.path.expanduser("~/ARC-AGI-3-Kaggle-Starter/ARC-AGI-3-Kaggle-Starter")


def run_seed(seed, tag, kit, steps, repeats):
    """One seed of the compressed set. Returns (seed, parsed json) or (seed, None)."""
    out = RUNS / tag / f"seed{seed}.json"
    out.parent.mkdir(parents=True, exist_ok=True)
    cmd = [
        str(pathlib.Path(kit) / ".venv" / "bin" / "python"),
        str(HERE / "bench.py"),
        "--kit", kit,
        "--games", ",".join(GAMES),
        "--max-steps", str(steps),
        "--repeats", str(repeats),
        "--seed", str(seed),
        "--out", str(out),
    ]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0 or not out.exists():
        return seed, None, proc.stderr[-400:]
    return seed, json.loads(out.read_text()), ""


def group_score(run, games):
    """Mean game score over a group -- the aggregate restricted to those games."""
    g = run["games"]
    vals = [g[k]["score"] for k in games if k in g]
    return st.mean(vals) if vals else 0.0


def band(vals):
    """mean, sd, standard error. sd of one sample is 0, which is honest, not stable."""
    m = st.mean(vals)
    sd = st.pstdev(vals) if len(vals) > 1 else 0.0
    se = sd / (len(vals) ** 0.5) if len(vals) > 1 else 0.0
    return m, sd, se


def load_tag(tag):
    d = RUNS / tag if (RUNS / tag).is_dir() else pathlib.Path(tag)
    return {int(p.stem.replace("seed", "")): json.loads(p.read_text())
            for p in sorted(d.glob("seed*.json"))}


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--seeds", type=int, default=8)
    ap.add_argument("--start", type=int, default=1)
    ap.add_argument("--tag", default="loop-tmp")
    ap.add_argument("--kit", default=os.environ.get("KIT", DEFAULT_KIT))
    ap.add_argument("--max-steps", type=int, default=250)
    ap.add_argument("--repeats", type=int, default=3,
                    help="every recorded baseline is 3; changing it makes runs incomparable")
    ap.add_argument("--vs", default=None, help="a previous --tag, for the paired diff")
    ap.add_argument("--jobs", type=int, default=max(1, (os.cpu_count() or 2) - 1))
    a = ap.parse_args()

    seeds = list(range(a.start, a.start + a.seeds))
    print(f"{len(seeds)} seeds x {len(GAMES)} games across {a.jobs} workers ...", flush=True)

    runs = {}
    with cf.ThreadPoolExecutor(max_workers=a.jobs) as ex:
        futs = [ex.submit(run_seed, s, a.tag, a.kit, a.max_steps, a.repeats) for s in seeds]
        for f in cf.as_completed(futs):
            s, r, err = f.result()
            if r is None:
                print(f"  seed {s} FAILED: {err}", file=sys.stderr)
            else:
                runs[s] = r

    if not runs:
        print("no seed produced a result", file=sys.stderr)
        return 1

    ok = sorted(runs)
    kb = [group_score(runs[s], KEYBOARD) for s in ok]
    cl = [group_score(runs[s], CLICK) for s in ok]

    print(f"\n{'':10} {'mean':>8} {'sd':>7} {'se':>7}   n={len(ok)}")
    for name, vals in (("KEYBOARD", kb), ("CLICK", cl)):
        m, sd, se = band(vals)
        print(f"{name:10} {m:8.4f} {sd:7.4f} {se:7.4f}")

    print(f"\n{'game':6} {'score':>8} {'sd':>7}   levels reached (per seed)")
    for g in GAMES:
        sc = [runs[s]["games"][g]["score"] for s in ok if g in runs[s]["games"]]
        lv = [runs[s]["games"][g]["levels_completed"] for s in ok if g in runs[s]["games"]]
        if not sc:
            continue
        m, sd, _ = band(sc)
        mark = "kb" if g in KEYBOARD else "  "
        print(f"{g:6} {m:8.4f} {sd:7.4f} {mark} {''.join(str(x) for x in lv)}")

    if a.vs:
        base = load_tag(a.vs)
        shared = [s for s in ok if s in base]
        if not shared:
            print(f"\n--vs {a.vs}: no shared seeds, cannot pair", file=sys.stderr)
            return 1
        print(f"\npaired against {a.vs} on {len(shared)} identical seeds")
        print(f"{'group':10} {'delta':>9} {'sd(delta)':>10} {'se':>8}  verdict")
        for name, games in (("KEYBOARD", KEYBOARD), ("CLICK", CLICK)):
            d = [group_score(runs[s], games) - group_score(base[s], games) for s in shared]
            m, sd, se = band(d)
            if len(d) < 2:
                verdict = "one seed -- not a result"
            elif se == 0:
                # Every seed moved by the same amount, which at n>=2 is a real and
                # usually deterministic finding -- not a missing sample. Reporting
                # it as "one seed" hid a delta of exactly zero across 8 seeds.
                verdict = ("IDENTICAL" if m == 0 else
                           "MOVED" if m > 0 else "REGRESSED")
            elif abs(m) < 2 * se:
                verdict = "inside the noise"
            else:
                verdict = "MOVED" if m > 0 else "REGRESSED"
            print(f"{name:10} {m:+9.4f} {sd:10.4f} {se:8.4f}  {verdict}")
        print("\na change must MOVE keyboard and leave click alone; anything else is not a win")
    return 0


if __name__ == "__main__":
    sys.exit(main())
