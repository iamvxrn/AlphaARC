# g50t is a gated door, and the agent's world model has no gates

## The separation is binary, not statistical

Furthest position reached, 16 seeds, 3 repeats each:

    5 seeds that clear level 1     max row 52, max col 52   -- every one
    11 seeds that do not           max row 34, max col 40   -- every one

Two discrete outcomes, no spread. All sixteen start at `(16, 16)`; the goal is
`(52, 46)`, outside the losing box on both axes. The avatar is a stable 24-cell
block in 741 of 743 readings, matching the README's description, so the position
trace is measuring the avatar and not a blend.

## The barrier is real, and it is conditional

Every winner breaks out the same way -- consecutive `k2` from `(16, 16)` through
`(22,16)`, `(28,16)`, `(34,16)` and onward to `(40,16)`, `(46,16)`.

Losers press `k2` **while standing on `(34, 16)` between 13 and 17 times** and the
avatar does not move once. Some of those presses are recorded as changing the
board -- the world responds, the avatar stays.

Run length is not the difference. Losing seeds produce `k2` runs of 5, 6 and 7
presses repeatedly; seeds 1, 4 and 9 each have five or more such runs. Pressing
the key more does not open the door.

## Correction

Two records ago this was written up as "the losing seeds never change column".
That is wrong: losers do move sideways, reaching `(10, 34)` early. The correct
statement is the box above. The claim was made from two seeds and the sixteen-seed
table replaced it.

## What it means

g50t contains a door that is shut until something else is done, and Rejected #9
already met it from the other side: a human cleared this level with the spacebar,
a key `decode.py` calls dead because eight presses from a fresh board change
nothing -- *until the avatar has arrived*.

The agent's model has no place to put that. It learns a key->offset map and
consumes it as a stationary fact, so a key that works only in some world-state is
recorded as a key that works, and its failures look like noise. Thirteen presses
into a shut door are not diagnosed as a precondition; they are spent.

This is the first direct mechanical instance of the non-stationarity that D4
could not support correlationally. D4 remains unsupported -- correlation over 17
games could not separate it from the click/keyboard split -- and this record does
not revive it as a general claim. It shows one door, on one game.

## Open

What opens it, and why five seeds of sixteen open it. The breakthrough steps are
39, 272, 272, 272 and 520 -- three at exactly the same step, which is worth
explaining before anything is built.

Status: BARRIER CONFIRMED AND CONDITIONAL. NO MECHANISM LICENSED.
