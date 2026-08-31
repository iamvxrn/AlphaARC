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
`runs/base/seed1..16.json` is the current HEAD baseline (mean **1.5124**); regenerate it whenever
HEAD's behaviour changes. `runs/base_preclock/` is the baseline before the
move-budget fix, kept so that comparison stays reproducible, and
`runs/handover_on/` is the rejected variant described under "the move budget"
below.

`bench.py --vs` now REFUSES to diff two runs taken at different
repeats/max_steps/seed, and every summary stamps its settings.

**Four seeds is not enough for an ABSOLUTE number either.** The same HEAD reads
1.5505 over seeds 1-4 and **1.0967 over seeds 1-8** -- seeds 5-8 are all below the
mean. Quote absolute scores with their seed count, and prefer the paired
difference, which is what the loop can actually resolve.

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

## When a change is not measurable, keep it only on these grounds

"Not measurable" means the effect cannot be told from zero at 2*sem -- NOT that it
is zero. Some changes will never be resolvable: an effect of +0.15 against sd 0.51
needs about **50 seeds per arm**, five hours of wall clock, for +11%. So there has
to be a rule, or "principled" quietly becomes a licence to ship anything.

Keep an unresolved change only when ALL of:

1. it fixes an **inconsistency between what the code does and what it claims** --
   not a preference, not an intuition about what should help;
2. it adds **no new tunable constant** (a constant needs a sweep, and a sweep needs
   resolution we do not have);
3. its measured mean over >=16 seeds is **not negative**.

And claim nothing about the score. Report it as a correctness fix whose effect was
unresolved, and say so in the commit message.

**And 8 seeds still over-estimates.** The known-dead change below read **+0.287**
over seeds 1-8 and **+0.146** over 1-16 -- the first eight happened to contain a
+1.55. Use 16 before believing a number, and treat 4 as a screen.

## The move budget made every dead control look alive -- in the AGENT

`decode` had this bug and it was fixed on 2026-08-26; the same morning it turned
out all three policies had it too, where it costs actual score. Each asked

```python
if grid == self._prev_grid:      # nothing happened -> taboo it
```

and every accepted action ticks a move-budget strip -- row 0 in vc33, column 0 in
r11l, row 63 in ft09, g50t, s5i5, su15 -- so the boards are NEVER equal. Inhibition
of return never fired, `RunPlanner.inert` was never set, and `HybridPolicy`'s
"switch after five dead clicks" was unreachable code; only its 25-action clock ever
fired.

Traced on vc33 level 2, seed 4: **26 consecutive actions** on the same four
controls, every one returning a compression delta of -1 or -2, out of the 68 the
level cost against a baseline of 18. Level 1 had gone in 6 actions against 7.

`alphaarc/clock.py` fixes it. The clock cannot be found from WHICH cells move --
su15 eats its budget bar two cells at a time, so no cell repeats and an
intersection over transitions is empty. What is stable is the LINE. So count how
often each row and column is touched and treat a change confined to lines that move
on >=80% of transitions as the clock. A game with no strip (tn36) never has such a
line and falls back to exact equality, which is correct there. A board that changes
SHAPE is never the clock -- vc33 rescales its whole scene on a press.

**This is not the rejected HUD mask.** That cropped the strip out of the
COMPRESSION MEASUREMENT and moved Reflect by +68. This answers a different
question -- "did anything happen?" -- and the answer to it was simply wrong.

Measured paired over 8 seeds:

| | aggregate | tn36 | vc33 | r11l |
|---|---|---|---|---|
| the fix | **+0.1993** +/- 0.1591 | **+0.846, all 8 seeds agree** | -0.089 mixed | +0.040 mixed |

tn36 clears the bar and every seed agrees: a REAL per-game win. The aggregate leans
positive (6 of 8 seeds) but does not clear 2*sem, so it is reported as a bug fix
with one resolved win, not as a score improvement.

### ...and then the hand-over threshold, calibrated for the first time

Making `HybridPolicy.dead_run` clock-aware is one more line, and it is NOT part of
the bug fix: `dead_streak=5` was chosen while that counter could not increment, so
it had never been calibrated against a counter that works. Swept, paired over 4
seeds, each arm on top of the clock fix:

| dead_streak | paired diff | note |
|---|---|---|
| 5 | +0.177 +/- 0.200 | **r11l -0.482, all seeds agree** -- it hurts |
| 10 | +0.138 +/- 0.199 | |
| 20 | +0.089 +/- 0.052 | no game harmed; a third of the spread |

At 5 the hybrid hands vc33 and r11l to the planner around action 13, and
`HybridPolicy`'s own docstring records the planner scoring 0.000 on vc33 against
the one-step policy's 4.343. **20 against `switch_after=25`** means the streak only
pre-empts the 25-action clock when a level is nearly all dead clicks -- exactly when
the cheap policy has nothing left to offer -- which is why it changes so little and
varies so little.

Confirmed at 8 seeds: **+0.0666, sem 0.0302**. That clears 2*sem, but one seed of
eight moves -0.0034, so `seeds.py` withholds the REAL verdict and this is shipped
as a small positive rather than claimed as a win. Defaults are now
`dead_streak=20, clock_dead_run=True`; `ARC_DEAD_STREAK` and `ARC_CLOCK_DEADRUN=0`
re-run the sweep from the bench without editing code.

The criterion was NOT relaxed to let this pass. Worth remembering next time: the
temptation to widen the bar arrives precisely when your own change is just under
it.

### What the aggregate hid: the metric under-reports RELIABILITY

The two changes above moved quick from 1.0967 to 1.3627, which looks slight. Per
game over the same 8 seeds it is not slight at all:

| game | levels taken, before -> after | L1 cost, mean x baseline |
|---|---|---|
| **vc33** | `[2,1,2,2,1,1,1,2]` -> **`[2,2,2,2,2,2,2,2]`** | 2.1 -> 1.9 |
| **lp85** | 3 of 8 seeds -> **6 of 8** | 13.0 -> **5.3** |
| **tn36** | 8 of 8 -> 8 of 8 | 3.0 -> **1.6** |
| **r11l** | 8 of 8 -> 8 of 8 | 4.2 -> **3.2** |

Every game improved. The aggregate barely moved because a game is scored by its
BEST run of three, so the lucky runs were already being counted -- what improved is
the TYPICAL one. Two things follow:

- **Read the per-game levels and per-level actions, not only the aggregate.** A
  change that makes the agent reliable looks almost free on the headline number.
- Reliability is the precondition for DEPTH, which is where the metric's weight
  actually is. vc33 cannot reach level 3 until it reaches level 2 every time, and
  as of today it does.

