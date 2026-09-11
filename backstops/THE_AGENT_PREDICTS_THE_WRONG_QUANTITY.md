# The agent predicts a 12-28% quantity while a 95% one sits in the same data

## The loop, as it actually stands

`Policy` keeps `succ[(control, state)] = levels_after` and consults it when
choosing. So the agent **predicts before acting**. It then records what happened
with a plain overwrite and **never compares the two** -- `predicted` appears only
at choice time, never at credit time. There is no error signal anywhere.

Instrumented that comparison (trace only, behaviour-neutral, tests pass).

## Measured, 4 seeds x 3 repeats

| game | checks | exact | by sign | levels |
|---|---|---|---|---|
| vc33 | 169 | **12.4%** | 53.3% | 2 |
| sp80 | 1806 | 24.6% | 65.6% | 0 |
| m0r0 | 520 | 27.5% | 53.3% | 0 |
| ls20 | **0** | — | — | 3 |
| g50t | **0** | — | — | 4 |
| re86 | **0** | — | — | 0 |

Two things at once.

**The agent's own predictions fail four times in five.** By sign they are near a
coin toss. Yesterday's "94-99.9% accurate" was *my* offline binary model, not
the agent's; the agent's own model is far worse and it never finds out.

**Three games produce no checks at all.** `succ` lives in `Policy`, the click
path. On keyboard games the forward model is never consulted, so the half of the
agent facing 17 of 25 games has no world model whatsoever.

## Bimodality: hypothesised, and refuted

The owner observed that human prediction is bimodal -- certain about movement,
absent about an unknown control, never 25%. So: is certainty already latent,
needing only to be measured?

No. Per control, with at least three checks each:

    exact   vc33 42% at the poles, sp80 8%, m0r0 0%
    by sign vc33  8%,               sp80 8%, m0r0 0%

Most controls sit in the middle at both granularities. There is no confident
subset to separate out.

## What replaces it

The comparison was unfair to the agent, and fixing it gives the real finding.
A human predicts *displacement of an object*; the agent predicts a *scalar
compression delta*. Those are not the same difficulty.

    predicted now:  size of the level change, from (control, level vector)
                    holds 12-28%

    available:      did the avatar move, from (control, position)
                    holds 94-99.9%

**A near-deterministic signal exists in the same traces and the agent is not
predicting it.** The owner's model -- certain or absent, never mediocre -- is
right as a target and unreachable on the quantity currently chosen.

It also explains the keyboard hole: displacement is exactly what the keyboard
games are about, and it is the path with no model.

Status: ERROR SIGNAL ADDED AND MEASURED. BIMODALITY REFUTED. THE PREDICTION
TARGET IS WRONG, AND A BETTER ONE IS ALREADY IN THE DATA.
