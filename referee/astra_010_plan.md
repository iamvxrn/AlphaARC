# Astra 010: deviation-family endpoint audit

## Claim to test

For exactly `d` exceptional sources, the direction asserted in dialogue/009
(``more exceptions raises the reported transfer'') is not valid across the
whole stated family.  It holds for the displayed interior examples but can
reverse at the endpoint `d=N`, where there is no nonexceptional source whose
rule can serve as the designated common rule.

## Fixed computation

Use the same exhaustive model and equal prior weights as
`referee/split_sensitivity.py`, with `N=3` and every permitted
`d in {1,2,3}`.  Enumerate `J`, exceptional sets, common rule and exceptional
rules; condition on agreeing acquired tables; calculate `P(A)`, conditional
accuracy, and posterior predictive score.

Expected counterexample: `d=2` has a larger predictive value than `d=3`.
Any non-decrease at that pair falsifies the claim.

## Consequence if confirmed

The correct methodological conclusion is not a preferred direction of
conservatism.  The complete, predeclared deviation family (including its
endpoints), its weights, and the source-count sampling measure must be
reported.  None is determined by the split procedure; all must be frozen
before evaluation.
