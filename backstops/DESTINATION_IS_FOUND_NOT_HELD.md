# The goal is identified. It is abandoned every ten steps.

## How this was reached

Not by inventing a sixth destination detector. By asking why g50t clears level 1
at exactly 33 actions on four seeds of sixteen -- a repeatable number that random
wandering does not produce -- and then following the measurements.

Two hypotheses died on the way, both cheaply:

1. **The router is gated on learning the key->offset map.** Refuted: the map is
   populated by step 2 on all 16 seeds, and the router returns a route on 493 of
   750 calls on winning seeds against 465 on losing ones. It is never gated.
2. **Fix B's diagnosis of the wasted presses.** Refuted by its own predeclared
   control.

All instrumentation is behaviour-neutral trace output, which `GROWTH_PROTOCOL.md`
exempts.

## Established

**The router names the right place.** g50t's goal room is rows 48-56, cols 42-50
(README, "the movement class, second pass"). The router's single most-chosen
destination, on **all sixteen seeds**, is `(52, 46)` -- the centre of that room.
Its two nearest rivals, `(5, 2)` and `(5, 6)`, are the top-left legend.

**And it does not stay on it.** Per run of ~750 router calls:

    destination changes           68 - 78
    longest unbroken hold on the goal   9 - 17 steps
    total steps aimed at the goal       47 - 114

The destination is re-chosen roughly every ten decisions, and is never held for
more than seventeen consecutive ones.

## Not established

That holding longer would help. The distributions overlap:

    won  (5 seeds)   longest hold 11, 12, 12, 12, 17
    lost (11 seeds)  longest hold  9,  9,  9,  9,  9, 11, 11, 12, 13, 13, 13

Every winner is at 11 or above and five losers sit at exactly 9, but losers at 12
and 13 exist and a winner sits at 11. At n=16 this is a suggestion, not a cause.
Note also that losing seeds spend *more* total steps aimed at the goal than
winning ones -- seed 9 spends 114, winning seed 7 spends 47 -- so the quantity of
attention on the goal is not what separates them either.

## Why this matters more than the number

Five candidates for the destination rule were closed by measurement: imagined
compression, the five-cell plus, enclosure, conditioning on position, and
conditioning on body configuration. Every one of them attacks **identification**
-- how to look at a board and say where the goal is.

Identification measurably works. On g50t the router picks the goal room more
often than anything else, on every seed, and eleven of sixteen still fail.

So the open problem is not the one the last five experiments were solving. It is
persistence: what should make a chosen destination survive the next ten steps.

## The next intervention needs its own predeclaration

Forcing a hold is a mechanism, and this record does not license one. A hold has a
duration, which is a new tunable constant, which the keep rule says needs a sweep
and a sweep needs resolution this bench does not obviously have. State that
before building anything.

Status: IDENTIFICATION ESTABLISHED AS WORKING. PERSISTENCE NAMED AS THE OPEN
PROBLEM. NO CAUSE ESTABLISHED, NO MECHANISM LICENSED.
