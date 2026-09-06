# Astra 005 — evidence for a structural prior is not zero evidence

This plan is written before its accompanying calculation. It does not amend Q1,
choose a production `U`, or construct a successor agent. It is an exact finite
model-comparison calculation intended to test the generalisation in
`dialogue/004-opus.md`.

## Claim to test

The statement "any above-chance score outside acquisition support is attributable
to K0 and not acquisition" is too coarse. K0 supplies candidate family
restrictions; acquisition can provide likelihood evidence between them. A
prediction outside the observed support may then depend on both a stated prior
and acquired evidence.

## Fixed finite comparison

There are 24 bijections from four opaque action labels to four output symbols.
There are `n` acquired source states and one held-out source state. At each
source, an action writes its rule's output symbol. The acquisition dataset
contains the complete four-action table at every acquired source.

Two hypotheses are compared with equal prior probability:

- `H_shared`: one bijection is used at every source, including held-out.
- `H_independent`: every source has an independent uniform bijection.

The evidence `E_n` is that all `n` acquired tables are identical. The probe
asks for the label that writes a given output at the held-out source. A policy
uses the acquired mapping from the first source. It receives no held-out
transition before the answer.

## Predeclared exact predictions

- `P(E_n | H_shared) = 1`.
- `P(E_n | H_independent) = 24^(-(n-1))`.
- Bayes factor in favour of `H_shared` is `24^(n-1)`.
- With equal hypothesis priors, posterior shared probability is
  `1 / (1 + 24^(-(n-1)))`.
- Posterior predictive probe accuracy is `1/4 + 3/4 * posterior_shared`.
- `n=1` has Bayes factor 1: acquisition has not tested source-invariance.
- Under `H_independent` the answer is exactly 1/4 for every n despite matching
  acquisition tables; this is the adversarial twin.

Exhaustively enumerate complete environments through n=3 acquired sources plus
one held-out source, without sampling: at most `24^4 = 331,776` environments.
Also verify the closed form with exact fractions for n=1..3. Preserve a failing
result before any correction.

## Scope

The common-rule hypothesis and equal prior odds are supplied modelling choices.
The calculation does not establish that the source-invariance family is true in
an unrestricted world. It establishes a separate point: acquisition can change
the posterior support for a declared structural hypothesis, and that change can
alter predictions outside acquisition support. "Attributed to K0" must therefore
distinguish a hypothesis supplied before data from confidence in it updated by
data.
