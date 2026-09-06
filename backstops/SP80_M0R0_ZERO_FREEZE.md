# Frozen failure: sp80 and m0r0 complete no level, and budget is not why

## The observation, frozen

    commit        817cf0c (fast-loop)
    artefacts     python/bench/runs/loop-base, python/bench/runs/d1-budget2000
    settings      repeats=3, 8 seeds -> 24 plays per game per condition

| condition | max_steps | actions spent | levels, sp80 | levels, m0r0 |
|---|---|---|---|---|
| baseline | 250 | 251 every play | 0 in 24 | 0 in 24 |
| D1 | 2000 | 2001 every play | 0 in 24 | 0 in 24 |

Score exactly 0.0000, standard deviation exactly 0, in both conditions. The
budget was applied (`max_steps: 2000` is recorded in each result file) and it was
fully spent (the cap is reached in every play). 96 plays in total.

Reproduce:

    python3 python/bench/loop.py --seeds 8 --tag loop-base
    seq 1 8 | xargs -P8 -I@ $KIT/.venv/bin/python python/bench/bench.py \
        --kit $KIT --games sp80,m0r0 --max-steps 2000 --repeats 3 --seed @ \
        --out python/bench/runs/d1-budget2000/seed@.json

A zero with zero variance is stronger evidence than a small noisy number: there
is no seed to blame, and the noise band cannot be invoked in either direction.

## Two failures, not one

Recording them together would assume a shared cause, which the existing evidence
contradicts. They look opposite where it matters:

| | sp80 | m0r0 |
|---|---|---|
| avatar detection | works (80-cell colour-9, 4 directions) | broken: two mirrored markers, the union is not a translation |
| census `simple_actions` | ACTION2/3/4 stable, `translate` x3 | ACTION1: `translate` -> `edit` -> `noop` |
| repeated (control, position) pairs | 1 in a whole run | 50, of which 40 contradictory |
| enclosed-region goal candidates | none | none |

They are frozen as `sp80-zero` and `m0r0-zero`.

## What D1 rules out

**H5 -- insufficient budget: REJECTED.** Eight times the actions, fully spent,
across 48 plays, produced not one level in either game.

**H2 -- reach/coverage: DAMAGED, not dead.** H2 said the agent so rarely returns
to a place that there is nothing to condition on. Revisits grow at least linearly
with steps in a bounded board, so eight times the budget should have supplied the
repeats that H2 says are missing. It supplied none of the benefit. What survives
of H2 is only its stronger form -- that the *policy* systematically fails to
revisit, so more steps buy proportionally more of the same non-repeating walk --
and that is now a claim about the policy, not about the budget.

## What survives, and why the observation favours it

Both remaining leading hypotheses predict exactly the thing D1 measured -- that
**more experience does not help**:

- **H1 -- a hidden mode makes the action model wrong, not incomplete.** Both zero
  games carry `varies=True` and a toggle (`sticky`, `cycle`); the two games that
  do score are `varies=False` with no toggle. Under non-stationarity, accumulating
  experience of a map that changes underneath makes the model worse, so budget is
  irrelevant by construction. This is the boundary E0 hit on `core-zero`, from the
  other direction.
- **H3 -- object identity (m0r0 only).** Two bodies collapsed into one centroid;
  the "avatar" is a different object step to step. More steps of a mislabelled
  variable are more mislabelled steps.
- **H4 -- no destination.** Zero enclosed regions on exactly these two games, and
  enclosure was the best destination candidate before it was closed by
  measurement. Undirected movement does not become directed with more of it.
- **H6 -- the null.** Not excluded.

Predicting the observed result is weak evidence: all four predict it, so D1
separates none of them from each other. It only removed the one that predicted
the opposite.

## Next discriminator, not yet run

`D2` -- a scripted or human route. If a short hand-written action sequence clears
level 1 of sp80, then the actions are usable and the failure is aiming (H4), not
the action model (H1). If no such sequence can be written by a person reading the
board, the problem is upstream of the agent. This is the cheapest experiment that
separates H1 from H4, and it changes the evidence rather than the agent.

`D4` -- the predeclared correlation: if H1 is the cause, `varies=True` should
depress the score across all 17 train games, and `click_density` should NOT split
them the same way. Predeclared here, before being looked at.

Status: FAILURE FROZEN. ONE EXPLANATION REJECTED, ONE DAMAGED, FOUR ALIVE.
No mechanism is licensed by this record.