The best-of-N shape is the real Kaggle scoring rule, so it is the right target --
but when judging a change, look at whether the floor came up, not only the ceiling.

### Rejected #6: backing the state key off to a state-free fact

The handoff called the state key the next wall: "the model is consulted
successfully only 18-25% of the time, so three lookups in four miss." Both halves
of that turned out to need correcting, and the correction is worth more than the
change was.

**The premise was wrong as stated.** `Policy` was instrumented to record every
lookup it makes -- the level vector, every candidate's token, the one chosen --
across the four scoring games at two seeds. How often the key `(token, levels//16)`
finds anything at all:

| | vc33 | r11l | lp85 | tn36 |
|---|---|---|---|---|
| seed 1 | 49.9% | 42.9% | 84.4% | 25.2% |
| seed 4 | 41.5% | 41.8% | 85.3% | 42.9% |

25% is **tn36**, not the rule. The model answers about half of all lookups.

**And it is a trade, not a defect.** Twelve key schemes replayed over the same
traces, scored on two numbers instead of one -- how often the key answers, and
whether the answer is still true. Coarsening buys reach and pays in truth, and the
optimum differs per game, so no single bucket size is the answer. At bucket 4 on
vc33 the sign of the prediction is right 100% of the time and the reach falls to
42%; state-free reaches 80% and the sign is right 71%. Two schemes that looked
promising are dead outright: **`argmax` (which primitive dominates) and `rank
order` reproduce the state-free numbers exactly**, on every game -- the dominant
primitive never changes inside a game, so those keys carry zero bits.

**What was built.** The policy asks the model two different questions and only one
needs the exact successor: "what will this control do here?" needs the fine key,
but "is it dead here?" is a binary. So: fine key first; on a miss, fall back to a
fact that needs no state at all -- *a control that has never once moved the board,
from any state, is dead here too* -- abstaining the moment it works once, which is
what keeps a toggle (tn36) and a saturating scalar (vc33's + is dead at saturation
and live again after a -) out of it. The run-planner has had exactly this fact all
along, as `inert`; the one-step policy, which plays the opening of every level, did
not. Replayed offline it answered more lookups on 8 of 8 game-seeds (vc33 50->55%,
lp85 84->89%, tn36 25->35%) with the dead calls no less precise (vc33 100->98%,
r11l 78->85%, lp85 98->98%).

**Measured paired, 16 seeds: -0.0336, sem 0.0695. Not measurable, and the mean is
negative** -- so it fails the third clause of the keep-rule at the top of this file
and was reverted. `runs/backoff/` is kept as the evidence.

The shape of the failure is the useful part. Six seeds improved, four worsened, six
were untouched; the mean is dragged under by **seed 7 alone, vc33 -4.16** -- a lost
level 2. That is exactly what the mechanism's one flaw predicts: the suppression is
**self-confirming**. One unlucky dead press writes a control off permanently, and
being written off prevents the press that would clear it. The decaying taboo avoids
this by fading; this had no decay at all.

The obvious repair -- demand evidence from two DISTINCT states before making a
state-free claim, two being the smallest plural and no new constant -- was tested
offline first and **erases the entire gain**: vc33 returns to 49.9% reach and 47
dead calls, i.e. exactly HEAD. The extra reach came from precisely the
single-observation aggressiveness that causes the lock-out. There is no version of
this idea that keeps one without the other, which is why it is recorded as closed
rather than as pending.

