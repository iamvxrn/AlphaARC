# core-zero contract, v0

Operational rules for this branch. `README.md` states the hypothesis; this file
states what we are and are not allowed to do while testing it. It contains no
cognitive nouns and names no architecture, on purpose.

## Why this file exists

The first draft of this branch broke its own rule inside one day. `README.md`
listed a "candidate K₀" of persistence, prediction, compression, memory and
intervention, and the first design proposed an action-conditioned predictor,
updated online, driven by prediction error. That is a specific
predictive-coding agent, committed to before any failure demanded it — the
exact move this branch exists to prevent. It is removed, not renamed.

The second error was quieter: "no pretraining corpus" was written down as a
principle. It is not one. It confuses *minimal task-independent prior* with
*blank weights*. A human's starting state is the product of enormous prior
optimisation; it is not empty, it is not task-specific. What is forbidden here
is leakage, not learning.

## Three levels, kept apart

| | what it is | example | fixed when |
|---|---|---|---|
| **ARCHITECTURE** | what the agent is physically able to do | it carries an internal state across steps; it can act; it can observe | before everything |
| **K₀** | what it holds before meeting a new environment | task-independent structure; **may be learned** | frozen before the test environment is chosen |
| **K_t** | what it acquired inside *this* environment | "the third press of that thing changes the top row" | starts empty, grows only from interaction |

The confusion this table exists to prevent:

- "an internal state can carry information across steps" — ARCHITECTURE
- "what happened before may matter for what happens next" — possibly K₀
- "pressing this twice arms it" — always K_t, never anything else

## The skeleton

    o_t                              observation, supplied
    a_t                              action, from the supplied action set
    h_t                              internal state

    h_{t+1} = U(h_t, o_t, a_t, o_{t+1})
    a_t     = P(h_t, o_t)

`h` is **not named**. It is not memory, not a world model, not a belief, not an
embedding, not a graph, not a program. It is whatever the agent carries between
steps.

`U` is **not chosen**. It is not gradient descent, not prediction-error
minimisation, not compression, not associative recall, not Bayesian update.
What `U` should be is the research question of this branch, not its premise.

`P` is not chosen either, and whether it is genuinely separate from `U` is open.

Minimal definition of a learning agent, assumed rather than discovered: a
persistent internal state that can be changed by experience. Demonstrating that
an agent without one fails a history-dependent task is a restatement of that
definition, not a result, and no experiment on this branch may be built to show
it.

## Supplied from outside (the interface)

Only these cross the boundary into the agent:

1. **an observation** per step, in whatever raw form the environment emits
2. **the set of available actions**
3. **an objective signal** — see Assumption A1

Nothing else. No environment identifier carrying meaning, no documentation, no
per-game metadata, no tags, no baselines, no decoded mechanics.

## What counts as leakage

K₀ may be as large, as trained, and as expensive as we like. It may not contain
information about the environment it will be tested on. Operationally:

- **L1 — order.** K₀ is frozen, and its hash recorded, before the test
  environment is selected. Anything changed after seeing the test environment is
  K_t or a new experiment, never K₀.
- **L2 — provenance.** If K₀ is learned, its training data and procedure are
  documented in full, and frozen before evaluation environments are selected, so
  that transfer can later be analysed at several notions of similarity rather
  than asserted at one. The test environment itself may not appear in the
  training data. *This is deliberately weaker than the criterion we want* — see
  below.
- **L3 — no constant fitted to a test environment.** No number in K₀ may be set
  by reading behaviour on an environment K₀ will later be scored on. This repo
  has already made that mistake once and had to retract a generalisation claim
  over it; the retraction is the reason this line is here.
- **L4 — the developer channel.** We personally know the decoded mechanics of 17
  games from the other branch. That knowledge may not enter K₀ through hand-set
  structure, chosen features, or "obviously it should look at X". Leakage
  through the author is still leakage.
- **L5 — more than one environment.** The same frozen K₀ is tested on at least
  two mechanically distinct environments. A K₀ that only works on one is
  indistinguishable from a K₀ fitted to it.

### The stronger criterion we do not yet have

We would like to require that the training distribution contain no environment
of the *same mechanic family* as the test. That requirement is currently
unusable: **"mechanic family" has no sharp definition**, and defining one in the
abstract — before any concrete training distribution exists — would be days
spent adjudicating whether "an entity moves when acted upon" is the same family
as "a coloured square moves on click". It is left open on purpose.

It is not, however, abandoned. When a learned K₀ actually exists, transfer gets
measured along explicitly named axes instead of a single intuitive word:

    IID
    new configuration
    new composition            (primitives seen, their composition not)
    new dynamics
    new causal structure
    new observation encoding
    new domain

That turns an unfalsifiable claim — "it never saw this mechanic family" — into
checkable ones: *"A and B appeared separately in training, A∘B never did"*, or
*"no training environment contained a transition rule of this class"*.

Until then this is the branch's largest open methodological risk, and it is
recorded as risk rather than resolved by wording: "trained on a million
synthetic worlds" differs from "a pretrained ARC solver" only to the extent that
this line can eventually be drawn and defended.

## Assumptions, recorded as assumptions

- **A1 — the objective is given.** The agent is told what to maximise; the
  environment's score is passed through. This is a deliberate simplification to
  keep *learning how the world works* separate from *discovering what to want*.
  It is not a claim that an externally supplied objective is part of cognition.
  Goal discovery is a different experiment and stays out of scope until this one
  produces something.
- **A2 — episodic boundaries are given.** The agent is told when an episode
  starts. Whether anything of `h` may survive that boundary is open, not settled.

## Open, and deliberately unanswered here

- What is `U`?
- Is `U` itself fixed, or does it change with experience?
- What is the structure of `h`?
- Is `P` separable from `U`?
- Does `h` persist across episodes, across levels, across environments?
- How is "mechanic family" defined sharply enough to make L2 testable?

## The rule that governs additions

Moved to [GROWTH_PROTOCOL.md](GROWTH_PROTOCOL.md), and weakened in the process.
"No capability without a failure that requires it" assumed a failure identifies
the capability that would fix it; it does not. A failure licenses an
investigation, and only a discriminating experiment licenses a mechanism.

The one clause that stays here, because it is about knowledge rather than
growth: a failure that is a corollary of the definition of a learning agent —
such as a stateless agent losing a history-dependent task — is not a finding and
may not motivate anything.
