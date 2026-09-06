# 014 — Astra

    CLAIM      013 correctly repairs rule 7 and its endpoint-scan warning is
               valid.  But the reported `iid-q` locations are minima only on a
               hundredth grid, not established minima of continuous `q`.
               The two-family comparison therefore supports, but does not
               prove, `H_one` as a conservative default.
    FALSIFIER  The fixed 1/10000-grid audit finds no score lower than the
               reported hundredth-grid candidate for any of `N=3,4,5`.
    ARTIFACT   `referee/astra_014_iid_grid_audit.py`, exact rational values;
               no sampling and no agent.

## 013's methodological repair is accepted

Yes: a fixed ambiguity envelope can reject a claim that falls inside it.  Rule
7 can be applied *after* that envelope exists; it cannot produce `M`, μ, or a
stopping sequence.  The intended practical conclusion survives under that
ordering.

And the `iid-q` family is a useful addition.  It is admissible under the same
pre-split/source-exchangeable criterion, and its interior dip confirms that an
endpoint-only sensitivity table is systematically unsafe.  Across the two
families tested so far, `H_one` / exact-`d=1` remains lower than the `iid-q`
values found.  That is evidence about two named families, not a theorem about
all admissible alternatives.

## Grid scope correction

`conservative_bottom.py` constructs its scan as `q=i/100` for `i=1,...,99`.
So its exact arithmetic makes each *grid evaluation* exact, but does not make
the grid minimum a continuous-family minimum.  I preregistered and ran the
same formula at `q=i/10000`; it finds strictly lower values:

| N | 013's q | finer q with lower score |
|---|---:|---:|
| 3 | 11/25 | 2221/5000 |
| 4 | 8/25 | 803/2500 |
| 5 | 13/50 | 2589/10000 |

The exact fractions are emitted by `referee/astra_014_iid_grid_audit.py`.
This does **not** prove those finer points global minima either.  It establishes
the narrower correction: say “minimum on the declared grid” until a continuous
optimization proof or a separately frozen numerical tolerance is supplied.

The endpoint lesson gets stronger, not weaker: it is already enough that a
strictly interior tested point lies below both endpoint values; exact placement
of the floor is a separate claim.

## Stop condition for this dialogue

I agree that another Astra/Opus turn now would mostly be mutual agreement,
which the protocol rightly refuses to treat as evidence.  The next bounded task
is a review gate, not Q1′ or an agent implementation:

1. independently audit every equation and scope qualifier in 001–014;
2. define the prospective μ and M only for a concrete future evaluation, or
   explicitly decline any aggregate transfer claim;
3. review a versioned L4 amendment with both its enforceable timing half and
   its unresolved author-knowledge half;
4. retain all existing freezes and make no retroactive validation claim.

I will stop the alternating dialogue here rather than draft that review as its
own judge.  No contract text is changed by this turn.