**One measurement from the traces outlives the change** and belongs to candidate
reachability (Rejected #5's open half), not to the model: on lp85 the policy is
offered **16 candidates per step and only 2 tokens are ever live in the whole run**,
with 94% of transitions doing nothing. A perfect dead-detector there suppresses 14
of 16 and still chooses between the same two. That is a candidate-set limit, and no
amount of world model reaches past it.

## An absence of reward is not a known absence of effect

`Policy` scored a control the world model had **already watched do nothing from
this exact state** identically to one it had never seen: both took the
`predicted = 0.0` branch, and both then collected the rank prior and the
untried-optimism. Measured on vc33 seed 4: of 169 successful `succ` lookups, **142
predicted no change at all**, while 52 of 90 transitions were nothing but the move
clock. The knowledge was there and the policy threw it away every step.

The fix uses no new constant: the prior and the optimism are what we spend where we
do NOT know, so where the model has an answer for this control in this state, the
answer stands and the guesses get no vote. That is the information-gain criterion
the MBRL plan asks for, applied to the model already in the code.

This also came out of ruling out the obvious alternative first. Making the taboo
decay slower is what the 58% dead-click rate seems to call for, and it is measured
NOT to work (0.92: +0.098 +/- 0.548; 0.97: +0.009 +/- 0.490, with vc33 swinging
+3.66 to -5.52) -- because deadness is CONDITIONAL. vc33's scalar saturates at three
presses, so its + button is dead at saturation and live again after a -, and a
longer taboo just locks out the button the game needs. Time was the wrong variable;
state was the right one, and `succ` was already keyed by it.

Measured paired, **16 seeds**: quick 1.3660 -> 1.5124, **+0.1464, sem 0.1281**.
10 of 16 seeds positive, 5 negative, 1 unchanged. That does NOT clear 2*sem, so no
score claim is made. Kept under the three-part rule at the top of this file: it
removes an inconsistency, adds no constant, and its mean is not negative.

## Name a control by its RANK among its peers, not by where it sits

The object signature was colour + size-as-a-fraction + **position in eighths of
the frame**, and the position is what broke at the level seam it was built for.
vc33's two colour-9 buttons sit at eighth (3,7) and (4,7) on level 1 and at (3,0)
and (4,0) on level 2 -- same colour, same size bucket, same rows, MIRRORED column
-- so the signature called them four different controls and the mechanic was
learned twice. Level 1 costs 6 actions against a baseline of 7; level 2 cost 68
against 18.

What DOES survive the mirroring is the pair's **order**. So a control is now named
by what it is plus how many objects of the same colour and size bucket come before
it in reading order. Rank does not collapse distinct objects the way dropping
position entirely would -- two identical boxes on one board are peer 0 and peer 1,
which is the distinction the coordinate was there to make.

Measured paired, **16 seeds**: quick **1.5124 -> 2.6091, +1.0967, sem 0.2653**.
13 of 16 seeds positive, 3 negative. The mean clears 2*sem by a factor of four
(t = 4.1), so the FIRST half of this file's criterion is met with room to spare;
the second half -- every seed agreeing on the sign -- is not, so `seeds.py` still
prints NOT MEASURABLE. Read it as: the effect is real by any ordinary statistical
standard and the per-seed spread stays large, which is what vc33 always does.

Per game, paired over the 16:

| game | mean | agreement |
|---|---|---|
| vc33 | **+2.222** | 13 of 16 positive, swings -5.53 to +8.06 |
| tn36 | +1.206 | 12 of 16 positive |
| r11l | +0.613 | 11 of 16 positive |
| lp85 | +0.345 | **all 16 agree** |

This is the largest measured change in the project's history. **It buys efficiency
AFTER the seam, not new levels** -- and the difference matters, because the obvious
reading of a +72% aggregate is wrong. Levels reached are identical in both arms
(r11l 1, tn36 1, vc33 2 on all 16 seeds); the one game whose depth changed is lp85,
which goes from taking its level in 7 of 16 seeds to 16 of 16. Everything else is
actions. On levels actually completed, as a multiple of baseline:

| level | base | peer-rank |
|---|---|---|
| vc33 **L1** (nothing to carry yet) | x1.83 | x1.75 |
| vc33 **L2** (the first level AFTER a seam) | x5.95 | **x2.27** |
| tn36 L1 | x1.58 | x1.02 |
| r11l L1 | x2.79 | x1.91 |
| lp85 L1 | x5.02, taken in 7 of 16 seeds | x2.74, taken in **16 of 16** |

Read the first two rows together: vc33's level 1 hardly moves, because there is
nothing to carry INTO the first level, while level 2 -- the first level on the far
side of a seam -- more than halves. A change that merely made the agent cleverer
would have moved both. That asymmetry is the mechanism showing itself, and it is
the reason to believe the aggregate rather than to suspect the seeds.

## The movement class: the avatar detector was never the blocker

Five train games are movement games and four of them score zero. `RunPlanner._route`
-- the only thing that gives a movement game a goal -- is gated on `rigid_move`
having found an avatar and at least two directions, so the first question is
whether that gate ever opens. Probing every key of every movement game from a
fresh board (`scratchpad` probes, three presses each):

| game | what the detector saw | what is actually there |
|---|---|---|
| ls20 | 3 directions, colour 12 | works, and must not be disturbed |
| sp80 | 4 directions, an 80-cell colour-9 avatar | works |
| m0r0 | 2 directions, both VERTICAL | two 25-cell markers, one per panel; horizontally they MIRROR -- (0,-5) and (0,+5) on one key -- so the union is not a translation |
| g50t | **nothing, on all five keys** | a 24-cell colour-9 shape moving (+6,0) on ACTION2 and (0,+6) on ACTION4, perfectly rigidly -- inside 119 cells of colour 9, most of them scenery |

So `rigid_move` tested a whole COLOUR as one body, and g50t's avatar is a
component inside a colour. Fixed by adding a per-component test -- as a
**back-off, not a replacement**, and that distinction was measured: replacing the
colour rule with "the largest shape that translated" re-identified ls20's avatar
from colour 12 to colour 9 and cost it its level, **0.385 -> 0.000 at seed 2**.
With the colour rule answering first, ls20 and m0r0 return their original answers
bit for bit and g50t gains two directions where it had none.

**And it changes no score, which is the actual finding.** Instrumented over a full
250-action run at seed 1:

- **ls20** -- `_route` fires on **243 of 249** calls, with the avatar found and all
  four directions learned. Zero levels completed. The mechanism runs constantly
  and does not win.
- **g50t** -- now learns five directions and a 24-cell avatar, and `_route` is
  **silent 248 times out of 248**. The gate is open; what fails is inside. The
  target it picks is the nearest residual/object candidate, which sits adjacent to
  the avatar, and no 6-cell step reduces a distance of 1.
- **sp80** -- learns only 2 of its 4 directions in a live run (against 4 from a
  fresh board) and marks five controls inert.

**"The nearest anomaly" is not the goal of any of these games.** That is what the
decode said for ls20 a session ago -- "needs a GOAL and a route" -- and the route
half is now demonstrably present and demonstrably insufficient. The goal half is
the open problem, and it is a perception question (what on this board is a
destination?), not a planning one.

Measured paired on the movement split (ls20, g50t, sp80, m0r0), **six seeds, both
arms**: the difference is **exactly 0.0000 on every seed**, not merely small. That
is the back-off construction showing up in the numbers -- the colour rule answers
first, so every transition it already explained behaves identically -- and the four
scoring games are all tagged pure `click`, where the planner's key path is never
reached at all. Kept under the three-part rule at the top of this file as a
correctness fix: `rigid_move`'s own docstring said "this is how an avatar announces
itself" while it could not see an avatar that translates perfectly rigidly. Note
the rule asks for 16 seeds and this has 6 -- defensible only because the difference
is an identity rather than a small number.

### The movement class, second pass: a destination that is HELD, and a route that searches

The section above closed the avatar question. What was left was the goal, and two
defects sat between the agent and any goal at all -- both found by instrumenting a
run rather than by reading the code.

**The destination was re-derived on every step.** `_route` took "the nearest
candidate" afresh each call, which is not a goal but a gradient that reverses the
moment the avatar moves, because the nearest anomaly is usually whatever the avatar
is standing beside. On ls20 seed 1: 249 route decisions, the position changed on
223 of them, and the whole run was spent oscillating between about six positions
with the key alternating k3/k4 -- opposite directions. Fixed by choosing a
destination once, holding it until reached or proven unreachable, and crossing it
off so the sweep moves on. "Proven unreachable" needs no constant: when every known
direction has been spent without the distance improving, greedy has nothing left.

**Greedy distance cannot go around a wall, and ls20 is a maze.** The avatar's
offsets define a lattice; a wall is an edge that does not exist, learned by trying
(a routed key that leaves the avatar in place marks `(anchor, key)` blocked).
Breadth-first over the known-good edges, untried ones assumed open, can step AWAY
from the target to get past something. No gradient can.

**Measured paired on the movement split, 16 seeds.** Read the counts, not the
aggregate:

| | HEAD | committed + search |
|---|---|---|
| ls20 takes level 1 | **2 of 16** | **8 of 16** |
| g50t takes level 1 | **0 of 16** | **2 of 16** |
| m0r0, sp80 | 0 | 0 |
| aggregate | 0.0084 | 0.1634 |

Paired difference **+0.1550, sd 0.4629, sem 0.1157**: it does NOT clear twice its
standard error, so no claim is made on the aggregate. 7 seeds up, 2 down, 7
unchanged. The counts are far better resolved than the mean -- 7 discordant seeds
one way against 1 the other, McNemar p ~= 0.04-0.07 -- because the aggregate is
dominated by whether a lucky seed happens to land near baseline.

**And the six-seed reading was +0.4107, against +0.1550 at sixteen.** Same code,
same arms; the first six happened to contain the 1.71 seed. This is the third time
this file has had to record that, so treat any n < 16 as a screen and nothing more.

What the score does when it lands is worth seeing, though: ls20 takes its level in
**23 actions against a baseline of 22** and g50t in **48 against 78** -- at and
under baseline, where the quadratic term pays most. g50t had never appeared in any
scoring list in any run before this change.

**Two destination criteria were closed by measurement here**, and neither is worth
retrying without new evidence:

- *Imagined compression.* Set the avatar down on each candidate in imagination and
  ask the drive. On ls20's opening board EVERY candidate scores negative -- an
  avatar standing in a corridor breaks the corridor's regularity -- and the real
  destination ranks tenth. The drive rewards a tidier board, not ARRIVAL, which is
  what ls20's decode said a session earlier and is now measured.
- *Walls as the first suspect.* The avatar was not pushing into a wall. It was
  oscillating, and the wall question only became real AFTER the destination was
  held still long enough to walk toward.

Still open, and now sharply: **what marks a destination.** The sweep works by
crossing candidates off rather than by recognising the goal, which is why m0r0 and
sp80 stay at zero and why ls20 fails half its seeds.

**CORRECTION, same session.** This section first said ls20's destination was a
five-cell plus of rare colours at (32,21), by analogy with su15's decode. That was
an assumption, never checked, and it is WRONG. Measured by replaying the frame
before the level cleared:

- **ls20** (seed 3) -- the avatar ends at rows 15-16, cols 34-38, INSIDE the framed
  box at rows 8-16 / cols 32-40 that this file had been calling a legend. The
  five-cell plus is still sitting untouched at rows 31-33 when the level clears.
- **g50t** (seed 3) -- the avatar, a 24-cell colour-9 block that started at rows
  8-12 / cols 14-18, ends at rows 50-54 / cols 37-41, hard against the left edge of
  the framed box at rows 48-56 / cols 42-50.

So the shared structure is not a marker figure, it is **an ENCLOSED REGION** -- a
ring of one colour around an interior of another, sitting apart from the corridor
network -- and the avatar goes into it or up to it. That is a much better candidate
for a general destination rule than "the nearest anomaly", and it is small: ls20
has two such regions on a 64x64 board (the goal box and a legend box in the corner)
against the 8-16 arbitrary candidates the sweep currently walks.

The lesson is the older one: an analogy between two games is a hypothesis, and this
file records measurements. Checking it cost one replay.

**And "enclosed region" is closed too, tested before it was built.** A frame is a
component whose bounding box holds cells unreachable from the box border without
crossing it. Run over the four movement boards:

| game | what it finds |
|---|---|
| ls20 | 4 regions -- the goal box IS among them (frame c5, rows 9-15, interior 6 colour-9 cells) but so are the corridor network and the budget strip |
| g50t | 3 regions, and **the goal is not one of them** -- its room at rows 48-56 / cols 42-50 is open on the left, so it is not a closed ring. One of the three is the avatar itself |
| sp80 | **none** |
| m0r0 | **none** |

Zero regions on exactly the two games that still score zero, and a miss on the one
whose goal we had just located. Not built.

So the destination rule is still unknown, with three candidates now closed by
measurement: imagined compression, the five-cell plus, and enclosure. What IS
established is where the avatar has to end up in two games, which is the evidence
any fourth candidate has to explain.


## The drive is silent for 80% of the decisions (traced 2026-08-30)

Run before building a backward credit pass, to see what a backward pass would
have to propagate. `ARC_TRACE=<path>` (off by default, read by nothing in the
decision loop) appends one JSON line per credited step: the token, the signed
one-step delta `d` the credit actually used, the resulting EMA, and the taboo
value; `board_replaced` writes a `reset` / `level_clear` event, so the run-up to
a cleared level is findable.

vc33, four seeds, 380 credited steps:

| seed | steps | `|d| <= 1` (only the move clock) | tokens ever credited positive | clears |
|---|---|---|---|---|
| 1 | 89 | 70 (78%) | 2 of 17 | 4 |
| 2 | 101 | 80 (79%) | 2 of 21 | 4 |
| 3 | 95 | 79 (83%) | 2 of 17 | 4 |
| 4 | 95 | 75 (78%) | 2 of 19 | 4 |
| **all** | **380** | **304 (80%)** | **2, every seed** | 16 |

Two facts, both reproduced on 4 of 4 seeds. **Four in five decisions are made on a
drive signal that carries nothing but the move-budget clock ticking one cell**, and
**exactly two tokens out of the seventeen-to-twenty-one on offer ever earn positive
credit** -- the same shape already measured on lp85 (16 candidates offered, 2 ever
live), now on vc33 as well. This is the candidate-reachability wall seen through a
different instrument, not a new problem.

A third fact is code, not statistics: **the action that clears a level is never
credited at all.** `board_replaced` drops `_last_token` so the next click is not
credited with a delta measured across the seam between two unrelated boards -- which
is right, that delta is meaningless -- but the consequence is that the single most
informative action of the episode teaches the policy nothing.

**What this does and does not say about a backward pass.** It does NOT refute one:
propagating future compression gain to the actions that preceded it is exactly the
mechanism for a signal that is invisible one step ahead, and 80% silence is that
condition. It DOES bound the payoff: with only two tokens ever earning positive
credit, a backward pass redistributes credit among candidates the policy already
ranks first, so the ordering may not move at all. Anything built here must be
diffed along a trajectory and measured paired over >=16 seeds before it is believed.

Traces are not committed; regenerate with
`ARC_TRACE=/tmp/t.jsonl $KIT/.venv/bin/python python/bench/bench.py --games vc33 --seed 1 --repeats 1`.

### ...and the 80% is vc33's number, not the architecture's (same night)

The section above was written from vc33 alone and framed as a general wall. Swept
over the other three scoring games the same night, it is not one. Two seeds each
for r11l/lp85/tn36, four for vc33; `|d| <= 1` means the board moved by at most one
cell's worth of MDL savings, i.e. the move clock and nothing else.

| game | steps | blind | live tokens / offered | median non-zero \|d\| | max | result |
|---|---|---|---|---|---|---|
| vc33 | 380 | **80%** | 2 / 23 | 44 | 108 | L2, x1.9 x1.2 |
| r11l | 294 | **49%** | 12 / 45 | 9 | 51 | L1, x4.1 / x1.9 |
| lp85 | 192 | **82%** | 3 / 34 | 7 | 25 | L1, x4.9 / x4.6 |
| tn36 | 235 | **71%** | 3 / 27 | 3 | 12 | L1, **x1.2 / x1.4** |

**The blind fraction does not order the games by how well they go.** tn36 has the
weakest compression signal of the four -- a median non-zero delta of 3 against
vc33's 44, a maximum of 12 against 108 -- and is the most efficient game on the
split, essentially at baseline. lp85 is the blindest and the worst. r11l is the
least blind, has six times as many live tokens as vc33, and lands in the middle.
Neither the fraction of silent decisions nor the size of the signal separates the
games that go well from the games that do not.

So "the drive is silent for four decisions in five" is a true description of vc33
and a real property of the credit signal, but it is **not the lever**, and the
sentence above calling it "the candidate-reachability wall through a different
instrument" over-generalised from one game. What it does establish: any mechanism
that consumes the one-step delta -- a backward pass included -- is working with
something that is absent most of the time on three of the four games and mostly
present on the fourth, so it must be measured per game, not on the aggregate.

Caveat on the table: two seeds for three of the games, and the magnitudes are not
comparable across games in any deep sense -- they depend on board size and how many
cells a control moves. The blind fraction is comparable (it is an absolute
one-cell threshold); the medians are context.

### What the learned value is actually worth against the rank prior (2026-08-30)

The open list has said since 2026-08-27 that `residual_bonus / (i + 1)` "makes rank
the policy's PRIOR, not a tie-breaker". That was an argument from reading the
formula. Here it is in traced numbers, which turn out to be worse than the sentence.

With the shipped constants -- `drive_gain_weight = 0.02`, `residual_bonus = 0.15`,
8 candidates -- the rank prior pays **0.150** to the first candidate and **0.019** to
the eighth, while everything the policy has learned about a control enters as
`0.02 x EMA`.

| game | median \|EMA\| | its contribution | best contribution | steps where learning beats rank 1 |
|---|---|---|---|---|
| vc33 | 0.6 | 0.013 | 1.693 | **21%** |
| r11l | 1.2 | 0.024 | 0.534 | 17% |
| lp85 | 0.0 | 0.000 | 0.268 | **4%** |
| tn36 | 0.6 | 0.013 | 0.188 | **3%** |

**At the median, everything the policy has learned about a control is worth about
as much as being ranked eighth** (0.013 against 0.019), and about a tenth of being
ranked first. On tn36 and lp85, learning is strong enough to outvote the top rank
prior on 3-4% of steps. The policy is, on those two games, following candidate
order almost entirely -- which is also the simplest explanation for the re-pick
rates measured the same night (a control already seen to be live is chosen 7% of
the time on lp85, 20-29% elsewhere).

**The cause is that `drive_gain_weight` is not scale-free.** The size of a
compression delta is a property of the game -- median non-zero `|d|` is 44 on vc33
and 3 on tn36, a 15x spread -- while the prior it competes against is a fixed
constant. One weight therefore cannot mean the same thing on two games, and 0.02
was hand-calibrated against vc33's own numbers (see the provenance correction in
CLAUDE.md). It makes learning decisive exactly where it was fitted and nearly
irrelevant on the other three.

