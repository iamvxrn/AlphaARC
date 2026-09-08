# The two submissions ran the same agent. The drop was leaderboard noise.

## The claim under test

    2026-08-31   0.19    agent at 59b1508
    2026-09-07   0.13    agent at 83d0911

Read as a regression. It is not one.

## Measured

Both agents, full 17-game train split, `repeats=1` (competition conditions),
16 paired seeds:

    differing (game, seed) pairs   0
    aggregate, 31 Aug agent        0.4799
    aggregate, 7 Sep agent         0.4799

Identical on every game and every seed. The only behavioural change between the
two commits was `ARC_CARRY`, and
`THE_BENCH_MEASURES_ITS_OWN_HARNESS.md` established that it is worth exactly
0.0000 when each game is played once, which is what the competition does.
Everything else in between was trace instrumentation or a flag defaulted off.

## What follows

**Nothing got worse.** There is no regression to find and no cause to look for.

**The leaderboard is noisy, and now there is a number.** Identical behaviour
produced 0.19 and 0.13 — a spread of 0.06 around a mean of 0.16, roughly 38%
relative, from n=2. Treat it as an order of magnitude, not an estimate.

The operational consequence is sharp: **a leaderboard change smaller than about
0.06 cannot be read from a single submission**, and submissions are rate-limited
per day. Local paired measurement at `repeats=1` is the only affordable
instrument. The leaderboard is for confirming plumbing and for changes large
enough to clear its own spread.

**The hidden set is harder than train.** Local `repeats=1` over 17 train games is
0.4799 against 0.13-0.19 observed. Roughly a factor of three, and this is after
the repeats artefact is removed, so it is a real property of the two sets rather
than a harness effect.

Status: NO REGRESSION. LEADERBOARD SPREAD ~0.06 AT n=2. TRAIN/HIDDEN GAP ~3x.
