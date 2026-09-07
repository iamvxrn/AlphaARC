# ARC_MAXCAND sweep, predeclared before it is run

## The measured defect this addresses

`python/bench/README.md`, Rejected #5:

> | game | what fills all 8 slots | live among them |
> | su15 | 20 one-cell dots of a dashed diagonal path | **0** |
> | s5i5 | 10 pixels shattered off diagonal plus-markers | **0** |
> | lf52 | 8 corner pixels of the `1EE1` tiles | 6, all one-shot |

The candidate generator ranks smallest-first. On su15 and s5i5 the board carries
many equal-smallest components which take all eight slots, and **not one of the
eight offered candidates does anything**. su15's real control is a five-cell plus
that teleports the avatar; it is never offered.

`agent_glue.py` already exposes the knob and states the principle:

> `ARC_MAXCAND`: the candidate cap, and nothing else. Raising it adds a tail
> without permuting the head -- the one thing Rejected #5 said a candidate change
> must not do.

Rejected #5 itself is one of the three conclusions the research map flags as
**single-seed** and unre-measured, on a split whose spread is sd 0.83.

## The sweep

`ARC_MAXCAND` in {8 (baseline), 16, 24}, paired on 16 identical seeds,
`repeats=3 max_steps=250`.

Set: `su15`, `s5i5` (probe), `lf52` (control), `vc33`, `lp85` (guard).

## Prediction, and the control that can refute it

    PREDICT   su15 and s5i5 improve. Their live controls are documented as
              sitting outside the top 8, so a longer tail is the only thing
              that can put them on the ballot.

    CONTROL   lf52 must NOT improve proportionally. It already has 6 live
              candidates among its 8 slots, so a longer tail adds little it
              did not already have. If lf52 gains as much as su15, the effect
              is capacity or noise, not the mechanism claimed.

    GUARD     vc33 and lp85 must not regress at 16 seeds.

## Expected shape, stated before the run

Not monotone. Every extra candidate costs a probing press, and the score punishes
wasted actions quadratically -- the same arithmetic that made Rejected #8's
budget increase worthless. So a larger cap buys reach and spends score, and 24
may well be worse than 16. An optimum, not a ramp.

I expect a small positive at 16 and a possible negative at 24. Recording that so
a null cannot be presented as a surprise afterwards.

## What passing licenses

Nothing about the destination rule, the door, or the memory ceiling. This is a
reach fix on a measured blindness, and its whole claim is that a control which
does something should be visible to the thing choosing controls.
