# Research map

`python/bench/README.md` is 1400 lines of research log and it holds everything
below. It is a log, not an index, so the same question gets asked twice: D1
(2026-09-06) re-ran Rejected #8's experiment at a larger multiple because neither
the owner nor the agent proposing it could find the earlier answer.

This file is the index. It adds no knowledge and settles nothing. Every row
points at the section that carries the evidence.

## How to read the reliability column

The scoring split carries **sd ~0.83**; eight seeds of unmodified code range
2.32 to 4.58. So:

| | |
|---|---|
| **16+ seeds** | trust the direction |
| **8 seeds** | trust it if the effect is large |
| **1 seed** | not a measurement -- the log says so itself at its own line 150 |

The log's own methodology changed partway through. Rejected #3, #4 and #5 are
from the single-seed era; from Rejected #6 onward the work is paired at 16 seeds.
That improvement was made before any of this conversation, by the owner.

## Closed by measurement

| # | claim tested | seeds | outcome |
|---|---|---|---|
| R3 | mask the move-budget HUD so compression stops measuring the clock | 1 | rejected -- **re-test before trusting** |
| R4 | two silent presses, not one, before a control is written off | 1 | rejected -- **re-test before trusting** |
| R5 | re-order click candidates away from smallest-first | 1 | rejected, two variants -- **re-test** |
| R6 | back the state key off to a state-free fact | 16 | rejected; the premise it rested on was itself wrong |
| R7 | put learned value and rank prior in the same units | 16 | rejected; the motivating measurement stands |
| R8 | more search time inside the episode | 16 | rejected -- doubling the budget bought nothing |
| R9 | lift the permanent `inert` write-off | 16 | rejected -- no levels gained, and cost lp85 and vc33 0.28 each |
| R10 | prefer a control not yet tried from this board | 16 | rejected -- the trace answered it before it was built |
| D1 | eight times the action budget on sp80/m0r0 | 8 | rejected -- 48 plays, zero levels, zero variance. Extends R8 |

## Established, and load-bearing

| finding | seeds | what it means |
|---|---|---|
| the avatar detector was never the blocker | 16 | the movement gate opens; the goal is the missing half |
| a destination must be **held**, not re-derived each step | 71 | two defects sat between the agent and any goal |
| name a control by its **rank** among peers, not its position | 16 | position breaks at the level seam it was built for |
| an absence of reward is not a known absence of effect | 33 | "watched it do nothing" and "never seen" scored identically |
| carrying the mechanic across a game's runs | 16 | **+1.21** -- the largest recorded gain |
| after GAME_OVER the engine hands back a clean level | 33 | recovery works; the waste is elsewhere |
| the drive is silent for 80% of decisions | 16 | most choices are made by the prior, not the mechanism |

## The destination rule: four candidates closed, none surviving

This is the open frontier, and it is where sp80 and m0r0 die.

| candidate | how it died |
|---|---|
| imagined compression | closed by measurement |
| the five-cell plus | closed by measurement |
| enclosed region | **zero regions on sp80 and m0r0**, and it misses g50t's goal, whose room is open on one side |
| conditioning on position | ls20 1% contradictory, g50t 23%, m0r0 **80%** |
| conditioning on body configuration | worse: m0r0 87%, g50t 84% -- a finer key loses repeats without buying truth |

What is established is **where the avatar must end up in two games**. That is the
evidence any fifth candidate has to explain.

## Open, with nothing spent on it yet

- `sp80-zero` and `m0r0-zero`, frozen in `backstops/SP80_M0R0_ZERO_FREEZE.md`.
  Four explanations alive: hidden mode (`varies=True` on exactly the two zero
  games), object identity, no destination, and the null.
- **D2**, unrun: does a hand-written route clear sp80 level 1? Separates aiming
  from the action model without touching the agent.
- **D4**, predeclared and unlooked-at: `varies=True` should depress score across
  all 17 train games while `click_density` should not split them the same way.
- Keyboard actuation: 17 of 25 public games need it, every win is click-only, and
  the sealed holdout is keyboard-only.
- Rejected #3, #4, #5 -- single-seed conclusions that were never re-measured.

## The rule this file exists to serve

Execution can be delegated. Epistemology cannot. Every row above can be checked
by asking three questions, and any answer that needs eloquence instead of a
number is not an answer:

    What experiment could have falsified this?
    How many seeds, and what was the spread?
    Is the difference larger than the spread?
