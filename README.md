# core-zero

## Hypothesis

Can an agent equipped only with a minimal, task-independent cognitive core
acquire — through interaction alone — enough knowledge of a single previously
unseen environment to reach competent performance (target: ARC-AGI-3 score
≥ 0.85 on a game it has never seen)?

Stated more sharply:

    AGI may require surprisingly little task-specific prior knowledge,
    if the agent begins with the right domain-general core for
    constructing persistent abstractions and causal models from experience.

Not: *human core knowledge = AGI*.
But: *human core knowledge + general learning machinery ⇒? AGI*, with the
`?` treated as the thing this branch exists to test, not assume.

## What this branch is not

This is not a refactor of `core-protocol` (M1–M4). That line of work asked
*what operations does an adaptive reasoner need?* and produced real results —
it stays frozen, untouched, on its own branch.

This branch asks a different, prior question:

    What must exist before such a reasoner can learn those
    operations at all?

No code, concepts, or solutions carry over from AlphaARC by default. If a
capability from that codebase turns out to be necessary here, it must be
re-earned: demonstrate the failure that requires it, then add the minimal
mechanism — never import it because "it already exists."

## The K₀ / K_t distinction

`K₀` is what the agent holds *before* meeting a new environment. It is **not
empty** — a human's starting state is the product of enormous prior
optimisation — and it **may be learned**. What it may not contain is anything
about the specific environment it will be tested on:

    I(K₀ ; test environment) ≈ 0

`K_t` is everything acquired about *this* environment, and it does start empty:

    K_t = ∅ at t=0, built from observation and action alone

The experiment's independent variable is what `K₀` contains. The forbidden
thing is leakage, not learning — the operational tests for that distinction,
and the separation of `K₀` from ARCHITECTURE beneath it, are in
[CORE_ZERO_CONTRACT.md](CORE_ZERO_CONTRACT.md), which is normative where this
file is only motivating.

## Growth discipline

    No new cognitive capability without a failure that requires it.

Not "it seems like the agent needs a causal graph." First show a task the
current core provably cannot learn from experience alone. Only then add the
minimal mechanism that resolves that specific failure, and record why.

## What K₀ contains

Unknown. That is the point of the branch, and naming a list here would settle
by assumption the question the branch exists to ask.

An earlier draft of this file named one — persistence, prediction, compression,
memory, intervention — and the first design that followed from it was an
action-conditioned predictor driven by prediction error. That is a specific
architecture chosen before any failure demanded it, which is the move the
growth discipline above forbids. It was removed rather than rewritten.

What is left at the conceptual level is only the skeleton:

    h_{t+1} = U(h_t, o_t, a_t, o_{t+1})
    a_t     = P(h_t, o_t)

with `h` deliberately unnamed and `U` deliberately unchosen. The question is
what `U` must be for new knowledge to appear from experience at all.

## Success criterion

Not: prediction loss decreases.

Instead: measure end-to-end competence on a game never seen before,
starting from K_t = ∅, as a function of accumulated experience —

    after 0 actions     ?
    after 5 actions      0.12
    after 20 actions     0.37
    after 50 actions     0.71
    after 100 actions    0.87

and compare against how much evidence a human needs to reach the same
mechanic-understanding, not just the final score.

Stronger test of generality: the same frozen K₀ should be able to acquire
competence in more than one kind of environment (ARC-AGI-3 games, at
minimum two mechanically distinct ones), not a K₀ silently specialized to
one benchmark.

## Status

Nothing built yet. This file is the entire branch.