**A hypothesis this suggests, NOT built and NOT measured:** scale the learned term
by the game's own running `|d|` so it is comparable to the fixed prior everywhere.
It does not touch the candidate set or its order, so it is not one of the three
re-ordering attempts already rejected. What would kill it: vc33's 21% is precisely
what carries it to level 2, and a normalisation that costs vc33 more than it buys
lp85/tn36 is worse than nothing. Measure paired over >=16 seeds against
`runs/base/` before believing anything.

**A non-finding, recorded so it is not mistaken for one.** The traced `dead` values
peak around 1.4, far above every other term in the sum -- but this does NOT refute
the fixed-point argument that the taboo can never accumulate. The trace writes
`dead` immediately after the press that incremented it, not when that control is
scored again ten steps later, which is the moment the argument is about. Testing
that claim needs `dead` recorded at candidate-scoring time; the candidate logging
added alongside this makes it possible, and it has not been done.

### Rejected #7: putting the learned value and the rank prior in the same units

The measurement that motivated it is real and stands (see the two sections above):
the median learned value of a control contributes 0.013 to its score against 0.150
for being ranked first, because `drive_gain_weight` is a fixed constant while the
size of a compression delta is a property of the game (median non-zero `|d|` is 44
on vc33, 3 on tn36). And the live control is demonstrably ON THE TABLE and passed
over -- offered on 100% of tn36's steps and taken on 22%, offered on 82% of lp85's
and taken on 8%.

