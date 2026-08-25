# AlphaARC

Trying to solve [ARC-AGI-3](https://arcprize.org) without LLMs — a small, from-scratch system that chases compression, not reward. If a move makes the world simpler to describe, it’s worth doing.

**Status: early stage, in active development. This README describes what is measured, not what is hoped for.**

## What this is

Not a large language model, not a transformer. A tiny brain that tries to find the shortest description of what it sees:

- **Compression as drive** — structural MDL primitives (`Reflect`, `Translate`, `Count`, `Correspondence`) compete on bits saved. No hand-picked goal — whichever compresses the current grid most, wins.
- **Residual = attention** — cells that break the best regularity are where the interactive controls usually live.
- **Small graph + memory underneath** — sparse Hebbian graph with WTA inhibition, Modern Hopfield associative memory, and a homeostatic drive system. Offline sleep compresses recurring structure.
- **Hybrid policy on top** — cheap one-step credit most of the time, switches to lookahead when the level proves long or clicks go dead.
- **Measured properly** — a reproducible Kaggle-faithful harness (`python/bench/`) with a frozen holdout that we don’t tune against.

The hypothesis: generality comes from “the shortest description wins” plus a few small specialists, not from hand-coded puzzles. Still unproven — that’s what the bench is for.

## What's actually verified

Run it yourself — no demo videos:

```sh
go test -count=1 -v ./...   # Go unit tests
make quick                  # 4 scoring games, ~2 min inner loop
make bench                  # full train split, official Kaggle score
```

- **Memory & graph plumbing still holds** — Modern Hopfield 96.0% exact recall at 200 patterns (10× capacity), 3-hop spreading activation against hand-computed values, cohesion-gated abstraction compression with zero edge loss.
- **Grid perception works** — flood-fill blob detection and object segmentation into stable tokens on real 64×64 game grids.
- **Real ARC-AGI-3 client** — endpoint-for-endpoint against the official server, including two live bugs fixed (stale host, missing cookie jar for sticky sessions).
- **First live wins by intrinsic drive** — L1 of 3 pure-click games (vc33, r11l, lp85) solved with no external reward. Train aggregate ~0.40 (scoring subset ~1.71), up ~27% from the ~0.31 baseline. `python/bench/runs/hybrid_train.json` vs `succ_model.json`.
- **8 games decoded** — we know what each control *does* (`make decode GAME=vc33`), including why some need 2–3 presses in one direction before the payoff appears.

## What's still hard

Honestly, most of it.

- Only 3 of 6 pure-click games solve; the other 4 (s5i5, tn36, su15, lf52) are relational/legend-matching and need more than self-symmetry. A blind test on a 6th never-seen click game (ft09) was 0/150.
- Key hyperparameters were tuned against vc33’s probe numbers — not derived task-agnostically. Holdout (8 games) is still untouched; until it’s run, “generalizes” is just a word.
- 17 of 25 public games need keyboard actions; every win so far is click-only. Non-click actuation (Phase B.5/C) exists but hasn’t scored yet.

## Structure

```
pkg/graph        sparse graph, Hebbian plasticity, spreading activation, routing
pkg/memory       Modern Hopfield memory
pkg/homeostasis  Ashby-style drive system (energy, curiosity, stress, dopamine …)
pkg/mlp, agent   micro-MLPs and predictor/actor/associator agents
pkg/offline      sleep: Louvain clustering + abstraction compression
pkg/pipeline     predictive-cycle engine tying it together

pkg/macro        MDL primitives (Reflect/Translate/Count/Correspondence) + residual
pkg/feataff      feature-level affordance: which control moves which feature
pkg/goalsel      causal goal selection (what feature actually leads to reward)
pkg/actuate      control discovery & actuation
pkg/mbrl         world model stub + imagination
pkg/environment  ARC-AGI-3 interface, practice game, REST client, perception

python/bench     Kaggle-faithful measurement: make bench / quick / holdout / decode
python/alphaarc  canonical Python package (bundled to agent/my_agent.py for Kaggle)
```

## Running it

Requires Go 1.23+ and Python 3. No heavy dependencies, no API key needed for offline tests.

```sh
go test -count=1 -v ./...          # full Go suite
make test                          # Go + Python checks + bundle freshness
make quick                         # inner loop: 4 games that ever score
make bench                         # train split (needs KIT checkout, no key)
go run ./cmd/alphaarc-arc-smoke    # live check vs three.arcprize.org (needs ARC_API_KEY, see .env.example)
make decode GAME=vc33              # reverse-engineer one game's controls
```

## License

MIT-0 (no attribution required) — see [LICENSE](LICENSE). Chosen for [ARC Prize](https://arcprize.org) open-source eligibility.
