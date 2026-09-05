# E0 — control audit

Added when E0 was committed into the repository. **Nothing else in this
directory is modified**: `SPEC.md` still hashes to the value frozen in
`FREEZE.json`, and every digest in `results.json` still matches its file, so
`run_experiment.py`'s own freeze check still passes and the run reproduces.

Reproduction was verified before committing: `7/7` tests pass, and a fresh run
reproduces `results.json` byte-identically in `learning_curve`, `controls`,
`enumeration` and `paired_full_acquisition_gain`.

## Finding

`SPEC.md` says a perfect result is "expected from construction". The audit
sharpens that: **five of the eight control rows cannot take any other value
than the one reported, for any learner whatsoever.** They are not weak evidence
about the learner; they are no evidence about it.

| row | reported | status |
|---|---|---|
| INDEPENDENT (all budgets) | `1/4` | **degenerate** — identically `1/4` for a perfect oracle, an anti-oracle, a constant and an adversarial policy |
| EXCLUDED-IDENTICAL | `5/23` | **determined** — exactly `(6 − MATCHED)/23`, an affine image of the matched arm |
| AXIS-ONLY | `1/2` | **forced** — the winner is always among the two candidates on the required axis, in all 27,648 cases |
| EXACT-OBSERVATION LOOKUP | `1/4` | **constant** — `run_experiment.py` asserts the key is unseen, then adds the literal `Fraction(1,4)`; no lookup learner is built |
| ORACLE / VIOLATION ORACLE | `1` | **constant** — `totals[...] += 1`, unconditionally; no oracle runs at scoring time (the oracle *is* checked independently, but in `test_reference.py`) |
| MATCHED curve | `1/4 … 1` | **real** |
| ERASED | `1/4` | **real** — intervenes on the evidence |
| STATIONARITY VIOLATION | `0` | **real** — intervenes on the evidence |

The cause of the first two rows is a single identity, derived and verified in
[`backstops/CONTROL_IDENTITY.md`](../../backstops/CONTROL_IDENTITY.md):

    Σ over all 24 rule instances R of p[ℓ⁺(R)] = 6,  for ANY distribution p

so any control that re-draws the rule at query time and re-scores the same
prediction is pinned to `(6 − MATCHED)/23`.

The arm E0 does **not** have — re-run acquisition under `R′`, query under `R`,
the analogue of `PHASE1_Q1_SPEC.md` §7 — was computed exhaustively over 24 rules
× 24 orders × 5 budgets × 48 queries during the audit. It is `(6 − MATCHED)/23`
at every budget, exactly. Building it would add nothing, and its absence is not
a gap.

## What survives

The matched curve, erasure and the stationarity violation. All three intervene
on the **evidence** rather than the answer key, which is precisely why they are
not pinned by the identity.

`BOUNDARY_FAILURE.md`'s argument is unaffected and stands as written: the
acquisition histories and the query observation are identical across the two
worlds while the correct actions differ, so no learner on that evidence can be
right in both.

## On the headline

The `25% → 50% → 75% → 100%` curve is the arithmetic of elimination among four
labels — `1/4`, then `1/2 + 1/2·1/3`, and so on — as `SPEC.md` states for the
`k=3` point. As a result it is empty.

The measured content of E0 is the other ladder, and it is worth stating in its
place:

    full observation as the key   25%
    axis only                     50%
    displacement                 100%

That is a measurement of transfer being determined by representation. It also
lands on the same wall as [`backstops/B1_PRIME_RESULT.md`](../../backstops/B1_PRIME_RESULT.md):
the `50%` rung is the pair of labels identified and the direction not, reached
there by an exhaustive projection sweep and here by an explicit ablation.

## Standing

E0 is an engineering reference, not a result about the hypothesis, exactly as
its own README says. It is committed for the measured starting point and the
boundary case, not as evidence for Core-Zero. Q1 remains rejected at `a4877fb`.