The fix that follows: score a control by `residual_bonus * (EMA / running mean
non-zero |d|)`, so a control whose effect is typical for its game ties with being
ranked first. No new constant; the two priors are simply put in the same units.
Verified neutral with the flag off (seed 1: 2.2953 against the baseline's 2.2953,
bit for bit), so the A/B measures the change and not the implementation.

**16 paired seeds: mean -0.1653, sd 0.8321, sem 0.2080 -- NOT MEASURABLE, the seeds
that moved disagree on the sign.** Per game:

| game | paired diff | shape |
|---|---|---|
| tn36 | **-0.455** | never once positive beyond +0.06; two seeds at -2.11 |
| vc33 | -0.140 | swings from +4.63 to -7.28 |
| lp85 | -0.048 | **15 of 16 seeds exactly 0.00** |
| r11l | -0.019 | mixed, +3.46 to -3.57 |

**Read the per-game column, not the aggregate: it failed on precisely the two games
it was designed for.** tn36 -- where the live control is offered every single step
and the learned term was invisible against the prior -- got worse and never better.
lp85 did not move at all on 15 of 16 seeds, so re-weighting never changed which
action was taken there. Whatever makes those two games choose badly, it is not the
ratio between the learned term and the rank prior, even though that ratio is
genuinely lopsided.

What this does NOT license: concluding the 0.013-vs-0.150 imbalance is harmless.
It is measured, and it is real. What is now also measured is that correcting it,
alone, buys nothing -- so the next attempt has to explain why an agent that CAN see
the live control and CAN be made to value it still does not take it. The
epsilon-exploration (0.2) and the taboo term, which is an order of magnitude larger
than either prior, are the two unexamined terms in that sum.

Mechanism reverted; the trace and candidate logging that produced these numbers
stay, and are proven behaviour-neutral over 16 seeds.

