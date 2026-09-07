# Five steps without a record and the goal is crossed off

## Established

**`(52, 46)` is g50t's goal, confirmed by outcome.** Release reasons for that
destination, 16 seeds, 3 repeats:

| | arrived | unreachable | stale |
|---|---|---|---|
| the 5 seeds that clear level 1 | **8** | 20 | 29 |
| the 11 that do not | **0** | 56 | 112 |

Every winning seed arrives there at least once. No losing seed ever arrives.

**Staleness is how the goal is usually lost** -- 141 of 225 goal releases, and
112 of the 168 on losing seeds.

**The threshold is small.** `len(self.moves)` on g50t is 5, and `stale` fires at
exactly 5 in 802 of 886 cases.

## The mechanism

    if self._dest_best is None or here < self._dest_best:
        self._dest_best, self._dest_stale = here, 0
    else:
        self._dest_stale += 1
        if self._dest_stale >= len(self.moves):
            self._release("stale")

The counter resets only on a **strict improvement over the best distance ever
recorded** for this destination -- not on progress, not on holding station. So
five consecutive steps that fail to beat the record cross the destination off,
and `crossed_off` then bars it until every other candidate is crossed off too.

Going around an obstacle cannot beat the record while going around: a detour does
not reduce Manhattan distance, that is what makes it a detour. g50t's goal room
is open on its left side only (README, "the movement class, second pass"), so it
has to be approached from one direction.

Seeds 9, 10 and 11 are the shape of this: 11-12 stale releases on the goal each,
one unreachable, and zero arrivals. They find the right room repeatedly and are
timed out of it every time.

## Not established

That relaxing the rule helps. `len(self.moves)` is already a constant chosen by
something else -- the number of learned directions -- and any replacement is a
new tunable, which the keep rule says needs a sweep, and a sweep needs resolution
this bench may not have.

Three cheaper things to know first, none of which touch the agent:

1. Does the avatar actually have to detour on g50t, or is the room reachable on a
   monotone path? If monotone, staleness is not the blocker and this record is a
   coincidence of counts.
2. What does `_plan` return during those five steps -- is it steering into the
   wall it recorded in `blocked`?
3. Do the winning seeds beat the record inside five steps, or do they arrive
   before the counter can run out?

Question 3 is the sharpest: it separates "the rule is too strict" from "the
winners simply started closer".

Status: MECHANISM LOCATED AND MEASURED. NO MECHANISM LICENSED.
