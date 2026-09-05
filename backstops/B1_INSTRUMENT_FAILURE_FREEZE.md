# Failure freeze — B1 instrument is under-specified

Recorded after auditing the B1 implementation at `a24e111`, before any repair
to the specification or the measurement code.  This document does not change
the Q1 semantics and does not supersede `B1_FAILURE_FREEZE.md`.

## Frozen failure

    object        B1 measurement instrument
    requirement   "exhaustively compute the best probe accuracy achievable by
                  every lookup table fitted to acquisition-reachable data"
    spec commit   a4877fb
    code commit   a24e111
    observed      the code enumerates 16 feature subsets, but evaluates one
                  fixed majority-vote fitting rule for each subset
    seeds         none — this is a deterministic source/specification audit
    noise band    zero

The numerical run remains exactly reproducible:

    python3 backstops/q1_backstops.py

It returns B1 `0.5000`, B2 `0.2500`, and B3 `1.0000`.

## Minimal reproducible case

In `q1_backstops.py`, `run_b1()` enumerates the 16 subsets of the four
observation features.  For each subset, `b1_for_subset()` constructs exactly
one table: progress-making actions are counted uniformly across all reachable
non-goal states, the largest count wins, and ties are broken uniformly.

No enumeration or optimization over table-valued functions `f` occurs.  Thus
the reported maximum is the maximum over 16 feature projections under one
particular fitting rule, not the maximum over "every lookup table" named in
the frozen requirement.

## Why the missing quantifier cannot be filled in mechanically

The frozen specification does not define:

- what observations from acquisition constitute the fitting data (all
  reachable transitions, one trajectory, rewards, or state/action/effect
  tuples);
- which fitting algorithms are admissible;
- whether a table may combine evidence across keys before writing an entry;
- what constraint distinguishes a forbidden shortcut from a table that has
  legitimately acquired the target action-effect mapping.

With unrestricted access to the enumerated transition table, a fitting
procedure can identify the complete label-to-effect bijection and write the
correct probe action, reaching `1.00`.  That is a lookup table, but it has also
acquired exactly the regularity Q1 is intended to measure.  Excluding it
requires a formal information or algorithm restriction that `a4877fb` does
not provide.  Choosing such a restriction now would change the instrument
after inspection of the result.

## Competing explanations

**H1 — implementation defect.**  B1 intended a well-defined finite policy
class, but `q1_backstops.py` accidentally evaluated only a majority-vote member
of it.

**H2 — specification defect.**  "Every lookup table fitted to acquisition
data" is not a finite policy class until the fitting data and admissible
fitting rules are defined.  The implementation had to invent the
majority-vote rule because the frozen text does not determine an algorithm.

## Discriminating evidence

The majority-vote rule and uniform weighting appear only in the implementation,
not in `PHASE1_Q1_SPEC.md`.  Conversely, the specification supplies no other
definition from which they can be derived.  This supports H2.  H1 cannot be
tested without first adding semantics that were not frozen.

## What is and is not licensed

- The concrete `0.5000` counterexample remains valid for the implemented
  majority-vote table.  Since frozen B1 says anything above `0.25` rejects the
  design, it is sufficient to keep Q1 rejected if that table is admitted.
- The stronger claims that B1 was executed exhaustively over every fitted
  lookup table, or that `0.5000` is the best achievable value for that class,
  are not supported.
- B2 and B3 are unaffected: their policy classes and exact calculations are
  fully specified, and their results remain `0.2500` and `1.0000`.

## Status

    FAILURE OBSERVED — CAUSE IDENTIFIED
    B1 INSTRUMENT NOT EXECUTABLE AS LITERALLY FROZEN
    Q1 SPEC UNCHANGED; IMPLEMENTATION UNCHANGED

No repair, successor design, agent, or `U` is introduced here.
