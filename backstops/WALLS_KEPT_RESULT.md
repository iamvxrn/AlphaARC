# Walls kept across a reset: waste halves, score unmoved, and it is the first change to qualify

Predeclared at 66af896, measured at 16 paired seeds, `repeats=1`, full train split.

## The claim that was made

"Repeated always-miss points on sp80 and m0r0 fall sharply. This is the claim,
and it is about wasted presses, not score."

Met, on direct measurement rather than the arithmetic the predeclaration argued
from -- sp80, same seed, flag off against on:

    misses            80 -> 36
    worst point       21 hits -> 3
    distinct points    4 -> 13   (freed budget reaches further)

## The score

    aggregate over 17 games   0.4799 -> 0.4800
    delta                     +0.0002 +/- 0.0003   inside the noise

Nil, which is what the predeclaration said to expect: fix A was this same
confusion in another field and moved its own target exactly as designed while the
score did not move.

Control held: `ls20` resets once per play and moved -0.0035 +/- 0.0031, barely.
Guard held: `vc33` and `lp85` are identical on every seed.

## Why this one is kept

The README rule needs all three, and this is the first change in the
investigation to satisfy them:

    1. fixes an inconsistency between code and claim   MET -- "walls are a
       property of THIS board", and a reset is the same board
    2. adds no new tunable constant                    MET -- binary ablation flag
    3. mean over >=16 seeds is not negative            MET -- +0.0002

Fix A failed clause 3 at -0.0201, fix B at -0.0301, the candidate cap at -0.1091.
`ARC_KEEP_WALLS` now defaults to 1.

Claim nothing about the score: this is a correctness fix whose score effect was
measured and is nil.

Status: KEPT. WASTE HALVED, SCORE UNMOVED, ALL THREE CLAUSES MET.
