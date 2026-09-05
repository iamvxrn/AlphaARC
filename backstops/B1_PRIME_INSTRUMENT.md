# B1′ — instrument specification

Written to close the gap frozen in `B1_INSTRUMENT_FAILURE_FREEZE.md`, and
committed **before** it is run (GROWTH_PROTOCOL rule 4).

This file specifies a *measurement instrument*. It is tooling, exempt under the
protocol's standing exceptions. It introduces no agent, no `U`, and no successor
environment.

## What it does not do

It does **not** amend `PHASE1_Q1_SPEC.md`. That spec stays frozen at `a4877fb`
and Q1 stays **REJECTED** by its own §11 criterion, on the `0.50` already
recorded in `B1_FAILURE_FREEZE.md`. Nothing here can un-reject it, and nothing
here is permitted to try. The instrument exists so that **Q1′** can be designed
against a criterion that is actually computable — not to relitigate Q1.

`backstops/q1_backstops.py` is likewise left byte-identical to `a24e111`, so the
frozen numbers stay reproducible. B1′ lands in a separate file.

## The gap being closed

Frozen B1 says:

> exhaustively compute the best probe accuracy achievable by **every lookup
> table fitted to acquisition-reachable data**

The instrument freeze established that this names no finite class, because the
spec fixes neither the fitting data nor the admissible fitting rules — and that
under the *unrestricted* reading a fitting procedure reads the whole transition
table, identifies `R`, and scores `1.00`. That reading is useless: it does not
describe a shortcut, it describes exactly the acquisition Q1 wants to measure.

So the class has to be pinned by a property that distinguishes *a lookup table
keyed on `K`* from *an arbitrary program that happens to emit a table*. There is
one such property, and it is the defining one.

## The class, pinned by locality

A **`K`-indexed lookup learner** is a pair `(K, F)`:

- `K ⊆ {d1, d2, t1, t2}` — the features the table is indexed by.
- The learner's entire memory is a table indexed by `π_K(o)`.
- **Fitting data**, unchanged from the frozen implementation: for every
  acquisition-reachable non-goal state `s`, every label that strictly reduces
  distance to the target is recorded at key `π_K(s)`. This is readable by an
  agent from its own transitions, since the target is in the observation.
- **Locality (the load-bearing clause).** The table's entry at key `k` is a
  function of the fitting data recorded *at key `k`* — and of nothing else.
- **Fitting rule.** Subject to locality, `F` is an **arbitrary** function from
  the local data to a label. Not majority vote. Not any named algorithm. Every
  function.

Locality is what the word *table* was already doing in the frozen text. A
procedure that consults data at other keys to decide the entry at `k` is not a
table indexed by `K`; it is a program with unrestricted access, and that is the
reading the instrument freeze showed to be vacuous.

## Why this makes the maximum computable

Fix `K`. Let `L(R)` be the local fitting data at the probe key `π_K(S*)` under
rule instance `R`. `F` is an arbitrary function of `L`, so `F` can distinguish
`R` from `R′` exactly when `L(R) ≠ L(R′)`. Partition the 24 rule instances by
the value of `L`. On each class the learner must emit one label. Therefore

    max over all fitting rules F  =  (1/24) · Σ_classes  max_ℓ #{R ∈ class : ℓ⁺(R) = ℓ}

where `ℓ⁺(R)` is the label that increments the probed dial. Finite, exact, no
sampling, no chosen algorithm. Two boundary cases fall out rather than being
stipulated:

- **Probe key unreachable.** The local data is empty for every `R`, so all 24
  instances lie in one class and the entry is a constant. Since `ℓ⁺` is uniform
  over the four labels across the 24 instances, accuracy is exactly `0.25`.
  This is the same default the frozen implementation used, now derived instead
  of assumed.
- **`K` = the full observation.** No information is lost — but `π_K(S*) = S*`
  is unreachable by §5, so the table has no entry there and the case collapses
  into the one above. The `1.00` of the unrestricted reading is excluded by
  locality, not by fiat.

The trade-off is now explicit and is the whole content of the instrument: a key
coarse enough to have a fitted entry at the probe is a key that has thrown away
information; a key fine enough to keep the information has no entry.

## Relation to the frozen numbers

Majority vote is one member of the class, so B1′ ≥ B1 for every subset. The
frozen `0.5000` is therefore a lower bound on what B1′ reports.

## Predeclared prediction, before the run

1. **B1′ ≥ 0.5000.** By dominance. This is not a prediction, it is arithmetic,
   and is stated only so that a value below it is read as a bug in the new code
   rather than a finding.
2. **The open question: is B1′ > 0.5000 on some subset?** Genuinely unknown at
   the time of writing. Not peeked at.
   - If **B1′ = 0.5000**: majority vote happened to attain the maximum of the
     class. The frozen `0.50` was the right number for a reason the frozen code
     did not establish, and now it is established.
   - If **B1′ > 0.5000**: Q1 is worse than its own failure freeze recorded, the
     rejection is strengthened, and `B1_FAILURE_FREEZE.md`'s closing claim —
     that the PARTIAL/FULL gap carries more weight than NONE/PARTIAL — gets
     quantitatively worse.
3. **`{d1,t1}` scores `0.5000`.** Its local data at key `(0,0)` is two votes for
   the increment label and two for the decrement (`B1_FAILURE_FREEZE.md`,
   minimal case), symmetric under swapping them, so no fitting rule can break
   the tie. Stated in advance as a check that the new code reproduces the known
   case for the known reason.
4. **Every subset containing `t2` scores `0.2500`**, by unreachability.

A result contradicting 1, 3, or 4 means the instrument is wrong, and is to be
frozen as an instrument failure before anything is concluded from 2.

## What B1′ licenses for Q1′

One sentence, and it is a design constraint rather than a result:

> A candidate successor environment passes B1′ only if, for **every** feature
> subset `K`, the local fitting data at `π_K(S*)` leaves the probed label
> unidentified — either because the key is unreachable, or because the data at
> that key is symmetric across all four labels.

This is the computable form of the requirement `B1_FAILURE_FREEZE.md` left as
prose: *make the probe situation unidentifiable from acquisition-reachable
features, not merely unvisited.* It does not say a satisfying design exists.
Whether one does is the next question, and it is not settled here.
