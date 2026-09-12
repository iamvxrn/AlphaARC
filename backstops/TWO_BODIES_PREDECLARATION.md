# Two bodies: the goal is a relation, not a place — predeclared

## What the owner saw, and what the traces confirm

m0r0 level 1 shows two mirrored halves with one marker in each. The task is to
bring the two markers together. Confirmed from the trace, seed 1:

    two bodies present on 190 of 190 steps
    mirrored: the midpoint between them sits at ~30 the whole run
    distance between them: starts 20, reaches 10, ends 35

The agent halves the distance by accident and then wanders back out. It has no
notion that closing that distance is the task.

This explains a standing puzzle: every destination detector returned **zero**
candidates on m0r0, including the enclosed-region one. There is no goal region
because the goal is not a region. It is `distance(bodyA, bodyB) == 0`.

Mirroring helps rather than hurts: one key moves the pair toward each other, so
the distance closes at double rate. Columns 21 and 41 become 26 and 36 in one
press.

## The intervention

When the avatar colour forms exactly two components, steer the first toward the
second: the destination is the other body's centroid, refreshed each step because
it moves too. Everything else -- the offset map, the BFS, the wall memory -- is
reused unchanged.

`ARC_TWO_BODIES=1`; default off until measured.

## Prediction and the control that can refute it

    PREDICT  m0r0's inter-body distance reaches 0, or at least a minimum well
             below the 10 it reaches by accident today.

    CONTROL  games whose avatar colour forms ONE component must not change at
             all -- ls20, g50t, sp80. The branch is gated on exactly two
             components, so any movement there means the gate is wrong and the
             result is void.

    GUARD    vc33, r11l, tn36, lp85 unchanged at 16 seeds, repeats=1.

## Expectation before running

Reaching distance 0 is not the same as completing the level -- the level may
require them to meet somewhere specific, or to meet and then do something. So a
plausible outcome is "distance closes, level still not completed", which would
still be progress and would locate the remaining step.

Measured at repeats=1, competition conditions.
