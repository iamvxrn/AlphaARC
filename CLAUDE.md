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

**Where the score is:** train 0.4010 (17 games) + ls20's first level; quick 1.7087
(4 games, all scoring). Target 10. The L1-everywhere ceiling is 3.52, so the metric
needs DEPTH, not more first levels.

**The loop that works:** `make decode GAME=xx` -> understand -> build exactly what
the decode specified -> `make quick` (127 s) -> confirm with `make bench` (~40 min).
Nine changes have been measured; every one that paid came out of a decode, and
every one that came from "this looks wrong, fix it" returned zero or worse. Read
`python/bench/README.md` "Tried and rejected" (four entries) before proposing
anything that widens exploration or edits the compression measurement.

**Decoded: 14 of 17 train games** — see `python/bench/README.md` and memory
`reference_decoded_games`. **Left to decode: lf52, s5i5, su15** (all click-only, so
run decode WITHOUT `--keys-only`).

**Open, in rough order of expected value:**
1. `stateful-mode` (dc22, m0r0, re86, sp80) is blocked by the STATE REPRESENTATION,
   not the credit: dc22's mode is a cursor position that moves 9 cells of 4096, and
   the per-primitive level vector cannot see it. The class needs a state that also
   carries WHERE the last small change happened.
2. Movement beyond ls20 — g50t, m0r0, re86, sp80 still zero.
3. Depth on the four scoring games; vc33 reaches level 2 of 7.

**Do NOT touch the holdout** — sealed by the user, criterion above.