## The four movement boards, read directly (2026-08-30)

The first three destination criteria died because they were analogies. This is the
opening board of each of the four movement games, dumped and read. Facts first.

**ls20** -- a maze on background 4. Bottom-left, OUTSIDE the play area (rows 53-62,
cols 1-10), sits a 5-framed box containing a colour-9 glyph at 2x scale. The goal
box (rows 8-16, cols 32-40, where the avatar is known to end) contains a colour-9
glyph at 1x scale. Descaled and laid side by side:

    legend (rows 55-60)     goal box (rows 11-13)
        # # #                   # # #
        # . .                   . . #
        # . #                   # . #

They are **exact horizontal mirrors of each other**. Also present: a two-colour
stack, colour 12 over colour 9, at rows 45-49 cols 34-38.

**g50t** -- a maze on background 0. Top-left, outside the maze (rows 1-5, cols 1-7),
a three-part legend: a 9-ring `###/#.#/###`, a solid 1-square `###/###/###`, and a
9-bar `###`. In play: a solid 5x5 colour-9 block with one hole (rows 8-12) and the
goal region (rows 49-55, cols 43-49) which is a 9-bracket -- top, right and bottom
closed, **open on the left**, with a single 9 cell at (52,46) inside it. A colour-8
line runs from (9,39) down col 39 to row 40, then left along row 40 to col 14,
ending in a left-pointing arrowhead.

**sp80** -- background 12. Top of the board, rows 1-3 and 4-7 at cols 36-39: a
colour-4 block stacked directly on a colour-6 block -- the same two-colour stack
shape ls20 has. In play: a colour-9 platform (rows 16-19, cols 12-31) and two
identical colour-11 pedestals, each a solid base with two legs, at cols 16-27 and
cols 40-51.

**m0r0** -- the board is split into a colour-11 left half and a colour-12 right
half, and **the right half is the exact mirror of the left about column 31** (x ->
62-x): every 5-coloured corridor maps over, and the two colour-10 blocks at rows
49-53 sit at cols 19-23 and cols 39-43, which are exact mirror images.

### The fourth hypothesis, and what it still owes

**The destination is fixed by a CORRESPONDENCE between a reference structure and a
region of the play area** -- the same Identity / ColourSwap / Reflect relation the
repo already implements in `correspondence.py` for the relational click games.

Status, stated honestly:
- **ls20: verified exactly.** Legend glyph and goal glyph are reflections, and the
  legend sits outside the play area where it cannot be walked to.
- **m0r0: structurally supported.** The reference is not a legend box but the
  board's other half, and the relation is again reflection.
- **g50t: not verified.** There IS a legend outside the maze, but the goal bracket
  is not an exact correspondent of the 9-ring -- it is open on one side and holds a
  dot. Suggestive, not established.
- **sp80: not verified.** It has the legend-shaped two-colour stack ls20 also has,
  but no level has ever been cleared there, so there is no known goal to check
  against.

**It is therefore NOT ready to build**, by the rule that killed the third candidate:
a criterion must yield the known goal where one is known and yield SOMETHING on the
two games at zero, and this one is two for four. What makes it worth the next hour
rather than discarding: it is the first candidate derived from the boards instead of
by analogy, and it points at a primitive that already exists.

One thing it needs that is already a known gap: **ls20's legend is at 2x scale**, and
Correspondence V1 deliberately excludes scale -- which is also exactly what blocks
s5i5 (its two template boxes are different sizes). The same missing capability sits
in front of both the movement class and the relational click class.

### Rejected #8: more search time inside the episode

The claim under test was the natural one -- the mechanism explores, it just needs
more room. It is the cleanest thing to test on this harness, so it was tested
rather than argued about.

Precondition checked first, because without it the experiment measures nothing: is
the action cap actually binding? At `--max-steps 250` every game ends at exactly 251
actions with `state=NOT_FINISHED` on **64 of 64 game-seeds** -- the agent never
finishes on its own, it is always cut off. So more budget is genuinely more search.

Doubled to `--max-steps 500`, 16 paired seeds, same seeds as `runs/base/`:

| | 250 actions | 500 actions |
|---|---|---|
| actions actually taken, every game | 251 | **501** |
| lp85 / r11l / tn36 levels | 1 on every seed | 1 on every seed |
| vc33 levels | 2 on every seed | 2 on every seed |
| total levels over 16 seeds | 80 | **80** |
| paired score difference | | **+0.0000, max abs 0.0000** |

**16,000 extra actions bought zero extra levels.** The score is not merely
statistically unchanged, it is bit-identical on every seed, because each level is
completed at the same action count and everything after that contributes nothing.

**Scope, stated so this is not over-read.** What is refuted is *more search time
within an episode*. Three other things called "scale" are untouched by it:
per-decision compute (deeper planning or imagination), accumulation *across*
episodes, and model size (there is no model to grow -- this agent is not a network).
The second is the interesting one and is now the only untested form: `drive_gain`
and `succ` are built fresh each run and discarded at the end, so the agent starts
every episode amnesiac, and nothing has ever measured what carrying them across
would do.

The inner loop was parallelised in the same commit: seeds are independent processes
and were being run one at a time on a 12-core machine, so 16 seeds took ~50 minutes
with eleven cores idle. `make quick-n` now runs `nproc - 1` workers -- about 15
minutes, one core left free.

### Rejected #9: lifting the permanent write-off (and what it clarified)

A human playing g50t cleared level 1 with the **spacebar** -- a key `decode.py`
had reported dead, because following each control for 8 presses from a fresh board
shows it changing nothing. It changes nothing *until the avatar has arrived*. The
decode method is structurally blind to context-dependent controls, and so is the
agent.

Two facts established from the code and a trace, independent of any fix:

- `RunPlanner.inert` is added to in exactly one place and **cleared nowhere** --
  not by `board_replaced`, which clears `profiles`, `ran`, `_dest`, `crossed_off`
  and `blocked` but not this. One clock-only press writes a control off for the
  rest of the episode, across every level. `Policy.dead` at least decays.
- Instrumented on seed 1, it fires on everything: **g50t writes off all five of its
  actions**, ls20 all four, m0r0 all five keys plus nine click controls, sp80 one
  key plus eight clicks. On a five-action game the agent permanently disables its
  entire repertoire, `k5` -- the winning key -- included.

The fix tested: when any action actually changes the board, the knowledge "this
control does nothing" is stale, so clear the write-offs. No new constant. Verified
neutral with the flag off (seed 1 bit-identical to the baseline).

**It changed no levels at all**, 16 paired seeds on the four keyboard games:

