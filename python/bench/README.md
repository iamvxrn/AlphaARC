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

## READ THIS BEFORE ANY NUMBER BELOW: one seed is not a measurement

Measured 2026-08-26, same code at HEAD, same `--repeats 3`, four seeds:

| seed | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| quick aggregate | 1.7087 | 1.0019 | 0.5205 | **2.9709** |

**mean 1.5505, sd 1.0653, a 5.7x spread.** vc33 alone reads 0.845 to 8.786 with
identical code, r11l 0.111 to 1.778. The engine scores a game by its BEST run, and
whether vc33 stumbles into level 2 within three repeats is close to a coin flip.

Everything on this page below was judged on ONE seed, and several of those
judgements compared against a baseline taken at a different seed or a different
`--repeats`. Two consequences, both uncomfortable:

- The headline **"+27%, 0.3145 -> 0.4010"** and every rejection below are inside
  a +/-1.0 noise band. They are not evidence. Treat them as UNMEASURED, not as
  settled, and do not cite them as reasons a direction is closed. Only one result
  on this page is clearly outside the band -- ordering candidates by shape rarity,
  which took every game to ZERO levels in every repeat, a qualitatively different
  event from an unlucky seed.
- `--repeats` is not a speed knob. Repeat *r* plays with seed `s + 1000r`, so
  best-of-3 is a superset of best-of-2 and always scores at least as high: the
  same agent reads **1.7087 at repeats=3 and 0.3383 at repeats=2**. A day was
  spent calling changes catastrophic against a number taken at the other setting.

The tooling now enforces the fix:

```
make quick                       # ONE seed: a smoke test. Never a measurement.
make quick-n SEEDS=4 TAG=mychange              # the inner loop, ~10 min
make quick-n SEEDS=4 TAG=mychange VSDIR=base   # ... compared against runs/base
python/bench/seeds.py runs/x/seed*.json --vs runs/base/seed*.json
```

`seeds.py` compares **paired on identical seeds** -- the agent's RNG stream is the
same for a given seed, so most of the variance is common to both variants and
cancels in the per-seed difference. Unpaired, an effect needs ~1.0 to be visible
and our entire score is 1.5. It reports a change as REAL only when the mean
difference clears twice its own standard error AND every seed agrees on the sign.
`runs/base/seed1..4.json` is the current HEAD baseline; regenerate it whenever
HEAD's behaviour changes.

`bench.py --vs` now REFUSES to diff two runs taken at different
repeats/max_steps/seed, and every summary stamps its settings.

**What pairing buys, measured on the first change put through it** (keying the
one-step policy's learned values by the board-invariant control signature instead
of by pixel coordinates):

```
raw aggregate      sd 1.0578        <- what every past decision was read against
paired difference  sd 0.0144        mean +0.0095, sem 0.0072
```

**~74x more resolution.** The loop can now see effects of about 0.03, where before
it could not see 1.0 -- on a total score of 1.5. That change itself came out
"too small to call": two seeds unchanged, two slightly positive. Neutral, kept
because it puts both policies on one naming scheme, and reported as neutral
rather than as a win.

A tie is not a disagreement: many changes only alter behaviour on some boards, so
`seeds.py` judges sign agreement among the seeds that actually MOVED and reports
the rest as unchanged.

## Tried and rejected -- ALL of it measured on one seed, so read the section above first

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

All seventeen train games, via `make decode GAME=xx` (the eight holdout games stay sealed). Decoding is not hardcoding:
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
| dc22 | all four keys behave identically: 9 cells once, then only the clock; Count collapses 688→80 | — | a 3×3 **cursor** moved by keys, then applied |
| m0r0 | 4 of 5 keys move ~100 cells; the board is two halves with two 5×5 markers | — | two-panel comparison |
| re86 | period-5 blocks: ~61-cell responses alternate with ~53; the mode shows as Count 82 vs 78 | — | `stateful-mode` |
| lf52 | two panels of 4×4 tiles; clicking a tile's corner flips its marker ONCE (29 cells), then that tile is inert; all 5 keys dead | none found | one-shot per tile |
| s5i5 | two bordered boxes, each holding a **−/+ pair**; each pair edits a bar drawn FAR AWAY (box A → the E block at rows 9-11, box B → the B bar at cols 9-11) | row 63 | remote effect on a scalar |
| su15 | a MOVEMENT game played with clicks: the 3x3 `F` block is an avatar, a dashed diagonal marks the route to the 9-blob goal, and a 5-cell colour-0 plus marks the NEXT waypoint. Clicking the plus teleports the avatar into it. Exactly one live cell on the board at a time; the single key is inert | row 63 | a goal and a route, like ls20 -- but clicked |

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
- **`stateful-mode` is blocked by the state REPRESENTATION, not by the credit.**
  dc22's mode is a cursor position, which moves 9 cells out of 4096 -- the
  per-primitive level vector, which is our entire notion of state, cannot see it.
  Quantising the model key fixed a real defect (raw levels drift with the clock and
  never repeat, so the model accumulated singletons) and cannot fix this one: the
  variable is missing from the representation. The class needs a state that carries
  WHERE the last small change happened, not only how compressible the board is.
- **`object_targets` ranks HUD fragments ahead of real controls**, because HUD
  segments are small and numerous and the rule is smallest-first. Widening the
  candidate set to compensate was the worst change measured (−0.16); handing over
  to the planner on a streak of dead clicks is what worked.

