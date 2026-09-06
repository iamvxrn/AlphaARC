# sp80 spends three quarters of its budget clicking, and hindsight is not its problem

## Why this was measured

The owner described how a person plays these games: explore, lose, and work out
the goal retrospectively. That is Hindsight Experience Replay, which exists in
this repository at `pkg/memory/hindsight.go` and is absent from the Python agent
that actually scores. Before porting it, its premise was checked: does the agent
reach anything a relabeller could learn from?

Traces only. `ARC_TRACE` is opt-in, read by nothing in the decision loop, and no
agent code was changed. One seed, one repeat, 250 actions, per game.

    ARC_TRACE=/tmp/tr/<game>.jsonl bench.py --games <game> --max-steps 250 \
        --repeats 1 --seed 1

## What the traces say

| game | tag | planner firings | click-credit records | resets | write-offs |
|---|---|---|---|---|---|
| **sp80** | keyboard_click | **67** | **327** | **8** | 9 |
| m0r0 | keyboard_click | 198 | 98 | 1 | 14 |
| ls20 | keyboard | 248 | 0 | 1 | 4 |
| g50t | keyboard | 248 | 0 | 1 | 5 |

All four spend the same 251 actions. `ls20` and `g50t` put 248 of them through
the planner. **`sp80` puts 67.**

sp80 is `keyboard_click`, so it has both kinds of control, and the 327
click-credit records say where the rest of the budget went: the agent spends most
of sp80 clicking. Its decoded mechanic is `object-translation` -- ACTION2, 3 and
4 move a body by 162, 34 and 34 cells. It is a movement game and the agent is
playing it as a clicking game.

It also resets **8 times** against one everywhere else.

## And a recorded conclusion needs correcting

`python/bench/README.md` says of sp80: *"The agent so rarely returns to the same
place that there is nothing to condition on. Its bottleneck is reach, not
representation"* -- one repeated `(control, position)` pair in a whole run.

Keyed on **state** rather than position, sp80 is the opposite. 67 firings over
**12 distinct states**, 55 of them repeats, the most common state entered 13
times. It is not failing to return anywhere. It circles a dozen states.

| game | firings | distinct states | repeats | new |
|---|---|---|---|---|
| sp80 | 67 | 12 | 55 | 17.9% |
| m0r0 | 198 | 109 | 89 | 55.1% |
| ls20 | 248 | 51 | 197 | 20.6% |
| g50t | 248 | 197 | 51 | 79.4% |

"Reach is the bottleneck" was a conclusion about the position key, not about the
agent. Note also that `ls20`, which scores best of the four, has the *second
lowest* state diversity -- so wide exploration is not what distinguishes a game
that scores.

## What this licenses, and what it does not

**Hindsight is not sp80's binding constraint, and porting it now would be
premature.** Nothing can relabel goals in a budget that never reaches the
decision loop. The premise check returns "not yet" rather than "no": HER may
still matter once sp80 spends its actions where its mechanic is.

This does not diagnose sp80. Two obvious explanations for the split are untested
and pull in different directions: the click candidates may be genuinely
out-competing the keys in the ranking, or the key path may be blocked and clicks
are what is left. One measurement cannot tell them apart, and no mechanism is
licensed until it can.

m0r0 is untouched by this finding -- it fires 198 of its actions and explores 109
states. The two remain separate failures, as frozen.

Status: PREMISE CHECK RETURNS "NOT YET". ONE RECORDED CONCLUSION CORRECTED.
NO MECHANISM LICENSED.