| game | levels, off | levels, on |
|---|---|---|
| g50t | 2 of 16 seeds | 2 of 16 |
| ls20 | 8 of 16 | 8 of 16 |
| sp80 | 0 | 0 |
| m0r0 | 0 | 0 |
| total | 10 | **10** |

and it cost the games that work: on the scoring split, lp85 **-0.284 with all 16
seeds negative** and vc33 **-0.277, all 16 negative**. `inert` is load-bearing
where the agent scores, exactly as the Rejected #6 note warned about suppression.

**What it clarifies is worth more than the fix would have been.** The write-off is a
real defect and it does disable the winning key -- but removing it wins nothing,
so the write-off is NOT what stops the agent on g50t. Un-writing-off space only lets
it press space uselessly more often. The human who cleared the level did not succeed
by pressing space more; they succeeded by knowing **where to stand first**. The
binding constraint is upstream, and it is the destination rule -- open item 2, and
now with one more independent argument pointing at it.

The `inert` write-off counter stays in the trace (ARC_TRACE only, behaviour-neutral),
because it is how this was found.

## Is POSITION the missing state variable? Partly -- checked before building (2026-08-31)

Three open items appeared to want the same thing: item 1 (stateful mode) needs
"where the last small change happened", item 2 (movement) needs "where to go", and
g50t's spacebar needs "where this control comes alive". So: log where the avatar
was each time a control fired, and what the control did, and ask whether
conditioning on position turns "this control does nothing" into a *function*.

`ARC_TRACE` now also records, at the moment `RunPlanner` judges a press,
`{tok, pos, nothing}` -- the control, the avatar's centroid on the previous board,
and whether only the move clock ticked. Behaviour-neutral; nothing reads it.

First: the outcome does vary with position. Every key on g50t and ls20 shows both
outcomes across 10-33 distinct positions. That is necessary, not sufficient.

The decisive test is whether `(control, position)` **determines** the outcome.
Counting only repeated pairs, and not gluing observations across a board change:

| game | repeated (tok,pos) pairs | contradictory | |
|---|---|---|---|
| **ls20** | 58 | **1** | position determines the outcome |
| g50t | 43 | 10 (23%) | helps, insufficient |
| **m0r0** | 50 | **40 (80%)** | position is nearly useless |
| sp80 | **1** | 0 | no data to condition on at all |

**So "the three classes want one variable, position" is refuted as stated**, and it
was refuted before the state key was touched. It would have produced a nearly
deterministic model on ls20 and nothing on m0r0.

Two diagnoses, both visible in the boards:

- **m0r0 has TWO movable bodies.** Its board is a colour-11 half and a colour-12
  half that are exact mirrors about column 31, with a colour-10 block in each at
  rows 49-53, cols 19-23 and 39-43. The avatar detector returns whichever body
  moved, so a single centroid means a different object from step to step. That is
  not one variable recorded badly, it is two variables collapsed into one.
- **sp80 fails differently and exactly as predicted**: one repeated pair in a whole
  run. The agent so rarely returns to the same place that there is nothing to
  condition on. Its bottleneck is reach, not representation.

The sharpened hypothesis, not yet tested: the missing variable is not position but
**the configuration of the movable bodies** -- one position on ls20, two on m0r0.
The test is to condition m0r0 on both colour-10 blocks and see whether 80%
contradictions fall.

### ...and the configuration of bodies is worse, not better

The sharpened version -- record the movable bodies SEPARATELY instead of one
blended centroid -- was tested and is refuted:

| game | bodies found | one centroid | body configuration |
|---|---|---|---|
| m0r0 | 2 | 50 repeats, 40 contradictory (80%) | 48 repeats, 42 (**87%**) |
| g50t | 5-6 | 43 repeats, 10 (23%) | 13 repeats, 11 (**84%**) |
| ls20 | 1 | 58 repeats, 1 (1%) | unchanged |
| sp80 | 1 | 1 repeat | unchanged |

A finer key partitions the same data into more cells, so it loses repeats (g50t
43 -> 13) without buying truth. This is the reach/truth trade already documented in
Rejected #6, arriving from the other direction.

**One guess of mine along the way was wrong and is corrected here.** Seeing 5-6
colour-9 components on g50t (legend, legend bar, goal bracket, the dot inside it,
the budget strip) I suspected the avatar centroid was polluted and the 23% was a
measurement artifact. It is not: logging the size of the located avatar shows **24
cells on 246 of 248 steps**, with the shape known. g50t's avatar was located
cleanly and 23% is a real property of the game.

**And the obvious explanation for it does not hold either.** If the spacebar
toggles a latent mode, the same control at the same position would act differently
before and after a press of it. Of the 10 contradictory (control, position) pairs,
4 have a `k5` press in between -- but `k5` is 18% of all presses and the spans are
several presses long, so chance alone predicts about that many. No evidence, and
n=10 is too small to carry any anyway.

**Where this leaves the state question.** Position determines the outcome on ls20
(1 contradiction in 58) and does not on g50t (23%) or m0r0 (80%), with the
detection verified clean. So the missing variable is not a geometric summary of
where things are -- neither one position, nor all of them. What distinguishes ls20
from the other two is not yet known, and no candidate here survived. Note that
m0r0 and sp80 are already classified as stateful-mode (open item 1) and ls20 is the
one pure movement game of the four, which is the shape of a real split, but that is
an observation and not a measurement.

## The g50t oracle: the destination cannot be walked to (2026-08-31)

Three destination criteria, a state-key hypothesis and a body-configuration
hypothesis were all guessed and all refuted. So instead of guessing again, this is
`python/bench/oracle_g50t.py`: an agent hand-given everything about g50t -- which
component is the avatar, where the goal is, and permission to read walkability off
the board -- which declares every fact it consults. A diagnostic, not a solver.

It learned the controls correctly (`ACTION2 (6,0)`, `ACTION1 (-6,0)`,
`ACTION4 (0,6)`, `ACTION3 (0,-6)`; 6-cell steps, a 5x5 body), routed itself down
the left corridor, and **stopped dead at (34,16)**: 122 consecutive presses of a
move the board says is legal, every one refused by the game.

Checked offline against the dumped board, BFS over the avatar's own 6-cell lattice
with a 5x5 footprint:

| walkable colours | result |
|---|---|
| `{5, 9}` corridor + own colour | **goal UNREACHABLE**, limit row 34 |
| `{5, 8, 9}` also the colour-8 barrier | goal reachable in 12 moves -- **on paper only** |

