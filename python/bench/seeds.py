"""Aggregate several single-seed bench runs into one honest number.

A single `make quick` is NOT a measurement. Same code, same repeats=3, four
seeds, measured 2026-08-26:

    seed 1  1.7087      seed 3  0.5205
    seed 2  1.0019      seed 4  2.9709

-- a 5.7x spread, sigma ~1.0 on a mean of 1.55, because the engine scores a game
by its BEST run and whether vc33 stumbles into level 2 is close to a coin flip.
Every change measured before this existed was judged on one seed against a
baseline taken at a different one, which is how a +27% and an -84% were both
reported for differences well inside the noise.

Two rules follow, and the tooling here exists to enforce them:

1. **Average over seeds.** One seed tells you nothing.
2. **Compare PAIRED, on the same seeds.** The agent's RNG stream is identical
   for a given seed, so most of the variance is common to both variants and
   cancels in the per-seed difference. An unpaired comparison of means needs a
   ~1.0 effect to see anything; our whole score is 1.5.

    make quick-n SEEDS=4                       # measure, writes runs/seedN_*.json
    python/bench/seeds.py runs/seed*_new.json --vs runs/seed*_old.json
"""
from __future__ import annotations

import argparse
import json
import statistics
from pathlib import Path


def load(paths):
    out = {}
    for p in paths:
        d = json.loads(Path(p).read_text())
        key = (d["repeats"], d["max_steps"])
        out.setdefault(key, {})[d["seed"]] = d
    if len(out) > 1:
        raise SystemExit(f"runs use different settings and cannot be pooled: {sorted(out)}")
    return next(iter(out.values())) if out else {}


def agg(run):
    g = run["games"]
    return sum(v["score"] for v in g.values()) / max(1, len(g))


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("runs", nargs="+", help="one JSON per seed")
    ap.add_argument("--vs", nargs="*", default=None, help="the baseline's per-seed JSONs")
    args = ap.parse_args()

    new = load(args.runs)
    seeds = sorted(new)
    aggs = [agg(new[s]) for s in seeds]
    print(f"{len(seeds)} seeds {seeds}")
    for s, a in zip(seeds, aggs):
        print(f"  seed {s:<4} {a:.4f}")
    sd = statistics.stdev(aggs) if len(aggs) > 1 else 0.0
    print(f"  MEAN {statistics.mean(aggs):.4f}   sd {sd:.4f}   "
          f"range {min(aggs):.4f}-{max(aggs):.4f}")

    if not args.vs:
        return
    old = load(args.vs)
    shared = sorted(set(new) & set(old))
    if not shared:
        raise SystemExit("no seed appears in both sets: a paired comparison is impossible")
    if set(new) != set(old):
        print(f"\n[paired on the {len(shared)} shared seeds only]")

    diffs = [agg(new[s]) - agg(old[s]) for s in shared]
    m = statistics.mean(diffs)
    print(f"\nPAIRED DIFF over seeds {shared}")
    for s, d in zip(shared, diffs):
        print(f"  seed {s:<4} {agg(old[s]):.4f} -> {agg(new[s]):.4f}  ({d:+.4f})")
    if len(diffs) > 1:
        sdd = statistics.stdev(diffs)
        se = sdd / len(diffs) ** 0.5
        print(f"  mean {m:+.4f}   sd {sdd:.4f}   sem {se:.4f}")
        # Not a p-value: with 4 seeds the t distribution is too wide to pretend.
        # The honest statement is whether the effect clears twice its own
        # standard error, and whether every seed agrees on the sign.
        if all(abs(d) < 1e-9 for d in diffs):
            print("  -> IDENTICAL on every seed: the change did not alter behaviour.")
            return
        agree = all(d > 0 for d in diffs) or all(d < 0 for d in diffs)
        if abs(m) > 2 * se and agree:
            print(f"  -> REAL: |mean| > 2*sem and all {len(diffs)} seeds agree on the sign")
        elif agree:
            print(f"  -> weak: all seeds agree on the sign, but |mean| < 2*sem. More seeds.")
        else:
            print(f"  -> NOT MEASURABLE: the seeds disagree on the sign. This is noise.")
    else:
        print(f"  mean {m:+.4f}  (one seed -- not a measurement)")

    per = {}
    for g in set().union(*[set(new[s]["games"]) for s in shared]):
        ds = [new[s]["games"][g]["score"] - old[s]["games"][g]["score"] for s in shared
              if g in new[s]["games"] and g in old[s]["games"]]
        if ds and any(abs(d) > 1e-9 for d in ds):
            per[g] = ds
    if per:
        print("\nper game (paired differences):")
        for g, ds in sorted(per.items(), key=lambda kv: -abs(statistics.mean(kv[1]))):
            sign = "all agree" if (all(d > 0 for d in ds) or all(d < 0 for d in ds)) else "MIXED"
            print(f"  {g:6} {statistics.mean(ds):+8.3f}   {[round(d, 2) for d in ds]}  {sign}")


if __name__ == "__main__":
    main()
