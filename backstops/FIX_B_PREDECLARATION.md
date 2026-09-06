# Fix B, predeclared before it is written

## The defect

`RunPlanner.choose` excludes written-off controls everywhere it selects, except
in its last branch:

    unprobed = [t for t in targets if sigs[t] not in self.profiles
                                   and sigs[t] not in self.inert]      excludes
    escalate = [... and sigs[t] not in self.inert ...]                 excludes
    scored   = [... for t in targets if sigs[t] not in self.inert]     excludes
    else:    target = self.rng.choice(targets)                         DOES NOT

So once nothing known pays -- the common case on a game that is not being solved
-- selection samples uniformly over every target including the ones already
written off as dead.

Measured on sp80, one seed: k1 is pressed **28 times and wastes 20 of them**,
while k4 (21 presses, 0 wasted) and k3 (1 press, 0 wasted) do not. k1 goes inert
on its first dead press and keeps being re-drawn by this branch.

## The intervention

Sample the fallback from the non-inert targets, and only fall back to the full
list if every target is inert. One line. No new constant.

## Prediction, and the control that can refute it

    PREDICT   wasted key presses on sp80 fall; k1's share of presses drops.
              This is the claim, and it is about presses, not score.

    CONTROL   a game that writes off nothing must not change. From the traces,
              write-off counts are sp80 9, m0r0 14, ls20 4, g50t 5 -- none is
              zero, so the control is instead: the effect must be LARGEST on
              sp80, which wastes the most. If ls20 moves more than sp80, the
              effect is not the one claimed.

    GUARD     vc33 and lp85 must not regress at 16 seeds.

## Expected outcome, stated before the run

I expect the press distribution to change and the score not to. Fix A gave the
keys three times the budget and produced no level, so redistributing within the
key budget is unlikely to produce one either. Recording that expectation here so
that a null result cannot later be presented as a surprise, and a positive one
cannot be presented as expected.

Unlike fix A this touches every game, not only those that reset, so the guard
matters more.

Paired, 16 seeds, against `runs/loop-base`.
