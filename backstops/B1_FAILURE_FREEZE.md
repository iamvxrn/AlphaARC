# Failure freeze — B1, Q1 backstop

Recorded under GROWTH_PROTOCOL rule 1, **before** any repair. The
specification is not modified by this document.

## The frozen failure

    metric        best probe accuracy of an acquisition-fitted lookup table
    required      0.25   (PHASE1_Q1_SPEC.md §11: "Anything above rejects the design")
    observed      0.50   on feature subset K = {d1, t1}
    spec commit   a4877fb
    code          backstops/q1_backstops.py
    seeds         none — exhaustive and deterministic over all 24 rule
                  instances and all 16 feature subsets; noise band is zero

Reproduce: `python3 backstops/q1_backstops.py`

## Minimal reproducible case

Probe is `S* = (d1=0, d2=0, t1=0, t2=1)`. Project onto `K = {d1, t1}`; the key
is `(0, 0)`. Acquisition-reachable non-goal states with that key, and the labels
that make progress at each (all readable by the agent from its own transitions,
since the target is in the observation):

    (d1=0, d2=1, t1=0, t2=0)   dist 1   progress: decrement
    (d1=0, d2=2, t1=0, t2=0)   dist 2   progress: increment OR decrement
    (d1=0, d2=3, t1=0, t2=0)   dist 1   progress: increment

Tally at key `(0,0)`: increment 2 votes, decrement 2 votes — an exact tie
between the two dial-2 labels, and **zero votes for either dial-1 label**. A
majority-vote table therefore emits a dial-2 label at the probe, and breaking
the tie uniformly gives `0.5`.

## Two competing explanations

**H1 — the design is broken.** The probe admits a shortcut that beats chance
without acquiring the target regularity, so a PASS would not mean what §13 says
it means.

**H2 — the B1 threshold is mis-specified.** §8's randomization argument proves
`0.25` for policies *independent of acquisition experience*. §11 implemented B1
over the strictly larger class of tables *fitted to acquisition data*, then
applied the smaller class's threshold to it. §8 itself already predicts `0.50`
for the larger class, in the paragraph headed "The randomization argument does
not close everything". Under H2 the design is intact and one sentence in §11
contradicts §8 and §9.

## Discriminating evidence (already in hand)

Does the `0.50` table need acquisition transitions? **Yes.** To place any dial-2
label at key `(0,0)` it must already know which two labels act on dial 2, and
that is only readable from its own transitions. It is therefore not a policy
independent of acquisition experience, and §8's randomization argument is not
violated — the exhaustive run confirms `0.25` for every subset keyed on `t2`
(all unreachable at the probe) and for the constant policy.

This supports **H2** and does not support H1.

## Corroborating results from the same run

- **Probe unreachability holds.** 64 acquisition-reachable observations, none
  with `t2 ≠ 0`; `S*` is not among them. §5's novelty-by-construction claim is
  verified exhaustively, not argued.
- **The SUPPRESSED prediction of §9 is confirmed exhaustively.** Subsets
  `{d2}`, `{d1,d2}`, `{d2,t1}` score exactly `0.0000` — below chance, as §9
  predicted for reward-following. This was a predeclared qualitative call and it
  came out right.
- **B2** `0.2500` exact — PASS. **B3** `24/24 = 1.0000` — PASS.

## Consequence for interpretation, whatever the resolution

Independently of H1/H2, one thing is now measured rather than assumed: `0.50` at
the probe is reachable by a majority-vote lookup table keyed on two features. So
a `0.50` result from a real agent is weak evidence of anything, and the gap
between PARTIAL and FULL on §9's ladder is doing far more work than the gap
between NONE and PARTIAL.

## Status

    FAILURE OBSERVED — CAUSE IDENTIFIED (H2 supported, H1 unsupported)
    SPEC NOT MODIFIED — resolution deferred to review
