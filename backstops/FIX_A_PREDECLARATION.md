# Fix A, predeclared before it is written

## The measured defect

`HybridPolicy.board_replaced` sets `switched = False`, returning control to the
one-step click policy. `agent_glue` calls it with two different reasons and the
method treats them alike:

    board_replaced("level_clear")   a genuinely new board
    board_replaced("reset")         the SAME level again, after GAME_OVER

The switch to the planner needs `since_level > 25` or 20 dead clicks. sp80 resets
**8 times**, so the cheap policy is handed roughly 8 x 25 = 200 actions. Measured
policy decisions on sp80: 168 of 251, over 12 candidates, none of them keys.

## The intervention

Retain the switch across a `reset`, release it on a `level_clear`. Nothing else.

## Prediction, and the control that can refute it

    PREDICT    sp80's planner share rises and its score is free to move.
               It is the only game with more than one reset.

    CONTROL    ls20 and g50t must NOT change. Both are keyboard-only, so the
               adapter routes keys to the planner regardless, and both reset
               once. If they move, the effect is not the one claimed.

    GUARD      vc33 and lp85 must not regress. R9's fix looked neutral on the
               keyboard games and cost these two 0.28 each.

Reset counts, from the traces: sp80 8, m0r0 1, ls20 1, g50t 1.

A score change on sp80 is not required for the fix to be correct -- the claim is
about where the budget goes. If the planner share rises and the score does not,
that is a real result and says the budget was not the binding constraint either.

Paired, 8 seeds, against `runs/loop-base`.
