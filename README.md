# core-zero

## Hypothesis

Can an agent equipped only with a minimal, task-independent cognitive core
acquire — through interaction alone — enough knowledge of a single previously
unseen environment to reach competent performance?

No numeric target is stated here. An earlier draft named 0.85 on ARC-AGI-3; it
is removed rather than kept as "provisional", because it was silently calibrated
to *adult* human performance — which carries acquired linguistic and cultural
scaffolding ("legend", "button") far beyond the core-knowledge `K₀` this branch
means to test. A number goes back in once there is an honestly calibrated
baseline to set it against. See CORE_ZERO_CONTRACT.md's calibration-risk note.

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

    a failure licenses an INVESTIGATION, not a mechanism

An earlier draft said "no new capability without a failure that requires it".
That is too strong logically: an observed failure almost never identifies the
capability that would fix it. "It does not transfer to a new configuration" is
equally consistent with a representation limit, a capacity limit, an
exploration failure, overwriting, misplaced credit, or noise.

So the rule is a chain, not a step:

    failure → competing explanations → discriminating experiment
            → identified limitation → minimal intervention → ablation

with `FAILURE OBSERVED — CAUSE UNKNOWN` as an explicitly valid ending. Full
version, including the predeclared-control rule and the per-capability
lineage files, in [GROWTH_PROTOCOL.md](GROWTH_PROTOCOL.md).

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
    after 5 actions      ?
    after 20 actions     ?
    after 50 actions     ?
    after 100 actions    ?

with no numbers filled in here on purpose (see the Hypothesis section above),
compared against how much evidence a *matched* baseline needs to reach the same
mechanic-understanding — matched meaning something whose own prior is argued to
be similarly minimal, not an adult human by default — not just the final score.

Stronger test of generality: the same frozen K₀ should be able to acquire
competence in more than one kind of environment (ARC-AGI-3 games, at
minimum two mechanically distinct ones), not a K₀ silently specialized to
one benchmark.

## Status

Nothing built yet. This file is the entire branch.
