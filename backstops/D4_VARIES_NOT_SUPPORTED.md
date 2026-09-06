# D4: the control fires, and H1 is not supported

## What was predeclared, and when

Written into `SP80_M0R0_ZERO_FREEZE.md` and committed at `7ec7f49`, before the
train split was run:

> **Prediction:** if the cause is non-stationarity, `varies=True` should depress
> the score across all 17 train games, not only these four.
> **Control:** `click_density` should NOT split them the same way. If both split
> them equally, the effect is non-specific and H1 is not supported, however good
> the numbers look.

Data: full `train` split, 8 seeds, repeats=3, max_steps=250,
`python/bench/runs/d4-train`. `varies` and `click_density` from
`runs/census_train.json`, which predates all of this.

## The prediction landed

| | n | mean score |
|---|---|---|
| `varies=True` | 4 | **0.0000** |
| `varies=False` | 13 | 1.3847 |

All four `varies=True` games -- dc22, m0r0, re86, sp80 -- score exactly zero.

## And the control landed harder

| | n | mean score |
|---|---|---|
| `click_density` > 0.562 | 7 | **2.0676** |
| `click_density` <= 0.562 | 7 | 0.1882 |

Separation of 1.88 against `varies`'s 1.38. The control does not merely split
them as well; it splits them better.

**By the rule written before the run, H1 is not supported.**

## Why the first table is weaker than it looks

Nine of the seventeen games score exactly 0.0000. Only four of them are
`varies=True`. So:

    varies=True  ->  zero      4 of 4
    zero         ->  varies=True   4 of 9      the converse fails

Drawing four games at random from the seventeen and finding all four at zero has
probability 126/2380 = **0.053**. With n=4 against a base rate that high, "all
four are zero" is close to what chance supplies.

The simpler account covers both columns: the top scorers vc33, r11l and tn36 are
pure `click` games at `click_density` 1.0, and the agent is a click agent. 17 of
25 public games need keys and every win it has is click-only. `varies` and
`click_density` are both entangled with that tag, so on this sample neither can
be separated from it.

## What this licenses

Nothing about non-stationarity, in either direction. H1 is not rejected -- it is
**unsupported by this test**, which is a different status and is recorded as one.

It also says something about the method rather than the agent: no correlation
over these 17 games can settle H1, because every candidate variable is confounded
with the click/keyboard split. Deciding it needs an intervention on the evidence,
not another correlation.

The result of the day stands elsewhere: sp80 puts 67 of its 251 actions through
the planner and spends the rest clicking (`SP80_BUDGET_SPLIT.md`). That is a
measured mechanical fact, not a correlation, and it does not depend on H1.

Status: PREDICTION MET, CONTROL MET EQUALLY, HYPOTHESIS UNSUPPORTED.
The control was worth the run: without it this would have been filed as evidence.
