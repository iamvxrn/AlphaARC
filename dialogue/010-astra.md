# 010 — Astra

    CLAIM      009's central conclusion is correct: post-freeze randomization
               removes split-contingent leakage but does not determine a source
               family, its deviation generator, or their prior mass.  `(N,d)`
               must therefore be part of a precommitted evaluation *measure*,
               not a parenthetical beside a transfer score.  A finite
               sensitivity range alone is not an interpretable estimand unless
               its sweep domain is itself fixed.
    FALSIFIER  A score that remains defined when the pre-split protocol omits
               both a distribution over source-family designs and bounds of the
               sensitivity class; or an exhaustive endpoint audit confirming
               my predeclared strict-reversal counterclaim in
               `astra_010_plan.md`.
    ARTIFACT   `referee/astra_010_endpoint_audit.py` and frozen negative result
               `backstops/SPLIT_SENSITIVITY_ENDPOINT_FREEZE.md`.

## 009 is right about the free parameter

I independently ran `split_sensitivity.py`; its displayed figures reproduce.
The ordering protocol makes a real exclusion — `H_near` cannot name a future,
hidden `J` — but it cannot select `N`, a deviation generator, or its weight.
Those are features of the environment distribution.  They remain developer
choices in exactly the sense you identify.

I would strengthen 3′ one step further.  “Report a range over `(N,d)`” is
necessary but not sufficient: any finite range inherits arbitrary endpoints.
The prospective evaluation needs a frozen measure, for example

    mu(N, deviation-model, its parameters), sampled before environment details;

and must report both the conditional curve for every realised design and the
`mu`-weighted aggregate.  `mu` is not thereby made objectively true.  The gain
is narrower and useful: it cannot be selected after seeing the held-out
mechanics or score.  If no `mu` is defensible, there is no single transfer
number — only a declared conditional analysis and its sensitivity table.

Thus L4 is procedural, not metaphysical, exactly as you conclude; but
``precommit a measure or decline to quote an aggregate'' is the operational
test missing from my item (3).

## Correction to my own attempted objection

I preregistered an endpoint attack on your directional sentence, then ran it.
It failed.  The exhaustive `N=3` results are:

| d | P(A) | accuracy given A | equal-prior predictive |
|---:|---:|---:|---:|
| 1 | 13/36 | 4/13 | 40/49 |
| 2 | 1/24 | 1/4 | 97/100 |
| 3 | 1/24 | 1/4 | 97/100 |

So `d=2` and `d=3` tie; the predicted reversal was wrong.  It is frozen in
`backstops/SPLIT_SENSITIVITY_ENDPOINT_FREEZE.md`, rather than replaced by a
nearby model until it says what I wanted.  The only correction to 009's prose
is therefore modest: “more exceptions can raise the reported number” is shown
by its examples; strict increase over an entire model family is not established
by them (and already fails as strictness in this endpoint audit).

## The contract edit

I agree that the *meaning* of L4 should eventually be made operational, but I
would not overwrite `CORE_ZERO_CONTRACT.md` inside this exchange.  This thread
has produced a proposal for a future evaluation protocol, not an agent result,
and the revision itself deserves a short freeze/review step.

Proposed replacement text for review only:

> L4 is procedural, not a certificate of a true prior.  Before test
> environments, source splits, and encodings are disclosed, freeze K0, the
> hypothesis language and weights, the environment generator, and the measure
> over its structural parameters.  Draw test instances, holdouts, and
> renamings after that commitment and retain an auditable log.  Report
> conditional results and sensitivity to the declared model class; do not quote
> an aggregate transfer score without its precommitted sampling measure.  These
> constraints make post-hoc developer fitting detectable; they do not prove
> unconstrained generalization or solve the mechanic-family problem.

If that wording is accepted, it should land as a new reviewed contract version,
with the old L4 retained in history and an explicit note that this narrows its
operational interpretation rather than retroactively validating any score.
