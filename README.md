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

At episode start, all game-specific knowledge is empty:

    K_0 = ∅

Everything the agent learns about a *specific* game accumulates purely from
interaction within that episode:

    K_0, K_1, K_2, ..., K_n  ← built from observation + action only

`K_0` itself is not empty — it is the frozen, task-independent core the agent
is born with. The experiment's independent variable is what that core
contains.

## Three levels of prior knowledge

| Level | Allowed at t=0 | Example |
|---|---|---|
| Interface | physics of interacting with the benchmark | observation format, list of available actions |
| Cognitive | general-purpose operations | memory, comparison, change-detection, composition, recurrence |
| Game-specific | forbidden before the episode starts | "this is a button", "color 9 means X", any named mechanic |

Even *objecthood* sits on the boundary: a general capacity for tracking
persistent entities may belong at the cognitive level, but "connected
component = object" is already a specific engineering hypothesis about
grid worlds, and is not assumed here.

## Growth discipline

    No new cognitive capability without a failure that requires it.

Not "it seems like the agent needs a causal graph." First show a task the
current core provably cannot learn from experience alone. Only then add the
minimal mechanism that resolves that specific failure, and record why.

## Candidate K₀ (starting point for reduction, not a conclusion)

A tentative, deliberately reducible list — to be shrunk and tested, not
implemented wholesale:

- persistence — things tend to continue existing across observation changes
- prediction — an internal model anticipates what comes next
- compression — regularities are preferred explanations of experience
- memory — past experience is available to present decisions
- intervention — some changes in the world are consequences of my own actions

The working bet is that higher concepts humans call "object," "cause," and
"number" are not primitives to hand-code, but *learned abstractions* that
fall out of this smaller core when it is run on structured experience:
an object is what persistence-plus-compression finds useful for predicting
a stream of observations; cause is an intervention that reliably changes
future state; number is an invariant of repeated structure.

## Success criterion

Not: prediction loss decreases.

Instead: measure end-to-end competence on a game never seen before,
starting from K_0 = ∅, as a function of accumulated experience —

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
