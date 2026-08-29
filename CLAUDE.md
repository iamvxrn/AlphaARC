# AlphaARC — context for Claude

A Go cognitive architecture that solves **ARC-AGI-3** (interactive games at three.arcprize.org) via an **intrinsic compression drive — no external reward**. The goal emerges as "the shortest description of the world." Aimed at a *meta-algorithm for AGI* (brain-principles-in-silicon), not a per-game solver. Module: `alphaarc`.

## Golden rules
- **NEVER `git push`** (the user pushes). I **do commit when asked**; end commit messages with `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- **Go toolchain lives at `/run/host/usr/lib/go/bin/go`** (plain `go` is not on PATH). Always build/test/vet before committing.
- **The ARC-AGI-3 API is FREE** — iterate freely, a run is ~2 min. To run games, load the key inline and **never print it**:
  `export ARC_API_KEY=$(grep '^ARC_API_KEY=' .env | cut -d= -f2-)`
  Then e.g. `go run ./cmd/alphaarc-play -game vc33-5430563c -actions 150 -maxblobs 20 &> play.log`. `.env` is gitignored + claudeignored — never expose or commit the key.
- **Honest calibration over hype.** 100% ARC is the unsolved AGI frontier — do NOT bank on it financially; keep the ARC dream as science. Scaffold reasoning Socratically. When pronouns are unknown, use they/them.

## The compression stack
- **Primitives (`pkg/macro/mdl.go` + `residual.go`, `correspondence.go`)**: structural MDL over objects/topology (NOT bytes). Each returns *savings* = bits a regularity explains. `BestPrimitive` = max-saving per grid; **no goal is hand-picked — it wins by bits.** Primitives: `Reflect`, `Translate`, `Count`, `Correspondence` (relational: A = copy of B up to Identity/ColorSwap/Reflect + residual). Every primitive is guarded against its degenerate cheat (blank / flood / scatter / solid).
- **`pkg/macro/drive.go`** — `DrivePreference` / `DriveScore` = the intrinsic goal signal.
- **Residual = attention (`residual.go`)** — `ResidualTargets` returns the cells that BREAK the best regularity (the anomaly), which is usually the interactive control.
- **Live agent `cmd/alphaarc-play/main.go`** — perceive → residual/object click candidates → **MODEL-FREE reinforcement**: `driveGain[token]` = EMA of the *observed* compression delta per click (`driveGainWeight`). The affordance *model* can't learn big / absolute / same-colour-colliding button effects, so we reinforce on observed ΔL instead of predicting. Cortex pixel-repaint macros are **disabled** (a Floor-2 static-ARC tool, wrong for interactive games). On a cleared level, stagnation **persists** instead of rewinding the whole game.
- **`cmd/probe/main.go`** — free-API diagnostic (`GAME=<id>` env): renders a grid, click-sweeps to find interactive cells, reports per-primitive deltas (`STEP=4|8` sweep resolution) or exact-coordinate breakdowns (`POINTS=x,y;x,y`). This is how we reverse-engineer a game's mechanic. Throwaway/experimental.

## Status (2026-08-22, calibration correction same day — see below)
- **FIRST live ARC-AGI-3 wins by pure intrinsic drive** — L1 of **vc33, lp85, r11l** (3 of **6**, not 7 — su15/lf52 were miscategorized as pure-click; they have extra actions), competitive with or beating baseline.
- **Correction, same session**: the "no per-game tuning" claim overclaimed. Several core hyperparameters (`driveGainWeight=0.02`, `refractorySteps=4`, the maxChangeCells cascade guard, GAME_OVER/reset handling) were hand-calibrated by reading **vc33's own probe numbers** in an earlier session — not derived task-agnostically. A genuinely untouched 6th pure-click game, **ft09-0d8bbf25** (never probed/rendered before), was found and blind-tested cold with zero prior calibration: **0/150, L1 never completed**. So: the *primitives* (Reflect/Translate/Count/Correspondence) are honestly game-agnostic; the *action-selection hyperparameters* are currently fit to one game and have NOT been shown to transfer blind. Say "3 of 6, hyperparameters tuned against one of them" — not "generalizes."
- **Correspondence** primitive is built + offline-proven (6 tests) but does **not yet** crack the 4 hard games (**s5i5, tn36, su15, lf52**). Those are RELATIONAL (template/legend matching) — anti-compression to self-regularity.
- **Aggregation fix DONE (commits 606f770 + 4b2f6f3):** model-free reinforcement now scores each click by `macro.BestPrimitiveDelta` — the SIGNED delta of whichever primitive moved most by absolute value (not the old `DrivePreference`-of-`DrivePreference`, and not a plain max-of-deltas either — that has its own floor-at-zero bug, see `TestAggregation_WorseningClickIsPunishedNotFloored`). Regression-tested live: vc33 solves L1 in ~9 actions again, matching pre-fix speed. This wall is CLOSED for the general mechanism, even though it didn't (yet) crack the hard games itself.
- **tn36's wall is NOT aggregation** (new finding): a fine probe sweep (STEP=4, 256 points) found `bestCorrespondenceDelta=+0` everywhere, even the *unmasked* direct Correspondence delta. Correspondence (built for same-shape template *pairs*) likely just doesn't recognize tn36's legend-to-checkerboard relationship at all — a representational gap, not a masking one.
- **Scale (s5i5-specific, still open):** its two template boxes are different sizes; Correspondence V1 deliberately excludes scale.

## Score measurement (Phase 0, 2026-08-25) — READ BEFORE OPTIMISING ANYTHING
The leaderboard number is NOT "levels taken". From the installed engine
(`arc_agi/scorecard.py`): `level_score = min(115,(baseline/actions)^2*100)` if the
level was completed else 0; `game_score = Σ(level_score_i · i) / Σ(i over ALL the
game's levels)`; total = mean over games, best run per game. Per-level
`baseline_actions` and a mechanic `tags` field are handed to us in each game's
`environment_files/<game>/<ver>/metadata.json`.
Consequences (computed over the real 25 public games):
- **Depth dominates**: taking L1 of *every* game at exact baseline = **3.52**, not 10.
  >10 needs the first two levels everywhere (10.6) or ~3 games solved end to end.
- **Efficiency is quadratic** and **baseline is the ceiling** (a per-game cap
  forbids earning more by being faster). Target baseline, don't race it.
- L1 has the LOWEST weight -> it is the cheapest place to spend a budget learning
  the mechanic, and levels 2+ repay it. The metric rewards the MBRL plan directly;
  the world model must survive a level transition.
- Tags: `keyboard_click` 13, `click` 7, `keyboard` 4 -> **17 of 25 games need
  non-click actions**, and every win we have is in the 7-game click bucket.
Harness: `make bench` / `bench-all` / `holdout` (see `python/bench/README.md`).
`python/alphaarc/` is now CANONICAL and `agent/my_agent.py` is a BUILD PRODUCT
(`python/bench/bundle.py`; `make check-bundle` fails when stale).
`python/bench/splits.json` freezes 8 never-probed games as a holdout — running it
requires `ARC_HOLDOUT_OK=1`; it is for measuring, never for tuning.

## Session 2026-08-25 result: 0.21 -> 0.40 on train (+ ls20)
Built and CONFIRMED: `make quick` (127 s inner loop over the 4 scoring games,
predicts the full split to the percent), `make decode GAME=xx` (follow every
control AND key for N presses, print the compression profile, flag the ones that
dip before they climb), and the **HybridPolicy** default -- one-step credit for
the opening, run-planning once the level proves long (25 actions) or the clicks
prove dead (5 in a row). Keys are controls too, and `rigid_move` + the residual
give movement games a goal; **ls20 scored for the first time**.
Seven earlier attempts at improving one-step credit returned zero or worse and are
catalogued in `python/bench/README.md` "Tried and rejected" -- read it before
proposing an eighth. The through-line behind everything that DID pay: reward is
routinely invisible one step ahead. Game-by-game facts: memory
`reference_decoded_games`.
**THE HOLDOUT IS SEALED** (user's ruling, 2026-08-25, at train 0.4010). Not
"untouched by accident" — deliberately locked. Running it now would measure our
MISSING MECHANICS, not our generalization: four movement games at zero and twelve
of seventeen never decoded would drag it to ~0.15 and tell us nothing about
whether the planner transfers. It is the only overfitting detector we get at the
end, and peeking at 0.40/10 would start shaping heuristics around what we saw.
**Unlock only when the obvious structural bugs on TRAIN are exhausted and every
control class is closed.** "It is only one look" is exactly the argument the rule
exists to refuse.

## Immediate next task (handoff, end of 2026-08-25)
Everything below is committed; tree clean, all suites green, `make check-bundle`
up to date, no background jobs.

**Where the score is:** quick = **2.6091 over 16 seeds** (HEAD, `runs/peerrank/`),
from 1.5124 -- the peer-rank control name (+1.0967, sem 0.2653) carrying the level
seam, on top of 2026-08-26's move-budget fix (+0.199), hand-over sweep (+0.067) and
known-dead override (+0.146, unresolved). **16 seeds, not 8**: the known-dead change
read +0.287 over seeds 1-8 and +0.146 over 1-16, and the peer-rank one read +0.787
over 1-8 against +1.097 over 1-16 -- eight seeds move a headline by 30-50% in EITHER
direction. NOT the 1.7087 once quoted; that was one lucky seed. Quote absolute
scores with their seed count. Target 10; the L1-everywhere ceiling is 3.52, so the
metric needs DEPTH.
**What the seam fix actually bought is EFFICIENCY AFTER THE SEAM, not new levels** --
levels reached are identical (r11l 1, tn36 1, vc33 2 on every seed, both arms); only
lp85 changed, from taking its level in 7 of 16 seeds to 16 of 16. Actions as a
multiple of baseline, on levels actually completed:

| level | base | peer-rank |
|---|---|---|
| vc33 **L1** (nothing to carry yet) | x1.83 | x1.75 |
| vc33 **L2** (the first level AFTER a seam) | x5.95 | **x2.27** |
| tn36 L1 | x1.58 | x1.02 |
| r11l L1 | x2.79 | x1.91 |
| lp85 L1 | x5.02, taken in 7 of 16 seeds | x2.74, taken in **16 of 16** |
The signature is the point: vc33's level 1 barely moves because there is nothing to
carry INTO it, while level 2 more than halves. That is what "the mechanic survived
the seam" looks like, and it is the quantity the metric weights by level index.
**`runs/base/` is CURRENT** -- regenerated from HEAD on 2026-08-30: 16 seeds, mean
2.6091, sd 0.6892, range 1.676-4.000, and IDENTICAL to `runs/peerrank/` on every one
of the 16 seeds. Diff new work against it directly; regenerate again when HEAD moves.

Use the PAIRED difference: it resolves ~0.03 for a change that touches little and
~0.3 for one that changes behaviour broadly. Add seeds until sem is small enough.

**And read the per-game levels, not only the aggregate.** A game is scored by its
BEST run of three, so a change that lifts the TYPICAL run barely moves the headline.
Today's two changes read as +0.27 on the aggregate and look like this per game over
8 seeds: vc33 reaches level 2 in `[2,2,2,2,2,2,2,2]` where it used to manage
`[2,1,2,2,1,1,1,2]`; lp85 takes its level in 6 of 8 seeds instead of 3, at x5.3
baseline instead of x13.0; tn36 x3.0 -> x1.6; r11l x4.2 -> x3.2. Reliability is the
precondition for depth, and depth is where the metric's weight is.

**Where the score actually hides, measured:** vc33 takes level 1 in 6 actions
against a baseline of 7 -- a CAPPED 115 points, nothing left to win there -- and
then spends 68 actions on level 2 against a baseline of 18. vc33 at baseline on
the two levels it ALREADY reaches would score 11.25 against its current 4.61,
which is more than the whole 17-game aggregate. Across the split, taking the same
levels we already take but at exact baseline is 0.40 -> 1.12. Efficiency on levels
already reached beats reaching new ones, because the efficiency term is quadratic
while a new deep level arrives heavily discounted by its own inefficiency.

**THE LEVEL SEAM IS NOW CARRIED, 2026-08-29 -- the largest measured change in the
project.** vc33's two colour-9 buttons sit at eighth (3,7) and (4,7) on level 1
and at (3,0) and (4,0) on level 2 -- same colour, same size bucket, same rows,
MIRRORED column -- so a name ending in position-in-eighths called them four
different controls and the mechanic was learned twice. What the mirroring does NOT
change is the pair's ORDER, so `control_signature` now ends in the object's RANK
among same-colour, same-size-bucket peers in reading order instead of its position.
Measured paired, 16 seeds: quick **1.5124 -> 2.6091, +1.0967, sem 0.2653** (t=4.1,
13 of 16 seeds positive; 3 disagree on sign, so `seeds.py` still withholds REAL and
no stronger claim is made). lp85 agrees on all 16. Pinned in
`test_the_name_survives_the_level_seam_that_broke_vc33`. The earlier "+0.0095 +/-
0.0072, neutral" refers to a DIFFERENT attempt and no longer describes the seam.

**THE MEASUREMENTS BEHIND EVERY PAST CLAIM WERE INVALID.** Measured 2026-08-26:
same code, same `--repeats 3`, four seeds give quick = 1.7087 / 1.0019 / 0.5205 /
2.9709 -- mean 1.5505, **sd 1.07, a 5.7x spread**, because the engine scores a game
by its best run and vc33 reaching level 2 is nearly a coin flip. The "+27%" headline
and all five "Tried and rejected" entries sit inside that band. Only ONE result is
outside it (ranking candidates by shape rarity: every game to zero levels in every
repeat). Treat the rest as UNMEASURED, not settled.

**The loop:** `make decode GAME=xx` -> understand -> build exactly what the decode
specified -> `make quick-n SEEDS=4 TAG=x VSDIR=base` (~10 min, PAIRED on identical
seeds) -> confirm on `make bench`. `make quick` is a single seed: a smoke test,
never a measurement. `python/bench/seeds.py` calls a change REAL only when the mean
paired difference clears twice its standard error and every seed agrees on the sign.
`python/bench/runs/base/` is HEAD's 16-seed baseline -- regenerate when HEAD moves.

**Decoded: 17 of 17 train games** (lf52, s5i5, su15 finished 2026-08-26) — see
`python/bench/README.md` and memory `reference_decoded_games`. Nothing on train is
left to decode; the holdout stays sealed.

**Open, in rough order of expected value:**
1. `stateful-mode` (dc22, m0r0, re86, sp80) is blocked by the STATE REPRESENTATION,
   not the credit: dc22's mode is a cursor position that moves 9 cells of 4096, and
   the per-primitive level vector cannot see it. The class needs a state that also
   carries WHERE the last small change happened.
2. **Movement: what MARKS a destination.** Narrowed twice on 2026-08-29. The
   avatar detector was never the blocker (fixed, measured 0.0000). What was
   broken beneath it: the destination was re-derived every step, so it was a
   gradient that reversed as the avatar moved -- ls20 spent 249 route decisions
   oscillating between six positions on opposite keys -- and greedy distance
   cannot round a wall, which a maze needs. Both fixed: a destination is held
   until reached or proven unreachable then crossed off, and the route is a BFS
   over the lattice the avatar's own offsets define, with walls learned as edges
   that do not exist. Result, 16 paired seeds: **ls20 takes L1 in 8 of 16 seeds
   (was 2), g50t in 2 of 16 (was 0, and it had never scored in any run)**, at 23
   actions against a baseline of 22 and 48 against 78. The aggregate difference
   is +0.1550, sem 0.1157 -- NOT resolved, and it read +0.41 over the first six
   seeds, so quote the counts, not the mean.
   **What is still missing is the goal itself, and there is NO CANDIDATE RULE
   right now.** The sweep works by crossing candidates off, not by recognising a
   destination, which is why m0r0 and sp80 stay at zero and ls20 fails half its
   seeds. "Nearest" is what caused the oscillation (fixed by holding a destination
   until reached or crossed off). Three ways of MARKING the destination are now
   closed by measurement, and none of them is the rule:
     - **imagined compression** -- scores EVERY candidate negative on ls20; an
       avatar standing in a corridor breaks the corridor.
     - **a five-cell plus** (an earlier claim in this file that ls20's goal was the
       plus at (32,21)) -- an unchecked analogy with su15, refuted by replay: that
       plus is untouched when the level clears.
     - **an enclosed region** -- a component whose bounding box holds cells
       unreachable from the border without crossing it. ls20 4 regions (the goal
       among them), g50t 3 (the goal NOT among them -- its room is open on the
       left, so no closed ring), sp80 none, m0r0 none. Zero regions on exactly the
       two games that score zero, and a miss on the one game whose goal had just
       been located. Run before building, so it was never built.
   **What a fourth hypothesis has to explain** (measured by replaying the frame
   before each level cleared): ls20's avatar ends INSIDE the framed box at rows
   8-16/cols 32-40 (which this repo had been calling a legend), and g50t's ends
   flush against the LEFT edge of an OPEN-SIDED room at rows 48-56/cols 42-50.
   Test the next criterion against those two boards -- and against sp80 and m0r0
   yielding something -- BEFORE building it; that is what killed the third one in
   five minutes. Details in `python/bench/README.md`.
3. Depth on the four scoring games; vc33 still reaches level 2 of 7 on every seed,
   and the seam fix did not change that -- it made the levels already reached much
   cheaper (vc33 L2 x5.95 -> x2.27). Remaining efficiency headroom, mean over 16
   seeds: lp85 L1 x2.74, vc33 L2 x2.27, r11l L1 x1.91, vc33 L1 x1.75, tn36 L1 x1.02
   (that last one is at baseline -- capped, nothing left to win there). Depth past
   vc33 level 2 is now the only large untried lever on this split.
4. **Dead clicks were 58% of the budget** (partly addressed; see below).** Measured on
   vc33 level 2, seed 7: 52 of 90 transitions do nothing but tick the move clock,
   with the detector working correctly. `Policy.dead` decays x0.75 a step, so it
   fades in ~10 while 8-16 candidates cycle on exactly that period -- the fixed
   point is v = (v+1)*0.75**10 ~ 0.06, i.e. no suppression can ever accumulate.
   **Slowing the decay is measured NOT to be the fix** (0.92: +0.098 +/- 0.548;
   0.97: +0.009 +/- 0.490, vc33 swinging +3.66 to -5.52), because deadness is
   CONDITIONAL: vc33's scalar saturates at three presses, so the + button is dead
   at saturation and live again after a -. The place for this is `succ`, already
   keyed by (token, STATE). DONE, in part: the model turned out to be consulted
   successfully 18-25% of the time and to predict "nothing happens" on 84% of vc33's
   hits, and the policy was ignoring that -- now it does not (+0.146 over 16 seeds,
   unresolved). **The state key is now CLOSED, 2026-08-27** -- and the premise was
   wrong. Instrumented over every lookup the policy makes, the fine key answers
   ~50%, not 18-25% (vc33 50%, r11l 43%, lp85 84%, tn36 25% at seed 1); 25% was
   tn36 alone. It is a reach/truth TRADE, not a defect: coarsening buys reach and
   pays in the answer being true, the optimum differs per game, and `argmax` /
   `rank order` keys carry literally zero bits (they reproduce the state-free
   numbers exactly -- the dominant primitive never changes inside a game). Backing
   off to the state-free fact "this control has never once moved the board" reached
   more on 8 of 8 game-seeds offline and measured **-0.0336 +/- 0.0695 over 16
   seeds**: reverted, because the suppression is SELF-CONFIRMING (one dead press
   writes a control off, and being written off prevents the press that would clear
   it -- seed 7, vc33 -4.16, a lost level 2). Requiring two distinct states erases
   the whole gain. Full write-up: `python/bench/README.md` Rejected #6.
   **Independently replicated 2026-08-29** from a fresh trace and a fresh 16-seed
   run: the same -0.0336, and the same offline picture (fine key answers ~50%;
   `argmax`/`rank order` reproduce the state-free numbers exactly). Treat this item
   as CLOSED and read it before re-opening the state key -- the replication cost an
   hour that the line above had already paid for.
5. **Candidate REACHABILITY (real, but do not attack it by re-ordering).** In s5i5
   and su15 not one of the eight offered click candidates is live, measured; the
   real controls sit at rank 21+. Three ways of editing the candidate set have now
   been measured worse — spread, rarity, per-class cap (Rejected #5) — because
   `residual_bonus / (i + 1)` makes rank the policy's PRIOR, not a tie-breaker.
   Any new attempt needs a mechanism that reaches those controls WITHOUT changing
   the order of the ones already there, and must be diffed along a trajectory
   rather than on the opening frame. **Measured 2026-08-27, and it sharpens
   the class:** on lp85 the policy is offered 16 candidates per step while only 2
   tokens are ever live in the whole run, and 94% of transitions do nothing. A
   perfect dead-detector suppresses 14 of 16 and still chooses between the same
   two -- no world model reaches past a candidate-set limit.
   **SPLIT BY MEASUREMENT, 2026-08-30 -- reachability is NOT what binds lp85 or
   tn36.** The trace now logs the candidate set per step (`ARC_TRACE`), so "the
   policy never saw the live control" and "it saw it and walked past it" are finally
   separable. Counting only steps where the policy already knew a control was live:

   | game | live control WAS offered | and taken | offered, passed over |
   |---|---|---|---|
   | tn36 | **100%** | 22% | **77%** |
   | lp85 | 82% | 8% | **91%** |
   | vc33 | 43% | 51% | 48% |

   On tn36 the live control is on the table at every single step and is taken one
   time in five. That is item 4's territory, not item 5's. vc33's 43% is the only
   real reachability component among the three. This does not touch the s5i5/su15
   evidence above, which was never re-traced -- it says reachability is not the
   binding constraint on the two games that were blamed on it.
   **And the obvious repair does not work.** The learned value is worth 0.013 at
   the median against 0.150 for being ranked first, because `drive_gain_weight` is
   fixed while `|d|` is a property of the game (median 44 on vc33, 3 on tn36).
   Putting both in the same units -- `residual_bonus * EMA / running mean |d|`, no
   new constant -- measured **-0.1653, sem 0.2080 over 16 paired seeds, NOT
   MEASURABLE**, and it failed on exactly the two games it was built for: tn36
   -0.455 and never positive, lp85 unchanged on 15 of 16 seeds (Rejected #7).
   So an agent that CAN see the live control and CAN be made to value it still does
   not take it. The two unexamined terms in that sum are the epsilon-exploration
   (0.2 -- one action in five is uniform over the candidates) and the taboo, whose
   traced values reach 1.4 against priors of order 0.15.

**Do NOT touch the holdout** — sealed by the user, criterion above.
