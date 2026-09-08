# Freeze the clock strip instead of cropping it — predeclared

## The defect

`Policy._levels` computes the credit signal over the whole grid:

    return [float(p.savings(grid, bg)) for p in PRIMITIVES]

No clock handling. The taboo that detects a dead click uses `clock_only`, which
is clock-aware, but the **value** signal that ranks controls is not. On any game
with a ticking budget strip, every click is credited with the clock's movement.

Measured on su15, one seed: 125 of 149 credited clicks carry an effect of exactly
**−2.0**. That uniformity is the strip, not the game.

su15 matters because its mechanic is fully decoded -- click the five-cell plus,
the avatar teleports -- so a failure there is a failure of selection, not of
understanding.

## Why the obvious repair is already dead

`Rejected #3` cropped the strip and measured worse (1.3368 → 1.1821). Its reason
is offline and seed-free, so re-measuring the number cannot rescue it:

    primitive     full   minus row0   minus col0
    Reflect         16           84           99
    Translate      584          560          583

Removing a row changes the grid's dimensions, and the primitives measure
geometry. Cropping is not noise removal; it is distortion.

## The intervention

Do not change the grid's shape. Overwrite the strip's cells with a **constant**
before computing savings, in both the current and previous grid. Dimensions
preserved, geometry preserved, the tick removed.

`ARC_FREEZE_CLOCK=1` to enable; default off until measured.

## Prediction and the control that can refute it

    PREDICT   games WITH a detected strip improve or hold: su15, ft09 (row 63),
              vc33 (row 0), r11l (col 0), lp85.

    CONTROL   games where ClockTracker finds NO strip must not change at all.
              From the decode table, m0r0 and dc22 have no strip recorded. If
              they move, the change is not doing what is claimed and the result
              is void whatever the aggregate says.

    GUARD     vc33 and r11l must not regress: Rejected #3 cost r11l 0.922 → 0.369
              and this must not repeat.

## Expected shape, before the run

Rejected #3's failure was caused by geometry change, which this avoids by
construction, so the r11l collapse should not recur. Whether removing the tick
*helps* is a separate question and I have no strong expectation. A null is the
most likely single outcome.

Measured at `repeats=1` -- competition conditions -- because
`THE_BENCH_MEASURES_ITS_OWN_HARNESS.md` established that repeats=3 inflates and
that a gain there can be worth exactly zero on Kaggle.
