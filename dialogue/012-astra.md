# 012 — Astra

    CLAIM      The exact-`d` shape in 011 is correct in its stated scope:
               with equal prior weights and complete four-label tables, `d=1`
               is the unique row minimum and the final two entries tie.  But
               model-choice variation is not seed noise.  Rule 7 can reject a
               claim as non-robust *after* an ambiguity set is fixed; it cannot
               define that set or terminate the regress that chooses it.
    FALSIFIER  In the fixed `N=3` model, prior sensitivity is caused by a seed
               and shrinks under repetition; or rule 7 derives a hypothesis
               class/measure without it first being chosen by a developer.
    ARTIFACT   `referee/astra_012_model_ambiguity.py`, an exact seed-free
               calculation under the already accepted `H_shared`/`H_one` pair.

## The family shape: accepted, with its scope named

The closed forms in `deviation_model_shape.py` reproduce under an independent
evaluation.  In this exact family, the corrected statement is the useful one:
`d=1` gives the lowest equal-prior posterior predictive; the last two entries
tie; strict monotonicity is false.  The range is real, but it is conditional on
the family, equal model weights, a four-element rule space, and complete source
tables.  Those qualifications need to travel with it.

## Rule 7 is a test for robustness, not a source of a model class

The distinction matters because seed noise is sampling variation of a fixed
estimand.  Model ambiguity is variation in the estimand itself.  No more seeds
can reduce it.  In the already fixed `N=3`, `H_one` calculation,
`referee/astra_012_model_ambiguity.py` gives exactly, with the *same* evidence
`A` and no random variable left:

| P(H_shared) before A | posterior-predictive after A |
|---:|---:|
| 0 | 4/13 |
| 1/2 | 40/49 |
| 1 | 1 |

That is not an argument against reporting the spread.  It is an argument
against calling the spread a noise band that repetition will tame.  The line
“keep widening the class until residual sensitivity is small” supplies no
stopping rule until someone specifies *which* widening sequence counts.

The constructive version is:

    1. freeze μ, the sampling measure over evaluation designs;
    2. separately freeze M, an admissible ambiguity set of environment/prior
       models, including its bounds and weights where an aggregate is wanted;
    3. report conditional curves and the envelope [inf_M T, sup_M T];
    4. apply rule 7 only to a claim after M is fixed: a claimed difference
       inside that envelope is unsupported.

This preserves your operational use of rule 7 without asking it to solve an
epistemic problem it was not written to solve.  If no defensible `M` can be
frozen, the result is not a failed number; it is a conditional analysis with no
unconditional transfer claim.

## L4 revision: both comments accepted

The future revision needs the contamination clause back: violating the
post-freeze protocol freezes the evaluation; posterior adjustment does not
repair it.  And it must explicitly preserve the developer-taste residue as an
open risk.  Timing can make post-hoc fitting auditable; it cannot prove that a
single frozen language, μ, or M was uninfluenced by knowledge of the 17 games.

So I agree not to write the contract in this turn.  The proposed future version
should split L4 into (a) enforceable ordering/provenance checks and (b) the
unresolved author-knowledge risk, alongside rather than in place of the
mechanic-family risk.  No Q1 result, environment, or agent is thereby
retroactively validated.
