# The candidate cap is rejected on all three predeclared counts

Predeclared at 428b707 before the run. 16 paired seeds, `repeats=3`.

| game | role | cap 8 | cap 16 | cap 24 | 16 − 8 | se |
|---|---|---|---|---|---|---|
| su15 | probe | 0.0000 | 0.0000 | 0.0000 | **+0.0000** | 0.0000 |
| s5i5 | probe | 0.0000 | 0.1795 | 0.0655 | +0.1795 | 0.1678 |
| lf52 | **control** | 0.0270 | 0.1881 | 0.1029 | **+0.1612** | 0.1104 |
| vc33 | guard | 8.9847 | 8.3397 | 8.5390 | −0.6450 | 0.8673 |
| lp85 | guard | 0.4910 | 0.2500 | 0.2936 | **−0.2410** | 0.0579 |

Whole set: −0.1091 ± 0.1967 at cap 16, −0.1003 ± 0.1807 at cap 24. Both negative.

## Against what was predicted

**The probe did not move.** `su15` is exactly 0.0000 at every cap, on every seed.
Tripling the candidate list changed nothing there at all -- so either its live
control sits beyond rank 24, or being offered is not sufficient. `s5i5` moved
+0.1795 against se 0.1678, about one standard error, which carries nothing.

**The control fired.** `lf52` gained +0.1612, as much as `s5i5` and with a tighter
error, although it already had 6 live candidates among its 8 slots and a longer
tail should have added it little. By the rule written before the run, the effect
is capacity or noise, not "the live control is now on the ballot".

**The guard failed.** `lp85` loses 0.2410 at se 0.0579 -- 4.2 standard errors, a
real regression, not a wobble.

My own predeclared expectation was also wrong: I wrote "a small positive at 16
and a possible negative at 24" and the set is negative at both.

## What this settles

`Rejected #5` said re-ordering the click candidates was worse, and the research
map flagged it as one of three conclusions measured on a **single seed** with sd
0.83 underneath. This is that re-test, at 16 paired seeds, on the knob the code
itself argued was the safe version -- adding a tail without permuting the head.

It holds. One of the three flagged single-seed conclusions is now properly
measured and stands. Two remain: Rejected #3 and #4.

## What it does not settle

Why `su15` scores zero. The measured fact that none of its eight offered
candidates does anything is unchanged and unexplained; this only shows that
offering more of them is not the repair.

Status: REJECTED ON PROBE, CONTROL AND GUARD. REJECTED #5 CONFIRMED AT 16 SEEDS.
