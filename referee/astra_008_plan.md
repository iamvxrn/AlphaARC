# Astra 008: split-independent alternative audit

## Claim to test

`H_near` is a valid *mathematical* indistinguishable alternative only when the
identity of the probe source is part of the environment prior.  If the source
held out for evaluation is sampled **after** the environment family and K0 are
frozen, that alternative is split-contingent and cannot be included as a
pre-split environment hypothesis.  A legitimate pre-split substitute is an
exchangeable `H_one`: exactly one source is exceptional, selected uniformly
before the evaluation split.

## Fixed finite audit

For `N = 3, 4, 5` sources and the 24 bijections:

1. sample the held-out source `J` independently of the environment;
2. under `H_one`, select an exceptional source `E` uniformly, use common rule
   `R` everywhere except `E`, whose rule is uniform `S`;
3. observe every source except `J`; evidence `A` is that all observed tables
   agree;
4. use the observed common table to answer the held-out source;
5. enumerate `(J, E, R, S)` exhaustively and compare the counts with closed
   forms.

No environment, feature, agent, update rule, or benchmark is changed.  This is
only an audit of the inference claim in dialogue/007.

## Pre-registered expected results

For `H_one`:

`P(A) = (N + 23) / (24 N)`;

`P(E = J | A) = 24 / (N + 23)`;

the specified history policy has conditional probe accuracy
`(N + 5) / (N + 23)`.

`H_shared` has `P(A)=1` and accuracy `1`, so its Bayes factor against `H_one`
is `24N/(N+23)`, not 1.  A counterexample to any equality above falsifies this
claim.

The audit starts at three sources because with one acquired source the
``all acquired tables agree'' event is vacuous.

## Limit

This cannot make an objectively correct prior emerge from data.  It tests the
narrower procedural claim: an evaluation-split-indexed alternative is not
admissible when the split is genuinely post-freeze and hidden; exchangeable,
pre-split alternatives remain and must be declared.
