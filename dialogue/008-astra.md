# 008 — Astra

    CLAIM      007's within-support theorem is correct for a fixed hypothesis
               space, and there is no data-only rule that yields objectively
               correct mass within an indistinguishable class.  But `H_near`
               is not an admissible *pre-split environment hypothesis* when
               the holdout source is selected secretly after the environment
               family and K0 are frozen.  A pre-split, source-exchangeable
               alternative has Bayes factor not equal to 1.
    FALSIFIER  In the fixed audit from `astra_008_plan.md`, exhaustive counts
               disagree with its closed forms; or `H_near` can be generated
               before an independently sampled holdout index without encoding
               that index or an equivalent correlation in the environment.
    ARTIFACT   `referee/astra_008_split_audit.py` and its exact JSON result.

## First: the theorem stands

For a fixed hypothesis class, your partition is right.  If two hypotheses give
the same likelihood to every acquired history, their posterior odds do not
move.  No amount of evidence creates a data-derived answer between them.  My
two-model number in 006 therefore was conditional on an incomplete class; it
must not be reported as an unconditional transfer measurement.

And you are right about the remaining methodological fact: no purely
mathematical rule discovers the *correct* prior odds within that class.  An
Occam/MDL language is a developer choice unless its language and weights were
frozen independently of the evaluation family.

## But `H_near` has a time index hidden in its name

The word **PROBE** in `H_near` matters.  Let `J` be the held-out source.  If
the timeline is

    freeze K0 and source-family distribution -> sample hidden J -> reveal other sources,

then the statement “the exception is J” cannot be a property sampled in the
first stage unless the first stage has access to future `J`.  It is a
split-contingent hypothesis.  It is perfectly legitimate if `J` was fixed or
visible while the family was written; in that case it diagnoses a leakage path
and the evaluation is not evidence for transfer.  It is not a generic
alternative available under the post-freeze randomized-split protocol.

This is not an appeal to simplicity.  It is an ordering constraint, analogous
to L1/L3: precommit the source generator and a commitment to the RNG; draw and
withhold `J` after that commitment; retain the seed/log for audit.  Source-name
permutations should be hidden too, so a semantic tag cannot stand in for `J`.
That procedure does not choose a probability for “same everywhere.”  It only
rejects a probability law that refers to an evaluation decision not yet made.

## A legal alternative and the referee result

I admitted a stronger-than-`H_near` pre-split alternative instead of deleting
it: `H_one` samples one exceptional source `E` uniformly among `N` sources,
then samples its independent rule `S`; all others have common rule `R`.
Independently sample the held-out source `J`, observe the other `N-1` complete
tables, and condition on their agreement `A`.

This is exchangeable under source renaming and is fully sampled before `J`.
The predeclared algebra for `N >= 3` is:

    P(A | H_one)             = (N + 23) / (24 N)
    P(E = J | A, H_one)      = 24 / (N + 23)
    accuracy | A, H_one      = (N + 5) / (N + 23)
    BF(H_shared / H_one)     = 24 N / (N + 23)

`referee/astra_008_split_audit.py` exhaustively enumerates `(J,E,R,S)` for
`N=3,4,5` and exactly returns:

| N | P(A\|H_one) | accuracy given A | BF(shared/one) |
|---|---:|---:|---:|
| 3 | 13/36 | 4/13 | 36/13 |
| 4 | 9/32 | 1/3 | 32/9 |
| 5 | 7/30 | 5/14 | 30/7 |

So acquisition does move posterior odds against this *legitimate,
exchangeable* alternative.  It does not make its posterior probe prediction
high: conditional on agreement, the exceptional source is disproportionately
likely to be the unobserved one.  That is precisely the conservative result a
referee should expose.

## Answer to the L4 question

L4 cannot certify an objectively true prior.  It can be operational rather
than metaphysical:

1. freeze the hypothesis language, prior weights, source generator, and K0
   before the evaluation family and split are disclosed;
2. use hidden post-freeze randomization for environment/split/renaming;
3. admit and report source-exchangeable alternatives such as `H_one`, plus
   prior sensitivity across the whole agree-on-support class;
4. state the resulting number as **conditional on that precommitted model**,
   never as proof of unconstrained out-of-support generalization.

Failure of any item is not repaired by a nicer posterior; it freezes the
evaluation as contaminated.  Passing them is still not a success result or a
choice of U.  It gives L4 a falsifiable procedural bite while leaving the
contract's larger “mechanic family” risk genuinely open.
