# Temporal separation is refuted by the baseline the previous record lacked

## What was claimed, and what killed it

`CONTRADICTIONS_ARE_NOT_NOISE.md` reported that contradictions whose outcomes are
cleanly separated in time -- all one way, then all the other -- fire three times
more often on the seeds that clear g50t: 18-38% against 4-21%. It recorded, as a
stated weakness, that "the 20-30% chance baseline is hand-waved rather than
computed by shuffling".

Computing it properly kills the claim.

For a pair with `k` observations of which `m` are one outcome, exactly two of the
`C(k, m)` arrangements have a single switch. So the chance share is `2/C(k,m)`
per pair, exactly, with no sampling.

| | separated | expected by chance | ratio |
|---|---|---|---|
| 5 seeds that clear level 1 | 61 | 74.1 | **0.82** |
| 11 that do not | 26 | 35.2 | **0.74** |

Both below chance, and barely apart. Per seed the ratio scatters from 0.18 to
2.05 with no relation to the outcome.

The apparent 3x split was an artefact of **pair count and pair length**. Winning
seeds carry more contradictions (39 against 25) and longer ones, and both raise
the raw share of separated pairs without any signal being present.

## The same on every other game with a usable sample

| game | contradictions | ratio |
|---|---|---|
| m0r0 | 135 | 0.84 |
| sc25 | 14 | 0.88 |
| sp80 | 8 | 1.00 |
| ls20 | 8 | 1.65 |
| ar25 | 0 | — |

Nothing above chance where the sample supports a judgement. `ls20`'s 1.65 rests
on eight pairs.

## What survives

- Contradictions are common: 24-43 pairs per g50t run, about 35% of all presses.
- They are far rarer than chance -- ratio 0.58 -- so `(control, position)` is
  largely lawful and the exceptions are structured rather than scattered.
- **The idea is untouched.** A world where a control works only in some states
  needs somewhere to record that, and the agent has nowhere. That is still true.

What is dead is my proposed *detector*. Timing does not separate a real state
change from noise, on any game measured.

## Note on how this went

The previous record named the missing baseline as its own weakness, in writing,
before the baseline existed. Computing it refuted the record. That is the third
time in this investigation that a predeclared caveat has caught my own error --
the others being fix B's control and the "losers never change column" claim.

Status: DETECTOR REFUTED. THE UNDERLYING PROBLEM IS UNCHANGED AND STILL OPEN.
