# AlphaARC

A graph-based cognitive architecture attempting the [ARC-AGI-3](https://arcprize.org) benchmark — Francois Chollet's interactive, novel-environment intelligence test.

**Status: early stage, in active development. No benchmark score has been submitted yet.** This README describes what is built and verified, not what is hoped for.

## What this is

Not a large language model, not a transformer. A small, from-scratch neuromorphic/graph system:

- A sparse concept graph with Hebbian (STDP-style) plasticity and Winner-Takes-All lateral inhibition
- A Modern Hopfield associative memory
- An Ashby-inspired homeostatic drive system (energy, curiosity, stress, dopamine, cortisol, serotonin)
- A dynamic competition router with eligibility traces for temporal credit assignment
- Offline "sleep" cycles that compress recurring co-activated structure into abstraction nodes (Louvain community detection + cohesion-gated merging)
- A Mixture-of-Experts pool of small predictor networks, one per discovered graph cluster
- A hierarchical goal stack and a conflict-resolution mechanism that keeps losing candidates as memory instead of discarding them

The working hypothesis: generality comes from many narrow specialists bound together by a router, and abstraction emerges from compressing repeated structure rather than being hand-designed in. Neither claim is proven — they're the thing this project is testing.

## What's actually verified

Every claim below is backed by a real, reproducible Go test — not a benchmark score, not a demo video. Run them yourself:

```sh
go test -count=1 -v ./...
```

- Modern Hopfield memory: 96.0% exact recall at 200 stored patterns (10x capacity), vs. 0% for classical Hopfield past 12 patterns.
- Spreading activation reaches 3 hops with hop-decay, verified against hand-computed exact activation values.
- Offline abstraction compression: a graph cluster that organically reaches sufficient cohesion (0.8059 observed, threshold 0.75) collapses into one abstraction node with zero loss of external connectivity (edge weights preserved as exact weighted averages).
- MoE specialist routing: distinct observation clusters correctly route to distinct, independently-seeded predictor networks; verified they produce genuinely different outputs, not clones.
- A real HTTP client for the ARC-AGI-3 REST API, endpoint-for-endpoint confirmed against the official server source rather than guessed, including two real infrastructure bugs found and fixed against the live service (a stale default host, and a missing cookie jar for the service's sticky-session routing).
- A grid-perception module: flood-fill color-region detection on real game grids, producing stable, graph-seedable observation tokens.

## What's not built yet

Honestly, the hard part. Perceiving a grid and connecting to the API is plumbing. Turning perceived structure into a correct action choice for an arbitrary, never-seen game is the actual unsolved problem this benchmark is designed to test, and it's where this project currently stops. There is no scored run yet.

## Structure

```
pkg/graph        sparse graph, Hebbian plasticity, spreading activation, routing
pkg/memory       Modern Hopfield associative memory
pkg/homeostasis  Ashby-style multi-hormone drive system
pkg/mlp          native micro-MLP with online backprop
pkg/agent        predictor / actor / associator agents built on pkg/mlp
pkg/offline      subconscious sleep cycles, Louvain clustering, abstraction compression
pkg/goals        hierarchical goal stack
pkg/conflict     conflict resolution that preserves losing candidates
pkg/pipeline     the predictive-cycle engine tying all of the above together
pkg/environment  ARC-AGI-3 interface, a local practice game, a real REST client,
                 and grid perception
```

## Running it

Requires Go 1.23+. No external dependencies.

```sh
go test -count=1 -v ./...             # full test suite
go run ./cmd/protaxon-arc-smoke       # live connectivity check against the real
                                       # ARC-AGI-3 service (needs your own ARC_API_KEY,
                                       # see .env.example)
```

## License

MIT-0 (no attribution required) — see [LICENSE](LICENSE). Chosen for [ARC Prize](https://arcprize.org) open-source eligibility.
