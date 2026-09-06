# Frozen negative result: endpoint reversal does not occur in the audited case

**Status:** REJECTED

## Claim frozen before the run

`referee/astra_010_plan.md` predicted that, for the exact-`d` deviation model
at `N=3`, posterior-predictive transfer would fall from `d=2` to endpoint
`d=3`.

## Minimal reproducible case

At baseline commit `18221f1`, run:

    python3 referee/astra_010_endpoint_audit.py

The enumerator exhausts every held-out source, exceptional subset, common
bijection, and exceptional bijection.  There is no random seed and no noise
band.

## Exact result

| N | d | P(agreement) | conditional accuracy | equal-prior predictive |
|---|---:|---:|---:|---:|
| 3 | 1 | 13/36 | 4/13 | 40/49 |
| 3 | 2 | 1/24 | 1/4 | 97/100 |
| 3 | 3 | 1/24 | 1/4 | 97/100 |

The predicted reversal is absent: the two endpoint-adjacent alternatives tie.
No new deviation model is introduced to recover the desired direction.

## What survives

Dialogue/009's broader conclusion survives: the pre-split protocol does not
determine the source-family or deviation model, and a reported conditional
transfer number must name them.  The stronger directional phrase “more
exceptions produces a higher number” is not a general theorem: this audit
already falsifies strict increase.
