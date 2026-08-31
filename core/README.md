# core — milestone 1: does one protocol fit three existing mechanisms?

This branch does **not** build a new system. `alphaarc-v0` is frozen on its own
branch; `python/bench/` and its 1600 lines of measured negative results are
untouched and stay the reference.

## The bet

Not "three engines share an internal language" — they do not, and forcing them to
emit the same `State'` would kill the idea with an API before it was tested. The
bet is narrower:

    claim  ->  testable prediction  ->  error scored AT THE LEVEL of the claim

`protocol.py` (124 lines) is the whole contract: `Evidence`, `Hypothesis`, `Claim`,
`PredictionSet` over five levels (cell / relation / transition / action-effect /
constraint), and `score()`, which returns **None when a prediction cannot be tested
by this observation**. Silence and error are different answers, and conflating them
is how an ensemble gets mistaken for reasoning.

## What is wrapped

Nothing new was written. Three mechanisms that already existed:

| engine | what it already was | what it now claims at |
|---|---|---|
| `mdl` | `alphaarc.mdl.PRIMITIVES` — Reflect/Translate/Count/Correspondence | CELL |
| `relational` | same-colour components grouped by shape | RELATION |
| `graph` | `(state, action) -> state'`, understands nothing | TRANSITION |

## Milestone 1 result

Three kinds of experience, one example each: a static mirrored grid, a relational
copy, and a **real captured vc33 level-3 transition** (the panel button at (46,56),
the T4 -> T5 transfer decoded on `main`).

    static-mirror     mdl speaks (cell, err 0.000); relational and graph silent
    relational-copy   mdl + relational speak (err 0.000)
    vc33-l3-transfer  mdl cell err 0.004; relational 0.000 and 1.000;
                      graph silent BEFORE observing, err 0.000 after

13 claims, 9 testable, across two levels, with no task-specific branching in any
engine. The graph engine's silence before it has seen the edge is the protocol
working, not a gap.

**And one claim was falsified**: on the vc33 transition the relational engine
asserted two objects correspond under identity and the board disagreed — error
1.000. That is the signal the whole plan is built on, produced on day one.

An earlier version of the runner scored RELATION by returning `"identity"`
unconditionally, which made that engine always correct. It was a task-specific
crutch of exactly the kind this milestone forbids; the relation is now re-read from
the observed board and compared, which is what turned a decorative 0.000 into a real
1.000.

## Explicitly NOT done

No MetaController. No memory, hierarchy, imagination or plasticity. Milestone 2 is
the prediction-error evaluator, 3 is disagreement-driven experiment selection, 4 is
the cross-format micro-suite, and 5 is the A/B/C-vs-Meta ablation, which is the
first go/no-go. Nothing is added if Meta does not beat the singles.

Three metrics are fixed **now**, before the controller exists, so that a later
`19 vs 17` cannot be argued about: paired solve delta on identical tasks/seeds;
synergy solves (tasks Meta solved that NO single engine solved); and intervention
gain (times one hypothesis's error actually changed which mechanism ran next, and
that led to the solve). The third is what separates metacontrol from a router.

    python3 core/run_milestone1.py
