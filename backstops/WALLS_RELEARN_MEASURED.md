# The relearning is real, measured directly rather than inferred

The predeclaration at 66af896 argued from arithmetic: sp80 hits `k1` from
`(12,12)` 126 times over 18 plays, which is 7 per play against 8 resets per play,
so each death must be forgetting the wall. Plausible, and not measured.

Measured now, same seed, flag off against on, sp80, 3 repeats:

| | resets | misses | distinct points | worst point |
|---|---|---|---|---|
| walls forgotten | 24 | **80** | 4 | **21** |
| walls kept | 24 | **36** | 13 | **3** |

Misses fall by 55%. The worst single point falls from 21 hits to 3.

Distinct failure points **rise**, 4 to 13, and that is the right sign: freed from
hammering four walls, the agent reaches further and finds others. The budget that
was going into relearning goes into coverage.

The predeclared claim -- "repeated always-miss points on sp80 and m0r0 fall
sharply" -- is met on the measurement it was written for.

The score claim is separate and still running: 17 games, 16 paired seeds,
`repeats=1`. The expectation recorded before the run was a null on score, since
fix A was this same confusion in another field and moved its target exactly as
designed while the score did not move at all.

Status: WASTE CLAIM CONFIRMED BY DIRECT MEASUREMENT. SCORE CLAIM PENDING.
