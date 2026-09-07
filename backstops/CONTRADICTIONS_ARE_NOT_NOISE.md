# Contradictions are common, mostly lawful, and their timing is the signal

## The question

The owner's framing: a person does not conclude "no game has doors" from one
door-less game. They carry "doors exist here" as a prior and check. The agent
carries nothing, because a key->offset map has no slot for "except sometimes".

Before proposing any mechanism: is there anything to notice? Counted on the 16
g50t traces already collected, no new runs, no agent change.

A **contradiction** is one `(control, position)` pair that has produced both
outcomes -- a press that changed the board and a press that did not.

## Contradictions are common

Per run of ~742 presses: 24-43 contradictory pairs, and **about 260 presses, 35%
of everything the agent does, land on such a pair**. Winners 36%, losers 35% --
the raw count does not distinguish them.

## And they are not noise

If outcomes at a pair were independent draws at the observed no-op rate (~0.19),
48-68 pairs would be contradictory by chance. Observed: 24-43. The ratio is
**0.58** -- outcomes are far more consistent than chance, so `(control, position)`
already predicts most of what happens.

That matters in the direction that helps: the exceptions are structured, not
scattered. It is also partly inflated by clustering -- the agent presses one key
at one place up to seventeen times running, and those agree trivially.

## The signal is the timing, not the count

A contradiction whose outcomes are cleanly separated in time -- all one way, then
all the other -- is what a state change looks like. Interleaved outcomes are what
noise looks like.

| | share of contradictions cleanly separated in time |
|---|---|
| the 5 seeds that clear level 1 | 18, 26, 33, 37, 38 % |
| the 11 that do not | 4, 4, 4, 8, 8, 8, 8, 12, 12, 17, 21 % |

Overall 19%, which is around what shuffled outcomes would give -- so "contradictions
are informative in general" is **not** supported. But the distributions barely
overlap: the losing maximum is 21 and the winning minimum is 18.

The reading: temporal separation detects *that the world changed*, and it fires
three times more often on the seeds where the door actually opened. On the eleven
where nothing ever changed, contradictions are noise, and correctly so.

## What this licenses

The idea is not dead. There is a large, structured, discarded signal, and a cheap
statistic that distinguishes a real state change from noise.

It does not license a mechanism. The 20-30% chance baseline quoted above is
hand-waved rather than computed by shuffling, and n is 5 against 11. Both need
fixing before anything is built on this.

Status: SIGNAL PRESENT AND LOCATED IN THE TIMING. BASELINE NOT YET COMPUTED
PROPERLY. NO MECHANISM LICENSED.
