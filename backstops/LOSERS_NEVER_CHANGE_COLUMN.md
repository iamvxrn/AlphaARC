# The losing seeds never move sideways, and staleness was the symptom

## Correction to the previous record

`STALENESS_CROSSES_OFF_THE_GOAL.md` located the mechanism that releases the
destination and asked the right follow-up: do winners beat the distance record
inside five steps, or do they simply start closer? Neither. The answer demotes
staleness from cause to symptom, and that correction is the point of this file.

## Established, from the traces already collected

All sixteen seeds start at `(16, 16)`, at Manhattan distance **66** from the goal
`(52, 46)`. So "the winners started closer" is rejected outright.

| | minimum distance reached | position at that minimum | distinct positions visited |
|---|---|---|---|
| 5 winning seeds | **0** | `(52, 46)` -- the goal | 34-39 |
| 11 losing seeds | **48**, every one | `(34, 16)` on 10 of 11 | 12-14 |

Eleven seeds stop at exactly the same number. That is not a timeout, whose
minima would scatter; it is a barrier.

`(16, 16) -> (34, 16)` is eighteen rows down and **zero columns across**. The
goal is at column 46. Of the thirty columns between, the losing seeds cross none,
on any seed, ever. They then spend 105-152 actions cycling among about thirteen
positions.

Staleness fires on the goal because nothing improves from `(34, 16)` -- the
distance cannot fall, so no record is beaten, so the counter runs out. Relaxing
the counter would buy more steps in the same place.

## Consistent with two things already in the log

`census_train.json` gives g50t no `translate` at all: every one of its five keys
reads `edit` or `noop` across the three probes. Movement there is conditional,
unlike ls20 whose four keys translate cleanly on every probe.

Rejected #9 recorded that a human cleared g50t level 1 with the **spacebar**, a
key `decode.py` calls dead because it changes nothing until the avatar has
arrived somewhere. A control that is inert until a precondition holds is exactly
what a fixed barrier looks like from the agent's side.

## The open question, now much narrower

Not "how does a program find the goal" -- it finds it, on every seed. Not "how
long should it persist" -- persistence at `(34, 16)` is worthless.

**What makes horizontal movement possible on g50t, and why do five seeds of
sixteen discover it?** The winners visit three times as many positions, so
whatever unlocks it is something they do and the others do not.

Cheapest next step, still no agent change: diff the action sequences of a winning
and a losing seed up to the point where the winner first changes column.

Status: STALENESS DEMOTED TO SYMPTOM. BARRIER LOCATED AT COLUMN 16.