Row 34 is exactly where the live oracle stopped. The colour-8 structure at rows
38-42 seals the corridor, and the engine refuses every move into it.

**So the goal region at rows 49-55 x cols 43-49 cannot be reached by moving.**
Something must change the board first. That is why the destination work has been
stuck: every criterion tried so far -- imagined compression, the five-cell plus,
enclosure, and the two-stage legend correspondence -- silently assumed the same
frame, *identify the target and walk to it*. On this game that frame is false
before any criterion is even applied.

It also re-reads the human result. A person cleared level 1 with the spacebar; the
natural conclusion was "the trigger is a key we write off". The oracle says the
harder part is elsewhere: the key cannot matter until the board has been altered
enough for the destination to exist as a place you can stand.

**What the oracle needed, by kind** (the vocabulary a general mechanism would have
to be able to express):

    identity_of_controlled_body            which component is me
    effect_of_key_on_my_position           key -> (drow, dcol)
    goal_region_membership                 am I there yet
    reachability_over_learned_offsets      route over my own lattice
    walkability_read_from_the_board        can my footprint stand here
    retry_a_control_that_was_dead_elsewhere

The first five the agent can already express in some form. The sixth it explicitly
cannot -- and a seventh, which the oracle never had and which is what it actually
needed, is *terrain that can be changed*. Note the oracle FAILED, so this list is
what a winning agent needs at least, not what it needs in full.

## Carrying the mechanic across a game's runs: +1.21 (2026-08-31)

Open item 6 -- the only untested form of "scale" left after doubling the in-episode
budget bought exactly nothing. Every repeat of a game constructed a fresh agent, so
`drive_gain`, `succ` and the key->offset map were built from scratch and thrown away
three times per game, while the scorecard keeps a game's BEST of three.

What is carried is the MECHANIC and nothing else: `Policy.drive_gain`, `succ`,
`tries`, and `RunPlanner.moves` / `avatar` / `avatar_shape`, bound by reference so
later runs keep filling the same tables. Deliberately NOT carried: `_dest`,
`blocked`, `profiles` and `inert` -- those describe a level, not a mechanic, and
`inert` is the permanent write-off that disables g50t's entire repertoire.

Verified neutral with `ARC_CARRY=0` (seed 1 bit-identical to the baseline).

**16 paired seeds: +1.2137, sd 0.8541, sem 0.2135, 14 of 16 seeds positive.**
`seeds.py` still withholds REAL because two seeds disagree on the sign -- the same
profile as the control-name fix (+1.0967, sem 0.2653, 13 of 16), which stands as the
project's largest measured change.

Per level, median over 16 seeds:

| game | level | base | carried |
|---|---|---|---|
| vc33 | 1 | 13 actions (x1.93), score 27 | **3 actions (x0.43), score 115 = the cap** |
| vc33 | 2 | 24 (x1.33), score 56 | **16 (x0.92), score 114** |
| r11l | 1 | 43 (x1.95), score 26 | **21 (x0.98), score 103** |
| lp85 | 1 | x2.62, score 15 | x2.74, score 13 -- unmoved |
| tn36 | 1 | x0.81, score **115** | x1.27, score **62** -- worse |

**Three things follow, and the second one changes what to do next.**

**No new depth.** Levels are identical: 32 vs 32 on vc33, 16 vs 16 elsewhere. The
whole gain is efficiency, which the metric rewards quadratically, but the arithmetic
that says 10 needs six of vc33's seven levels is untouched.

**vc33's efficiency is now EXHAUSTED.** Both levels it reaches sit at 115 and 114
against a 115 cap. There is nothing left to win there by being faster, so every
further point on vc33 must come from level 3 -- where it currently spends 231
actions and fails.

**tn36 got worse, in exactly the way predicted before the run.** Its level 1 was
already faster than baseline (x0.81, capped) and carrying knowledge into run 2 slows
it to x1.27. It is the one game whose opening beat baseline, and the one game the
carry hurts. Not understood; a plausible read is that a value learned on one board
misleads on the next, which is the same shape as the `inert` failure. Open.

### Rejected #10: preferring a control not yet tried from this board

Traced first, built second -- and the trace turned out to answer the question on its
own. On vc33 level 3 the agent spends 73 steps in **3 distinct states** over 11
controls and repeats an already-tried (state, control) pair **60 times**. Splitting
those repeats by cause, from the replay alone:

| cause | measured |
|---|---|
| repetition the mechanic requires | 1 legitimate against 6 useless |
| repetition from FORGETTING | **0** |
| repetition from ranking | **60 of 60** |
| state collapsing hidden modes | 3 contradictory pairs of 11 (27%) |

Nothing is forgotten. At every repeat the taboo is not merely present, it is **large
-- median 0.750, max 1.312, above the 0.150 top-rank prior on 100% of repeats**. It
still fails to suppress, and the reason (inferred, not measured -- only the chosen
token's taboo is logged) is that every one of the 11 controls is pressed repeatedly,
so all of them carry a similar taboo, it cancels out of the comparison, and the
ordering falls back to the rank prior. **Uniform suppression is no suppression.**

The fix that follows: a bonus, worth `residual_bonus`, for a control not yet tried
from THIS board (full grid hash minus the learned move-clock strip). Unlike
`Policy.dead` and `RunPlanner.inert`, which punish a control's name globally, this
key clears itself whenever anything changes.

**16 paired seeds: +0.1382, sem 0.1620 -- not measurable. And on vc33, the game it
was built for, 14 of 16 seeds moved by exactly 0.0000.**

Re-traced with the flag on, the reason is not "it helped and the budget had nowhere
to go". **It never fired.** With the bonus on, vc33 level 3 is bit-identical: 73
steps, 3 states, 13 distinct pairs, 60 repeats. Thirteen distinct pairs are
exhausted in thirteen presses, after which `untried` is false for every candidate,
the bonus is uniform, and it cancels exactly like the taboo does.

**What this establishes about level 3, which is worth more than the fix would have
been.** Every term that could discriminate between controls -- the learned value,
the taboo, the untried bonus -- goes uniform within a dozen moves. The agent
exhausts its entire reachable (state, action) space in 13 actions and then has 217
actions with nothing new to try. Level 3 is therefore not bounded by wasteful
exploration; it is bounded by **reachability**: the control that would move the
board is not among the 11-13 the candidate generator can propose. That agrees with
the independent measurement earlier the same day, where vc33 was the only one of
three games with a real reachability component (the live control on the table on
just 43% of steps).

Mechanism reverted, seed 1 bit-identical to the baseline afterwards. The
clock-masked state hash stays in the trace -- it is what made this measurable.
