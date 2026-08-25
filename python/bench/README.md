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

**The holdout is SEALED, not merely unused** (ruling of 2026-08-25 at train
0.4010). Running it while the engine is structurally incomplete measures what we
cannot play, not how well we generalize — and it is the only overfitting detector
available at the end, so a single early look starts shaping heuristics around it.
Unlock only once the obvious structural bugs on `train` are exhausted and every
control class is closed.

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

## Tried and rejected (do not re-litigate without new evidence)

Both of these were built for the census's headline finding — that the dominant
mechanic, 12 of 17 train games, is a click whose effect lands somewhere the click
is not — and both were **measured worse** than the policy they replaced. The runs
are kept in `runs/` as evidence.

| change | train aggregate | verdict |
|---|---|---|
| baseline (`succ_model.json`) | **0.3145** | kept |
| \+ aim-at-the-anomaly and known controls as candidates (`remote_effect.json`) | 0.2708 | rejected |
| \+ spatially spread object candidates, alone (`spread_only.json`) | 0.1522 | rejected |

**Aim at the anomaly.** Track where each control's effect lands, then score a
candidate by how close that effect is to the residual the drive wants fixed.
Sound in principle; it moved nothing in three synthetic worlds, and on real games
it cost 0.044 — vc33 dropped from level 2 to level 1.

**Spread the candidates.** `object_targets` returns the smallest components first,
and a patterned board breaks into dozens of equal-size ones, so the whole
candidate budget is spent in one corner and a lone button elsewhere is never
offered. Selecting the farthest-apart members of each size tier fixes exactly
that — and is the single worst change measured so far, −0.16. vc33 still reached
level 2 but took 4.6x baseline on level 1 instead of 0.9x, collapsing its score
from 4.34 to 1.14.

The lesson both share: candidate DILUTION is expensive. The scoring function
punishes wasted actions quadratically, so widening the search to find a control
we might be missing costs more than the control is worth. Anything that widens
exploration has to pay for itself in found levels, and neither did. A cheaper
control-discovery path — one that does not spend real actions on candidates the
drive has no reason to prefer — is the open problem.

## The inner loop

`make quick` plays only the four train games that have ever scored (vc33, r11l,
lp85, tn36): **127 s against ~2400 s** for the full train split, with identical
per-game numbers. A change that moves nothing here moved nothing anywhere, so
iterate on `quick`, confirm on `make bench`, and only then spend holdout evidence.

Most of the first day of this push was lost to running the 40-minute suite on
every micro-idea. Don't.

## What reverse-engineering the games has taught the mechanism

Decoding a game is not hardcoding it — the point is to find which TRANSITION CLASS
it needs, then build that class generally and let the holdout judge whether the
generalization was real. Two games are decoded so far, and both landed on the same
structural problem:

- **ft09** — a 3x3 panel of tiles, each click toggling one; a 32-move budget drawn
  on row 63; 2^8 states against 32 moves, so the target must be inferred, not
  searched. A toggle is involutive, so an average of signed one-step deltas
  cancels on it.
- **vc33** — the two colour-9 squares are + and - on ONE SCALAR and are exact
  inverses; the scene rescales wholesale on each press; row 0 is the move budget.
  Reflect savings along the up-direction go 308 -> 200 -> 256 -> 364: **the payoff
  needs three presses in one direction and the first press looks harmful.**

So both of our best-understood games fail for the same measurable reason: the
reward is not visible one step ahead. That is not something a fifth heuristic term
fixes — four have been tried and returned zero (see "Tried and rejected"). It is
what lookahead over a learned model is for.

### Rejected #3: masking the move-budget HUD

Three of the four decoded games draw their move budget as a strip on the board
edge — row 0 in vc33, column 0 in r11l, row 63 in ft09 — and it ticks on every
single action. Compression measured over the raw grid therefore measures the
clock as well as the puzzle, and on tn36-sized signal (±3) a −1-per-action drift
buries it. Detecting such a strip (a border row/column involved in ≥80% of
transitions) and cropping it out looked like pure noise removal — the opposite of
the dilution mistake above.

**Measured worse: 1.3368 → 1.1821 on the quick split.** r11l 0.922 → 0.369,
lp85 0.082 → 0.016, vc33 unchanged.

The reason, measured offline on a synthetic periodic board:

```
primitive        full   minus row0   minus col0
Reflect            16           84           99
Translate         584          560          583
```

**Cropping one line moves Reflect by +68 and +83** — many times the per-step
puzzle signal. So masking does not subtract a small noise term; it relocates the
whole measurement basis, and levels either side of the crop are not comparable.

Two things follow, and they outlive this experiment:

1. **The primitives are not invariant to grid dimensions.** Any future
   segmentation, masking or windowing must fix a basis once and keep it, never
   change the frame mid-episode and compare across the change.
2. The HUD drift is **common-mode** — it hits every candidate click equally, so it
   never changed the ranking it appeared to be corrupting. The thing that looked
   like noise in the signal was not noise in the DECISION.

## What the decoded games are

Eight of the twenty-five, via `make decode GAME=xx`. Decoding is not hardcoding:
the output is a specification for a general mechanism, and the frozen holdout is
what judges whether the generalization was real.

| game | controls | budget strip | needs |
|---|---|---|---|
| vc33 | two squares = +/- on one scalar, exact inverses, saturating at 3; the scene rescales wholesale | row 0 | a valley: Reflect 308→200→256→**364** |
| ft09 | 3×3 panel, click toggles a 6×6 tile; 2^8 states against 32 moves | row 63 | involution; the target must be inferred |
| r11l | a path scene; each control fires **once** | column 0 | one-shot controls |
| tn36 | period-2 toggles, amplitude 3 against a −1/action drift | none | a toggle with a weak signal |
| lp85 | a palette of 4×4 swatches with selection brackets | column 0 | select-then-apply |
| ls20 | keyboard only; 3 of 4 keys move an avatar ~52 cells | — | a GOAL and a route |
| g50t | keyboard; **3 of 5 keys inert** | row 63 | movement, mostly-dead keys |
| sp80 | ACTION2 moves 162 cells a press, ACTION4 34, ACTION5 inert | row 0 and rows 60-63 | a profile that RISES then FALLS |

Facts that generalize, and that cost something to learn:

- **Budget strips sit on a board edge and tick on nearly every action.** Do not
  crop them: the primitives are not invariant to grid dimensions (cropping one
  line moves Reflect by +68 on a synthetic board), and the drift is common-mode,
  so it never changed a ranking it appeared to corrupt.
- **A profile can peak in the middle as easily as it can dip.** vc33 pays on the
  third press; sp80's ACTION4 pays on the second and then declines. `_best_run`
  takes the maximum of a profile rather than its last value, which is what covers
  both — worth knowing before anyone "simplifies" it into a hill-climb.
- **Dead controls are everywhere** — whole key sets on g50t, the entire top row on
  lp85 — and one press each is the cheapest way to find out.
- **`object_targets` ranks HUD fragments ahead of real controls**, because HUD
  segments are small and numerous and the rule is smallest-first. Widening the
  candidate set to compensate was the worst change measured (−0.16); handing over
  to the planner on a streak of dead clicks is what worked.
