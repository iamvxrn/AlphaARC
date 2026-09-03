# growth protocol, v0

How a capability is allowed to enter `core-zero`.
`CORE_ZERO_CONTRACT.md` governs what the agent may *know*; this file governs how
it may *grow*. Both are normative.

## What this replaces

The opening rule of this branch was:

    no new capability without a failure that requires it

It is not enough, because of the word *requires*. It assumes an observed failure
identifies the capability that would fix it. It does not.

"The agent does not carry what it learned into a new configuration" is equally
consistent with, at least:

    the representation cannot express the invariant
    the update rule cannot reach it from this data
    capacity is exhausted
    exploration never produced the relevant transitions
    what was learned earlier was overwritten by what came after
    credit landed on the wrong step
    the experience was enough, but the objective rewarded something else
    nothing is wrong and this is noise

If each such failure licenses an architectural hypothesis, this branch rebuilds
AlphaARC inside a month, and in six months nobody will be able to say which of
the resulting modules was ever load-bearing.

## The chain

    failure
      → competing explanations
      → discriminating experiment
      → identified limitation
      → minimal intervention
      → ablation

The load-bearing move, stated once:

    a failure licenses an INVESTIGATION, not a mechanism.

## The rules

**1 — Freeze the failure first.** Before any change to the core, record: the
minimal reproducible case, the metric, the commit hash, the seed count, and the
noise band. A failure that has not been frozen cannot later be shown to have
returned, which makes rule 6 unenforceable.

**2 — No single-story diagnosis.** Write down at least two plausible competing
explanations *before* testing any of them. Being able to think of only one is
evidence that the failure is not understood yet — not evidence that the
diagnosis is obvious.

**3 — Discriminate before you modify.** Prefer experiments that change the
evidence or the conditions over experiments that change the agent. Most of these
are far cheaper than an architectural change, and several can be run in an
afternoon:

| hypothesis | discriminating experiment | leaves the core |
|---|---|---|
| not enough experience | more transitions of the same mechanic | untouched |
| not enough capacity | scale the existing `U`, change nothing qualitative | qualitatively same |
| exploration never found it | supply scripted/oracle trajectories | untouched |
| earlier learning was overwritten | replay stored experience | untouched |
| the representation cannot express it | re-encode the observation, preserving structure | learning/control same |
| genuinely missing structure | vary surface statistics while holding the causal decomposition fixed | — |

Only the last row is a reason to build something. This repo has already saved
itself several days this way once: a structural hypothesis about movement goals
was run over every board in its class first and died in five minutes, before any
code existed to implement it.

**4 — Predeclare the specific prediction, including the control.** Before
implementing X, write down (a) the metric and condition **C** where X must
improve, and (b) a control condition **D** where X must *not* improve. In the
commit, before the run, so the wording cannot drift to fit the numbers
afterwards. If X improves C and D equally, the improvement is non-specific —
capacity, or noise — and the stated explanation is unsupported even though the
number went up. Do not vary the intervention along the same feature the proposed
mechanism keys on; that confound makes the decisive comparison meaningless.

**5 — Minimal intervention.** The smallest implementation that could fix the
predicted failure. If a larger version is needed before any effect is visible,
that is evidence about *capacity* — a different hypothesis, which now needs its
own row in the table above.

**6 — Ablate, or it did not happen.** After the capability is in and the metric
has moved, take it back out. The frozen failure must return. If it does not, the
capability is not doing what was claimed, whatever the aggregate says.

**7 — Measure above the noise, paired.** A difference smaller than the noise
band is not a result. The other branch measured `sd 1.07` on a score of `1.5` —
identical code, four seeds, a 5.7x spread — and had to retract one headline
result and five "rejected" conclusions that had all been living inside that
band. Compare paired on identical seeds, state every number with its seed count,
and add seeds until the standard error is small enough to carry the claim.

## Valid outcomes

    REQUIRED    the failure returns on ablation, and only C improved
    SUPPORTED   consistent with the evidence, ablation not yet decisive
    UNCERTAIN   the number moved, the explanation did not survive rule 4
    REJECTED    the discriminating experiment ruled the explanation out

and, explicitly:

    FAILURE OBSERVED — CAUSE UNKNOWN

This last one is a **valid terminal state**. It is written into the protocol on
purpose: without it, every investigation is under quiet psychological pressure
to end in an addition, which is exactly how a minimal core becomes a museum.
Leaving a failure unexplained is a normal, publishable outcome here.

## Lineage

Every non-trivial capability in the agent keeps a file at
`lineage/<capability>.md`:

    CAPABILITY: X

    Observed failure:        F (frozen at commit ..., N seeds, noise band ...)
    Competing hypotheses:    H1, H2, H3
    Discriminating evidence: E
    Predicted intervention:  adding X should improve M on condition C
                             and should NOT improve control D
    Result:                  ...
    Ablation:                ...
    Status:                  REQUIRED / SUPPORTED / UNCERTAIN / REJECTED

The purpose is a specific question, askable in six months of any strange thing
in the architecture: **what are you doing here?** — and answerable with an
experiment instead of "we liked predictive coding in February".

## Standing exceptions

The protocol governs *capabilities of the agent*. It does not govern: the harness
that runs the loop, logging and tracing, measurement tooling, bug fixes, or the
definitional baseline from the contract (a persistent internal state that can be
changed by experience). None of those need a lineage file.

## The protocol's own failure mode

This is expensive, and the honest risk is paralysis — a chain long enough that
nothing is ever built, and the protocol is quietly abandoned around week three.
The escape is rule 3: reach for the *cheap* discriminators, which mostly do not
require writing an agent at all. The escape is never to skip the chain and call
a plausible story a diagnosis.
