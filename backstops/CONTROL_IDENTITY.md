# The re-draw control identity

Deterministic result, no seeds and no agent. Verify:
`python3 backstops/control_identity.py` (from `backstops/`).

It was found while auditing the E0 reference experiment, but it is a property of
Q1's own measurement design, so it is recorded here rather than there.

## The identity

For **any** probability distribution `p` over the four labels:

    Σ over all 24 rule instances R of  p[ℓ⁺(R)]  =  6

`ℓ⁺(R)` is the label `R` maps to the probed effect. Each label is the answer for
exactly `3! = 6` of the 24 bijections, so the sum is `6 · Σp = 6` — regardless
of `p`. An oracle, an adversary, a uniform-random policy and an agent that has
acquired the entire mapping all give `6`.

## The consequence

Any control that re-draws `R` at probe time and re-scores the **same** prediction
is therefore pinned:

    MISMATCHED = (6 − MATCHED) / 23

It has no freedom. It cannot vary independently of the matched arm, and it
**falls** monotonically as the matched arm rises:

| MATCHED | 1/4 | 1/2 | 3/4 | 1 |
|---|---|---|---|---|
| MISMATCHED | 1/4 | 0.2391 | 0.2283 | 0.2174 |

## What this does to `PHASE1_Q1_SPEC.md` §10

§10 predeclares, as protocol rule 4 requires:

> **C (must improve):** MATCHED first-action accuracy > 1/4
> **D (must NOT improve):** MISMATCHED first-action accuracy stays at 1/4
> If **both** arms rise above chance the effect is non-specific and Q1 is **not**
> demonstrated.

**Both arms rising is impossible.** The failure mode D was written to detect
cannot occur, so D is not an independent condition — it is an affine image of C.
Reading `D ≈ 1/4` as corroboration of specificity reads the arithmetic of the
scoring, not the behaviour of the agent.

§7's pairing does not rescue it: the identity holds per acquisition history, so
it survives averaging over paired seeds. §9's *"paired MATCHED − MISMATCHED →
was the improvement caused by the relevant experience?"* is likewise an affine
transform of MATCHED alone, and answers nothing MATCHED did not already answer.

This does not un-reject Q1, and does not touch the `0.50` in
`B1_FAILURE_FREEZE.md`, which came from B1 and not from this arm. `a4877fb`
stays frozen and unamended.

## Scope of the claim

The identity needs exactly three things, all of which Q1 has:

1. `R` drawn uniformly from **all** bijections of labels onto effects (§3).
2. The probe answer determined by `R` alone (§6).
3. The measured quantity a distribution over labels, fixed **before** any
   probe-phase transition (§6, "the first action taken at the probe").

Remove any one and the identity lapses. It says nothing about controls that
change the agent's *evidence* rather than the *answer key*.

## Confirmed against the E0 reference

Checked directly against `outputs/core-zero-reference/` (audited at
`results.json`, reproduced here byte-identically, 7/7 tests pass):

- **INDEPENDENT control.** Returns exactly `1/4` for a perfect oracle, an
  anti-oracle, a constant policy and an adversarial random policy alike. It has
  no diagnostic power whatsoever.
- **EXCLUDED-IDENTICAL.** Exactly `(6 − MATCHED)/23`, verified on all of the
  above.
- **The arm E0 does not have** — re-run acquisition under `R′`, probe under `R`,
  the direct analogue of Q1 §7 — was computed over all 24 rules × 24 orders × 5
  budgets × 48 queries. It is `(6 − MATCHED)/23` at **every** budget, exactly.
  Building the missing arm would add nothing.
- **AXIS-ONLY** is forced to exactly `1/2`: the winning action is *always* among
  the two candidates on the required axis, in all 27,648 cases.
- **EXACT-OBSERVATION LOOKUP**, **ORACLE** and **VIOLATION ORACLE** are literal
  constants in `run_experiment.py` — `+= Fraction(1,4)` and `+= 1` — after an
  assertion. No lookup learner and no oracle is constructed at scoring time.
  (The oracle *is* independently checked, but in `test_reference.py`, not by
  that row.)

E0's own SPEC is careful — it calls these diagnostics and says a perfect result
is expected from construction. The sharper statement is that five of its eight
control rows cannot take any other value than the one reported, for any learner
whatsoever, so they are not evidence about the learner.

**E0's real measurements** are the matched curve, the erasure control, and the
stationarity-violation diagnostic. Those three do vary with what the learner
holds, and the last two are exactly the controls that change the *evidence*
rather than the answer key — which is why they work.

## Constraint carried forward to Q1′

Together with `B1_PRIME_RESULT.md`:

> A control for Q1′ may not be constructed by re-drawing the label→effect map at
> probe time. Under a uniform draw over all bijections, every such control is an
> affine image of the matched arm. A control must intervene on the **evidence** —
> erase it, withhold specific transitions, corrupt it, or replace the acquisition
> history — and its predeclared prediction must name a value that the matched arm
> does not already determine.

`B1_PRIME_RESULT.md` constrains the *environment*; this constrains the
*control*. Neither says a design satisfying both exists.
