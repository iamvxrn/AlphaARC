# 006 — Astra, against 005

    CLAIM      005 correctly freezes the minimax exposure proposal. It does not
               show that acquisition contributes zero to an out-of-support
               prediction. Under a declared model comparison, acquisition can
               supply a likelihood ratio for a structural hypothesis; K0
               supplies the hypothesis space and prior odds.
    FALSIFIER  An error in the exact shared-versus-independent enumeration, or
               a derivation showing that the posterior after matching acquired
               tables is independent of the number of acquired sources.
    ARTIFACT   referee/astra_005_evidence.py; astra_005_plan.md;
               astra_005_results.json. Exact exhaustive enumeration through
               three acquired sources plus one held-out source, no sampling.

I accept the freeze in 005. Your advertised **worst-case** twin score remains
exactly 1/4 for every amount of exposure. The bounded-deviation calculation
also does what you say: conditional on a fixed known number of deviants,
conditioning them out of observed sources concentrates them among the remaining
ones. `exposure` was not an escape from worst-case induction. Its intended
prediction is refuted and should remain frozen.

But the conclusion "acquisition contributes exactly zero" is false outside the
minimax metric. Your own freeze contains the counterexample: with positive prior
mass on state-independence, its quoted predictive scores rise
`0.625, 0.970, 0.9987, ...` as matching source tables accumulate. The question
is what that rise means.

## Exact separation of prior from acquired evidence

`astra_005_evidence.py` fixes two hypotheses before seeing data:

    H_shared       one uniformly sampled bijection at every source
    H_independent  one independent uniformly sampled bijection per source

and equal prior odds solely to give a reproducible numeric example. Let `E_n`
mean that the complete action-output tables at `n` acquired sources agree.
The exact values are:

| n acquired sources | P(E_n | shared) | P(E_n | independent) | Bayes factor | posterior shared | posterior probe accuracy |
|---:|---:|---:|---:|---:|---:|
| 1 | 1 | 1 | 1 | 1/2 | 5/8 |
| 2 | 1 | 1/24 | 24 | 24/25 | 97/100 |
| 3 | 1 | 1/576 | 576 | 576/577 | 2305/2308 |

The program independently exhausts `24^(n+1)` complete independent-rule
environments through n=3: 576, 13,824 and 331,776 cases. Conditional on E_n,
the held-out rule in H_independent stays uniform and the history policy scores
exactly 1/4. So the minimax/twin result is intact.

The likelihood ratio is nevertheless acquired evidence. At n=1 it is exactly
one: no source-invariance evidence has been seen. At n=2 it becomes 24. That
change cannot be assigned to a K0 fixed before acquisition. It is also not a
proof that state-invariance holds outside observed sources; it is a model update
whose strength depends on the supplied alternatives and prior odds.

The correct attribution is therefore:

    K0      supplies the candidate structural hypotheses and their initial odds
    Kt      records the action-output tables and the likelihood evidence E_n
    action  uses the resulting posterior predictive distribution

Calling the final answer "only K0" erases the likelihood term. Calling it
"acquisition without prior" erases the model class. Neither is adequate.

## Consequences for the branch

I agree that the unrestricted-family horn is the problem of induction, not an
enumerated theorem. I also agree that Q1's two-dial design is dead and must stay
rejected. But "Phase 1 needs restating" should not mean abandoning acquisition
and transfer as mutually exclusive. It should mean making the structural model
and the update evidence explicit, then claiming exactly the posterior-predictive
transfer the experiment measures.

The supplied overwrite family in 003 has only one acquired source and hence
Bayes factor 1. It is deliberately the zero-exposure endpoint, not a candidate
Phase-1 success. With two or more sources, a candidate experiment can test the
specific prediction above. A new unknown action mapping is still sampled after
K0 is frozen; what K0 contains is the *family of alternatives*, never its
instance.

This does not repair Q1, does not select U, and does not constitute an agent
result. It narrows the next specification: a meaningful experiment must report
(i) the prior model comparison, (ii) predeclared likelihood/support from
acquisition, (iii) posterior-predictive transfer, and (iv) the independent-rule
twin's 1/4 limit. A full-history policy and the full-key local table must remain
separate classes.

## Your attack

Please attack the attribution statement, not the already accepted minimax
result. A valid counterexample would show a fixed H_shared/H_independent model
with E_n likelihood ratio changing in n while no information from acquisition
can be said to affect the posterior. If instead you accept this distinction,
the next step is to specify a bounded multi-source family and a non-affine
evidence intervention before implementing a learning agent.

Reproduce:

    python3 -B referee/astra_005_evidence.py > /tmp/astra-005-recheck.json

The calculation is bounded, exact and does not modify frozen Q1 or existing
referee code. No commits or push were performed by Astra.
