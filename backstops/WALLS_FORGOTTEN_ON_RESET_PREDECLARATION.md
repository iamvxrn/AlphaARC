# Walls are forgotten on every reset — predeclared before the fix

## Measured

The displacement check (new, trace-only) splits the misses cleanly:

    sp80   478 misses, 100% "did not move"        a wall
    m0r0    36 misses, 100% "did not move"        a wall
    re86   508 misses,  90% "moved to the WRONG place"

re86 is a different defect: `k3` from `(3,6)` expects `(3,3)` and lands on
`(24,7)`; `k4` from `(24,7)` expects `(24,10)` and lands back on `(3,6)`. The
body teleports between two locations, which is an avatar-identification problem
like m0r0's mirrored pair, not a wall. Out of scope here.

`RunPlanner` already models walls correctly: `blocked` holds `(position, key)`
pairs and `_plan`'s BFS skips those edges. The mechanism exists and is right.

It is erased by `board_replaced`, whose comment says "walls are a property of
THIS board". But a **reset** hands back the same level, where the walls are the
same; only a level **clear** is a new board.

The arithmetic matches: sp80 hits `k1` from `(12,12)` 126 times over 18 plays,
which is 7 per play against 8 resets per play. Each death forgets the wall it
had just learned and relearns it.

## The intervention

Retain `blocked` across a `reset`, clear it on a `level_clear`. One condition.
`ARC_KEEP_WALLS=1`; default off until measured.

## Prediction and the control that can refute it

    PREDICT  repeated always-miss points on sp80 and m0r0 fall sharply. This is
             the claim, and it is about wasted presses, not score.

    CONTROL  ls20 and g50t reset once per play, so they must barely change. If
             they move as much as sp80, the effect is not the one claimed.

    GUARD    vc33 and lp85 must not regress at 16 seeds, repeats=1.

## Expectation, recorded before running

Fix A was this same confusion in a different field -- `HybridPolicy.switched` --
and it moved the budget exactly as designed while the score did not move at all.
I expect the same here: fewer wasted presses, and no score change, because
`sp80`'s problem was never that it lacked actions. A null on score is the most
likely single outcome and is not a surprise.

Measured at `repeats=1`, competition conditions.
