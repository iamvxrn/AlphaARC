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

## Milestone 1.5: semantic validity of the predictions

Milestone 1 only proved the interface fits. It did not prove the predictions meant
anything, and two of them did not:

* `MdlClaim.predict` returned every fourth cell of the board it had just been shown
  and asserted they would stay -- so its 0.004 on vc33 measured the stability of a
  sample, not the error of an MDL model;
* the relational verifier checked a relation on the frame AFTER an action while the
  claim had been made about the frame BEFORE, so its 1.000 meant "the relation
  stopped holding", which the claim had never asserted.

Both are fixed by construction, not by tuning.

**Evidence is now masked.** Engines receive a grid with holes and are told where the
holes are, never what is in them. A `HELD_OUT_CELLS` proposition is therefore about
something the engine could not see. `Reflect` proposes `cell(r,c) = cell(r, axis-c)`
and `Translate` proposes `cell(r,c) = cell(r, c±period)`; both fill the actual holes.
Count and Correspondence generate a saving but no per-cell consequence, so they are
**not offered at all** rather than faked into a 0.000.

**Claims are split by tense.** `RELATION_NOW` is a statement about the current frame
and is checked there; `RELATION_PERSISTS` asserts survival across an action and is
checked on the successor. Conflating them was the whole of the earlier 1.000.

**The verifier is generic.** It dispatches on the proposition's kind and never sees
the engine, the source or the task. Digests are SHA-256 over the board rather than
Python's salted `hash()`, so a fixture means the same thing in two processes.

### Verifier probed directly, independent of any engine

    held_out, every value right      0.000   expected 0.000   ok
    held_out, every value wrong      1.000   expected 1.000   ok
    relation_now, no such objects    None    expected None    ok
    transition, correct successor    0.000   expected 0.000   ok
    transition, wrong successor      1.000   expected 1.000   ok
    nothing to check against         None    expected None    ok
                                                              6/6

The third probe was a genuine bug when written: both named points fell inside the
same background component and the verifier scored "X is identical to X" as a correct
claim. It now refuses when two names resolve to one object -- untestable, not right.

### Negative controls, and the result nobody ordered

    static-mirror            mdl predicts 18 held-out cells   error 0.000
    static-mirror-BROKEN     mdl SILENT
    relational-copy          relational                       error 0.000
    relational-copy-BROKEN   relational SILENT
    vc33-transition          graph                            error 0.000
    vc33-transition-BROKEN   graph                            error 1.000

The expectation was `corrupted -> high error`. What happens is that **two engines of
three answer corruption with silence rather than error**: MDL requires a perfect
symmetry, so one violated cell removes the hypothesis; the relational engine pairs
only components with identical shapes, so damaging one removes the pair. Neither
ever emits a wrong prediction, which is honest, and which means their error path is
exercised only by the direct probes above.

**So milestone 1 is NOT closed by the stated criterion.** One generic verifier
discriminates positive from corrupted through an engine for exactly one family --
`transition`. For the other two the discrimination exists in the verifier and is
unreachable through the engines as written.

That is worth more than a green tick, because it lands directly on milestone 2: an
evaluator that accumulates *prediction error* will mostly receive **silence** from
engines that abstain under uncertainty. Error is only available as a control signal
from mechanisms willing to commit when they might be wrong. Whether to make MDL and
the relational engine commit -- a tolerance instead of a perfect-fit requirement --
is a design question the metacontroller's usefulness may depend on, and it is now on
the table before the controller exists rather than after.

## Milestone 1 result (interface only)

