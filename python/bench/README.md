# Phase 0 — the measurement loop

Everything here exists to answer one question honestly: **what would this agent
score on Kaggle right now**, and did a change move that number.

## The number

`arc_agi/scorecard.py` computes

```
level_score = min(115, (baseline_actions / actions_taken)**2 * 100)   if the level was completed, else 0
game_score  = Σ(level_score_i · i) / Σ(i over ALL levels of the game)
aggregate   = mean(game_score) over games
```

Two consequences drive everything:

- **Depth is weighted by level index.** Level 1 of a 7-level game is worth at
  most `1/28` of that game. Taking L1 of *every* public game at exact baseline
  efficiency scores **3.52** — not 10. Above 10 needs the first two levels
  everywhere, or ~3 games solved end to end.
- **Efficiency is quadratic, and baseline is the ceiling.** 2× the baseline
  actions keeps ¼ of the level's points; beating baseline earns nothing extra
  (`max_score` caps the game at the weight fraction it solved).

Because level 1 carries the *lowest* weight, it is the cheapest place to spend a
budget learning the game's mechanic — and levels 2+ are where a learned model
gets to pay that back at near-baseline efficiency. The scoring function rewards
"learn once, then execute", which is what the MBRL core is for.

## Commands

```
make bench                    # bundle + play the TRAIN split, print the score
make bench-all OUT=python/bench/runs/x.json
make bench VS=python/bench/runs/baseline.json    # diff against a previous run
make holdout                  # the frozen split — measure, never tune
```

`--repeats` matters: the reference agent explores randomly, and the engine scores
a game by its **best** run, so one play is a noisy estimate. The bench pins the
RNG seed (the agent otherwise seeds from the clock) so a diff means a real change.

## The splits

`splits.json` freezes 8 of the 25 public games as a **holdout** — our local
stand-in for the hidden Kaggle set. They were picked from the games that had
never been probed or read as of 2026-08-25; the 9 already-studied games all sit
in `train`. Every look at the holdout spends generalization evidence, so the
bench refuses to run it without `ARC_HOLDOUT_OK=1`.

All 7 click-only games were already contaminated at freeze time, so the holdout
covers `keyboard` / `keyboard_click` games only. Worth remembering when reading
its number: it is a harder-than-average sample for a click-centric agent.

## The control

`agents/random_agent.py` plays uniformly random legal actions. Any mechanism we
build has to beat *that*, not zero — some levels fall to button-mashing.

## The bundler

Kaggle splices only `agent/my_agent.py`, so the submission must be one
self-contained file. `bundle.py` builds it from `python/alphaarc/`, which is
therefore canonical; `make check-bundle` fails if the built file is stale, and
`python/tests/test_bundle.py` pins that the bundle is import-free and computes
the same numbers as the package.
