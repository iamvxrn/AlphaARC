# Two-body steering: m0r0 goes from 0 of 16 seeds to 16 of 16

Predeclared at TWO_BODIES_PREDECLARATION.md. 16 paired seeds, `repeats=1`, full
train split.

| game | off | on | delta | se | seeds completing |
|---|---|---|---|---|---|
| **m0r0** | 0.0000 | **0.3868** | **+0.3868** | 0.0224 | **0 -> 16** |
| ar25 | 0.3558 | 0.3525 | -0.0034 | 0.0033 | 7 -> 6 |
| g50t, lf52, lp85, ls20, r11l, re86, tn36, vc33 | | | **+0.0000** | 0 | unchanged |

    aggregate  0.4800 -> 0.5026   +0.0226 +/- 0.0013   17 standard errors

## What it is

The first game in this investigation to leave zero, and the first measured score
gain. Nine games scored exactly nothing on every seed of every run; this is one
of them, now completing on all sixteen.

## Why it worked

Not a new destination rule. The blocker was **upstream** of the goal: the router
was called 200 times and returned a route 6 times, because it had learned two
offsets and both were vertical. A body offset horizontally is unreachable by
vertical moves, so no goal of any kind could have helped.

The horizontal key sends one marker `(0,-5)` and the other `(0,+5)`. Their union
is not a translation, `rigid_body` records nothing, and the key stays unlearned.
Per body it is an ordinary translation.

So the repair is two halves, and only the first is interesting: learn the offset
from one body; then aim it at the other. Offsets learned 2 -> 4, routes 6/200 ->
114/201.

## The control

Every single-component game is `+0.0000` on every seed, which is what the gate
predicts. `ar25` is the exception at -0.0034 +/- 0.0033 -- one standard error,
inside the noise, and it drops one seed of sixteen. It presumably resolves into
two components sometimes. Recorded rather than dismissed.

## Provenance

The read came from the owner looking at the board and saying there are two
players who must be brought together. Every measurement here followed from that
sentence. The agent's own detectors had returned zero candidates on this game for
weeks, which is what a goal that is a *relation* rather than a *place* looks like
from inside a system that only searches for places.

Status: KEPT, DEFAULTED ON, LINEAGE AT lineage/two-bodies.md, STATUS REQUIRED.
