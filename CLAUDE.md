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
- **`cmd/probe/main.go`** — free-API diagnostic (`GAME=<id>` env): renders a grid, click-sweeps to find interactive cells, reports per-primitive deltas. This is how we reverse-engineer a game's mechanic. Throwaway/experimental.

## Status (2026-08-22)
- **FIRST live ARC-AGI-3 wins by pure intrinsic drive** — L1 of **vc33, lp85, r11l** (3 of 7 pure-click games), no per-game tuning, competitive with or beating baseline. The mechanism **generalizes** — that is the real result.
- **Correspondence** primitive is built + offline-proven (6 tests) but does **not yet** crack the 4 hard games (**s5i5, tn36, su15, lf52**). Those are RELATIONAL (template/legend matching) — anti-compression to self-regularity.
- **Two walls found on the hard games:**
  1. **Aggregation (general, deeper):** `DrivePreference = max` lets the big background's `Translate` savings *hide* any local relational signal (live: `best=Translate` for all actions). This gates *every* local-signal primitive.
  2. **Scale (s5i5-specific):** its two template boxes are different sizes; Correspondence V1 deliberately excludes scale.

## Immediate next task
**Fix aggregation first** (it gates everything). Leading candidate: in the model-free reinforcement, reward the **best-improving per-primitive delta** (max over per-primitive Δsavings) instead of the `DrivePreference`(max) delta — so a click that improves *any* compression axis (incl. Correspondence) gets credit even when a big-background primitive dominates the absolute max. Re-test s5i5/tn36, then add **scale** to Correspondence.

## Deep running log
Full history, decisions, and the honest-stakes stance live in the auto-memory:
`$HOME/.claude/projects/-home-user-extra-git-Protaxon/memory/project_protaxon_resume.md` (read it at session start). Note: Protaxon is the *older* line; **AlphaARC is canonical** — do not sync from it.