### Rejected #4: two strikes before a control is written off

Decoding sc25 showed that **every one of its four keys absorbs its first press** and
only starts working on the second. `RunPlanner` marks a control inert after ONE
silent press, and `inert` is permanent until the board is replaced, so it writes
off the entire game after four actions and never returns. The rule looked plainly
wrong, and requiring two consecutive silences instead is nearly free.

**Measured worse, and worse at its own job.** sc25 stayed at zero -- the one-strike
rule was not what was blocking it -- and lp85 lost its level, 1.7087 -> 1.7043.

lp85 is a knife-edge case worth knowing about: across six runs it takes its level
in exactly one repeat of three when it takes it at all, so its 0.018 flips to 0.000
under small perturbations. That makes it a poor tie-breaker, not a reliable signal.

The lesson repeats one already on this page: a defect being real is not evidence
that fixing it buys anything. Nine changes have now been measured against the
board; the ones that paid all came from decoding a game and building exactly what
the decode specified.

### Rejected #5: re-ordering the click candidates (two variants, both worse)

This page already recorded the pathology, from decoding lp85: *`object_targets`
ranks HUD fragments ahead of real controls, because HUD segments are small and
numerous and the rule is smallest-first.* Decoding s5i5 and su15 measured how far
it goes. A board carries many equal-smallest components, and they take **all
eight** candidate slots:

| game | what fills all 8 slots | live among them |
|---|---|---|
| su15 | 20 one-cell dots of a dashed diagonal path | **0** |
| s5i5 | 10 pixels shattered off diagonal plus-markers | **0** |
| lf52 | 8 corner pixels of the `1EE1` tiles | 6, all one-shot |

The zeros are measured, not inferred: `decode` tags a click "live" if any cell
changed, and every action ticks the move clock, so its tag was true for provably
dead clicks. A probe that subtracts a null click's effect found that in su15 and
s5i5 **not one** of the eight offered candidates does anything at all, while the
controls that do exist sit at rank 21 and beyond. No credit-assignment fix can
reach them.

Two orderings were then measured -- badly. Both runs were taken at `repeats=2`
and compared against 1.7087, which was taken at `repeats=3`. The correct control,
HEAD at repeats=2 seed 1, is **0.3383**:

| variant (repeats=2, seed 1) | quick | verdict |
|---|---|---|
| HEAD control | 0.3383 | -- |
| rank by shape-class RARITY | **0.0038** | genuinely worse: ZERO levels in every game and repeat |
| keep smallest-first, cap any one class at half the slots | 0.2750 | **not distinguishable** from control -- lp85, tn36 and vc33 identical to three decimals, only r11l differs, and r11l swings 0.111-1.778 across seeds by itself |

So only the rarity ordering is actually rejected. The per-class cap is UNMEASURED:
it needs `make quick-n SEEDS=4 TAG=cap VSDIR=base` to say anything.

Both looked excellent offline. Rarity put su15's only live control first (it was
never offered) and made five of s5i5's eight slots live. The cap left the opening
candidate lists of vc33 and tn36 **bit-identical** while still surfacing one live
control in each of su15 and s5i5 -- it is not even a widening, the candidate count
is unchanged.

Why the "bit-identical" check was worthless, and the real lesson: **the opening
frame is not the experiment.** vc33 rescales its whole scene on every press, so
its candidate list is regenerated from a different board at every step; equality
at step 0 says nothing about step 5. Any future check of a perception change has
to diff the candidate lists ALONG A TRAJECTORY, not on one frame.

And why the ordering is so much more load-bearing than it looks: rank is not a
tie-breaker, it is the policy's PRIOR. `Policy.choose_click` scores a candidate
with `residual_bonus / (i + 1)` = 0.15 at rank 0 falling to 0.019 at rank 7, while
the learned terms enter at `w = 0.02` times a drive gain that starts at zero. For
the whole opening -- the part the quadratic efficiency term punishes hardest --
the order IS the policy. Reordering candidates is not "offering a different menu";
it is rewriting the prior over every board the agent will ever see.

What survives: the reachability of s5i5's and su15's controls is a REAL defect,
measured -- not one of the eight offered candidates in either game is live. Ranking
by rarity is a genuinely bad answer to it. Everything else about this direction is
still open, and the per-class cap in particular was never actually measured.

Two things this cost, worth keeping:

1. **Diff the candidate list offline BEFORE spending a bench run** -- one RESET per
   game and a printed top-8 is nearly free. Diff it along a TRAJECTORY rather than
   on the opening frame: vc33 redraws its whole scene every press, so equality at
   step 0 says nothing about step 5. (The claim that this cost vc33 its second
   level was itself a noise artefact -- see the section at the top of this file.)
2. **`decode` was lying, and is now fixed.** Its "live" tag counted any changed
   cell, and every action ticks the move clock, so it reported eight live controls
   in su15 where there are none -- which is how these games read as decoded when
   they were not. It now follows every control first and calls the clock what
   changes IDENTICALLY across all of them at the same press index (all runs start
   from a fresh board, so the clock is in the same state for each). A null click
   cannot do this job: a click the engine does not accept never ticks the clock at
   all. `--points x,y;x,y` probes chosen cells, for checking what the ordering
   crowds out.
