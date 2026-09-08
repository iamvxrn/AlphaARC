# The second-largest recorded gain is exactly zero under competition conditions

## What the leaderboard said

    2026-08-31   0.19    agent at 59b1508, before ARC_CARRY
    2026-09-07   0.13    agent with ARC_CARRY

Both far below the ~0.42 the submit checklist projected from local numbers.

## What the competition actually does

`vendor/ARC-AGI-3-Agents/agents/swarm.py`:

    for i in range(len(self.GAMES)):
        g = self.GAMES[i % len(self.GAMES)]

One agent per game, each game played **once**. There are no repeats.

`ARC_CARRY` carries the learned mechanic across the *repeats* of a game. With no
repeats there is nothing to carry.

## Measured, 16 paired seeds, scoring subset

| condition | score |
|---|---|
| carry=1, repeats=3 | 3.8229 |
| carry=0, repeats=3 | 2.6091 |
| carry=1, repeats=1 | **1.8416** |
| carry=0, repeats=1 | **1.8416** |

    ARC_CARRY at repeats=3   +1.2137 +/- 0.2067
    ARC_CARRY at repeats=1   +0.0000 +/- 0.0000     exactly, on all 16 seeds

The recorded `+1.2137 (sem 0.2135)` reproduces to four decimals. That measurement
was correct. It measured a condition the competition does not run.

## And the harness inflates on its own

The bench scores a game by the **best of three runs**; the competition scores one.

    best-of-three markup, carry off   +0.7675 +/- 0.1431
    best-of-three markup, carry on    +1.9813 +/- 0.1665

So the usually-quoted 3.82 is 1.84 under competition conditions. Roughly a factor
of two, before any question of the hidden set being harder.

## What this invalidates, and what it does not

**Does not invalidate** past A/B comparisons as comparisons: both arms ran under
the same harness, so a paired difference is still a paired difference.

**Does invalidate** two things:

1. Every projected leaderboard number in this repo, including the checklist's
   0.61 and 0.42. They were computed at `repeats=3`.
2. Any conclusion whose mechanism works *by* carrying across repeats.
   `ARC_CARRY` is the proven case: real at repeats=3, exactly zero at repeats=1.
   Whether other recorded decisions change rank at repeats=1 is **unknown and
   untested**, and is now the open question.

## The correction to make

`repeats=3` is fine for reducing variance when comparing two arms. It is wrong
as the setting that produces the headline number. Those are different jobs and
the bench does not currently distinguish them.

Note what did *not* cause this: the drop from 0.19 to 0.13 is two single runs and
this project's own rule says a single run is not a measurement. Nothing here
attributes that drop to anything. What is established is that the local number
was never predicting the leaderboard number in the first place.

Status: HARNESS ARTEFACT MEASURED AND CONFIRMED. PROJECTIONS INVALID.
RANK-STABILITY OF PAST DECISIONS AT repeats=1 UNTESTED.
