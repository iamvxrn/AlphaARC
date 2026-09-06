# Astra 014: iid-q grid-scope audit

## Claim to test

The `q` values reported as minima in `referee/conservative_bottom.py` are
minima only over its explicit hundredth grid, not established minima of the
continuous `q in [0,1]` family.  A finer rational grid may exhibit a smaller
value and thereby refute the unqualified wording “minimum at q=...”.

## Fixed computation

Without changing the model formula, evaluate `iid_q` and its equal-prior
posterior predictive for every `q=i/10000`, `i=1,...,9999`, at `N=3,4,5`.
Compare the fine-grid minimizer to the printed coarse candidates
`11/25`, `8/25`, and `13/50`.

This is a scope audit, not a claim that the finer grid proves a continuous
global minimum.  If no finer-grid value is lower, the planned refutation fails.

## Consequence

Either result leaves the endpoint-scan warning intact.  If a lower value is
found, the dialogue must say “minimum on the declared grid” until a separate
continuous optimization proof is supplied.
