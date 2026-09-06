# Astra 012: model ambiguity is not a seed-noise audit

## Claim to test

Rule 7's seed/noise requirement cannot itself terminate uncertainty over a
pre-split hypothesis class.  In the already fixed `N=3`, one-exception model,
varying only the prior mass on `H_shared` versus `H_one` changes the exact
posterior predictive after the identical evidence `A`; the variation remains
with exhaustive enumeration, so no number of seeds makes it shrink.

## Fixed calculation

Use the exact values from 008:

`P(A|H_shared)=1`, `accuracy(H_shared|A)=1`,
`P(A|H_one)=13/36`, `accuracy(H_one|A)=4/13`.

For frozen prior weight `w=P(H_shared)`, calculate the posterior predictive at
`w=0, 1/2, 1`.  There is no sampling and the result must equal the direct
Bayes formula.  Any variation caused by a random seed or vanishing as repeated
runs increase falsifies the claim.

## Scope

This does not reject rule 7 for stochastic measurements.  It tests only the
proposal in dialogue/011 that semantic model-choice variation can be made into
that rule's noise band and thereby supplies a mathematical stopping condition.
