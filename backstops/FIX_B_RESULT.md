# Fix B: refuted for sp80 by its control, and not kept on the rule

## 16 paired seeds against `runs/loop-base`

| group | baseline | fix B | delta | se | verdict |
|---|---|---|---|---|---|
| KEYBOARD | 0.3348 | 0.3048 | -0.0301 | 0.1137 | inside the noise |
| CLICK | 4.7378 | 4.7305 | -0.0073 | 0.0215 | inside the noise |

| game | delta | worse/better seeds |
|---|---|---|
| ls20 | **-0.1556** | 4 / 5 |
| vc33 | -0.0491 | 2 / 0 |
| g50t | +0.0353 | 1 / 1 |
| lp85 | +0.0345 | **0 / 9** |
| sp80 | +0.0000 | 0 / 0 |
| m0r0 | +0.0000 | 0 / 0 |

## Verdict

The predeclared control already refuted the diagnosis for sp80: the effect had to
be largest there and sp80 is byte-identical while ls20 moved.

On the keep rule, clause 3 -- "measured mean over >=16 seeds is not negative" --
fails in both groups. `ARC_LIVE_FALLBACK` defaults to 0; the instrument stays,
the behaviour does not ship.

One detail worth keeping in view rather than acting on: lp85 improves on 9 of 16
seeds and worsens on none. That is a consistent direction inside a group mean
that is negative, and it is not a result. If the fallback is revisited, lp85 is
where to look first.

## Where sp80 stands after today

Eliminated, each by measurement and each with a predeclared control:

    budget too small                D1, 8x actions, 48 plays, zero levels
    clicks out-rank the keys        trace: keys are not in the policy's candidates
    budget split between paths      fix A, planner share 67 -> 209, score unmoved
    fallback re-draws dead controls fix B, refuted by its own control on sp80

Located but not repaired: `_route` returns a key from the key->offset map without
consulting that key's reliability. sp80's k1 delivers its offset on one census
probe of three and takes 28 of 54 key presses, wasting 20.

Nothing shipped. Two arms measured at 16 seeds, both defaulted off on the rule.

Status: FOUR EXPLANATIONS ELIMINATED. NO MECHANISM KEPT.
