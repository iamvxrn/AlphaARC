# Fix A does exactly what it claimed, changes no score, and is not kept

## Predeclared at 9e946c6, before the code

> PREDICT sp80's planner share rises. CONTROL ls20 and g50t must NOT change.
> GUARD vc33 and lp85 must not regress. A score change is not required -- if the
> planner share rises and the score does not, that says the budget was not the
> binding constraint either.

## The prediction landed exactly

    sp80, one seed, 251 actions

                  planner    click path    resets
    baseline           67           168         8
    fix A             209            25         8

The click path drops to 25 -- exactly one hand-over window, the opening before
the first switch. The budget moved where the mechanic is.

## The score did not move at all

16 paired seeds against `runs/loop-base`:

| group | baseline | fix A | delta | sd | verdict |
|---|---|---|---|---|---|
| KEYBOARD | 0.3348 | 0.3348 | **+0.0000** | 0.0000 | IDENTICAL |
| CLICK | 4.7378 | 4.7177 | -0.0201 | 0.0780 | inside the noise |

Per game, `g50t`, `ls20`, `sp80`, `m0r0` and `vc33` are identical on all 16
seeds. Only `lp85` moves, -0.0403, on 1 seed of 16. The control held: the two
keyboard-only games did not change, as predicted.

## What this establishes

**Budget allocation between the two paths is not sp80's binding constraint.**
The keys were given roughly three times the budget and sp80 still completes no
level on any seed. This was the predeclared reading of exactly this outcome.

It also answers the owner's proposal -- probe each new game, then budget toward
whatever pays. That is a priority scheme, and this is the priority scheme's best
case: the keys took 209 of 251 actions. Nothing happened.

## Why it is not kept

`python/bench/README.md`, "When a change is not measurable, keep it only on these
grounds", requires all three:

    1. fixes an inconsistency between code and claim    MET -- the docstring says
       "a new level is a fresh chance" and a reset is not a new level
    2. adds no new tunable constant                     MET -- a binary ablation
       flag, matching ARC_CLOCK_DEADRUN, nothing to sweep
    3. measured mean over >=16 seeds is not negative    FAILED -- CLICK -0.0201

lp85 resets twice, so the change alters it legitimately rather than by accident.
Clause 3 still fails, so `ARC_KEEP_SWITCH` defaults to 0 and the behaviour does
not ship. The instrument stays so the arm can be re-run.

Keeping it anyway was available and was not taken. The rule exists because,
quoting the file, otherwise "principled quietly becomes a licence to ship
anything."

Status: PREDICTION MET, CONTROL HELD, SCORE UNMOVED, CHANGE NOT KEPT.
Budget allocation is eliminated as sp80's cause.
