# Freezing the clock is a wash, and the strip turns out to be load-bearing

Predeclared at 961a01a. 16 paired seeds, full train split, `repeats=1` --
competition conditions.

| game | baseline | frozen | delta | se |
|---|---|---|---|---|
| r11l | 1.1189 | **2.3470** | **+1.2281** | 0.5411 |
| tn36 | 1.8996 | 1.1701 | −0.7295 | 0.2630 |
| vc33 | 4.0930 | 3.2514 | −0.8415 | 0.5502 |
| lf52 | 0.0039 | 0.0019 | −0.0020 | 0.0019 |
| **su15** | 0.0000 | **0.0000** | +0.0000 | 0.0000 |

Aggregate over 17 games: 0.4799 → 0.4596, **−0.0203 ± 0.0514**, inside the noise.

## The control passed

`m0r0` and `dc22` have no strip in the decode table and changed on **0 of 16
seeds**. So the intervention touches exactly the boards it claims to touch, and
the numbers above are its effect rather than a side effect.

## The target did not move

`su15` is 0.0000 before and after, on every seed. The diagnosis that started this
-- 125 of 149 credited clicks carrying exactly −2.0 from the strip -- was correct
as an observation and **wrong as a cause**. Removing the tick does not let the
agent find the plus. Whatever stops su15 is upstream of the credit signal.

## What was learned instead

The strip is **load-bearing, not noise**. Freezing it doubles `r11l` and costs
`tn36` and `vc33` roughly 0.8 each. A quantity that can move a game either way by
that much is participating in the primitives' geometry, not sitting on top of it
as a nuisance.

That is the same conclusion Rejected #3 reached by cropping, arrived at from the
opposite direction and with the geometry objection removed by construction. Two
different removals, two different distributions of harm, no net gain either time.

`r11l` +1.2281 is the largest single-game gain measured in this repo at
competition conditions and it is not usable on its own, because it is paid for
twice over by tn36 and vc33. Whether the gain can be had without the losses --
whether "which primitive should see the strip" is a per-primitive question rather
than a per-board one -- is open and not answered here.

## Verdict

Aggregate is negative, so clause 3 of the keep rule fails. `ARC_FREEZE_CLOCK`
stays defaulted off. The instrument remains for the per-primitive question above.

Status: WASH ON AGGREGATE. CONTROL CLEAN. TARGET UNMOVED, ITS DIAGNOSIS REFUTED.
THE STRIP IS LOAD-BEARING.
