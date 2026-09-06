# The keys are never candidates on the path that owns the budget

## The discriminator, and its answer

Two explanations for sp80 spending its budget on clicks:

1. clicks out-rank the keys in the agent's scoring
2. the key path is blocked, and clicks are what is left

Answered from the trace already collected. No agent change, no new run.

sp80 has two decision paths. `planner` fired 67 times, mostly keys. `policy`
made **168 decisions** -- two thirds of the 251-action budget -- and its
candidate list is:

    sp80   168 policy steps, 12 distinct candidates, keys among them: ZERO
    m0r0    50 policy steps,  7 distinct candidates, keys among them: ZERO

**Explanation 1 is rejected.** The keys do not lose the ranking. They are never
in it. On the path that spends most of the budget, a key cannot be chosen because
it is never offered.

## What the keys did on the path where they are offered

| control | presses | wasted |
|---|---|---|
| k1 | 28 | 20 |
| k4 | 21 | 0 |
| k2 | 4 | 0 |
| k3 | **1** | 0 |

Census says ACTION2/3/4 translate reliably on sp80 and ACTION1 does not. The
agent spends 28 of its 54 key presses on the unreliable one and one press on k3.
So there is a second, smaller misallocation *inside* the keys -- but it is worth
at most the 54 presses that path controls, not the 168 the other one does.

## Consequence for the proposed fix

The owner proposed probing at the start of a game and then budgeting toward
whatever pays. That is a fix for explanation 1, and explanation 1 is now
rejected: a priority cannot be applied to a control that is absent from the
candidate set. The fix has to be upstream of ranking -- either the keys enter
`policy`'s candidates, or the split of the budget between the two paths is
decided rather than emergent.

Which of those, and whether either helps, is not settled here and no mechanism is
licensed by this record.

Status: EXPLANATION 1 REJECTED. THE DEFECT IS STRUCTURAL, NOT A RANKING.
