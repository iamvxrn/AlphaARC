# Phase 1, Q1 — experimental specification

Frozen before `U` is chosen. Environment and protocol only: no agent
implementation, no substrate, no learning rule.

> **Q1.** Can an agent acquire a previously unknown regularity from its own
> interaction and use it correctly in a novel state, with that regularity absent
> from `K₀`?

Operationally, and no larger than this:

    experience → persistent internal change → transfer

Composition is deliberately **not** tested here; it is a possible Q2, and only
after Q1 passes.

---

## 1. Why the obvious design fails its own audit

The illustrative sketch —

    X1 + A → Q,   X2 + A → Q,   X3 + A → Q,   then novel X4

— is rejected. If action `A` produces `Q` regardless of state, the acquired
content is a single fact about an *action*, not a regularity over *states*, and
the constant policy "always A" scores perfectly with zero acquisition. The
correct action must be a **function of the state**.

## 2. Environment: TWO DIALS

State, fully observable:

    d1, d2 ∈ {0,1,2,3}      current dial values   — agent-controlled
    t1, t2 ∈ {0,1,2,3}      target dial values    — ENVIRONMENT-controlled

Observation is exactly the tuple `(d1, d2, t1, t2)`. Nothing else: no cumulative
score, no step counter, no episode index, no history summary. Everything the
agent carries from its past must travel through `h_t`, not through the
observation — this is what makes §6 enforceable.

Actions: four opaque labels `{α, β, γ, δ}`, carrying no meaning.

**Hidden rule instance** `R`: a bijection from the four labels onto the four
effects

    d1+1,  d1−1,  d2+1,  d2−1        (all mod 4)

Objective (A1): `+1` when `(d1,d2) == (t1,t2)`; a new target is then issued.

State space 256, actions 4, dynamics deterministic — 1024 transitions, and 24
rule instances. Enumerable exhaustively.

## 3. `R` is absent from `K₀` by construction, not by promise

`R` is **drawn uniformly from the 24 bijections after `K₀` is frozen**,
independently per run. No `K₀` — however large, however trained — can carry
information about which label increments dial 2. Contract rules L1 and L2 are
discharged *mechanically*: `I(K₀ ; R) = 0` by sampling, not by argument. The
undefined notion of "mechanic family" never has to be adjudicated here.

The observation carries no action identity, so `R` cannot leak through the
rendering either.

## 4. Acquisition phase

Every acquisition episode is generated with

    t1 ≠ d1       dial 1 must be moved — this is what the score rewards
    t2 = 0        ALWAYS, and d2 starts at 0

Dial-2 effects are therefore **reward-irrelevant throughout acquisition**. They
remain fully *observable*: any press of a dial-2 label visibly changes `d2`, and
having broken the match, the agent must undo it. Evidence about the dial-2
effects is available in the agent's own transitions and never useful.

Budget: **exactly `N` steps**, identical in both arms, no early termination.
When a target is met a new one is issued and the step count continues, so the
two arms are matched on total steps regardless of how well either did.

## 5. Novelty by construction

The held-out coordinate is **`t2`**, and it is held out *structurally*:

> `t2` is set by the environment and there is no action that changes it.
> Throughout acquisition `t2 ≡ 0`. Therefore **no observation with `t2 ≠ 0` is
> reachable during acquisition, under any policy whatsoever.**

The probe uses `t2 = 1`. Its observation is not rare in acquisition — it is
*impossible*. Novelty is a property of the protocol, not a filter applied to
results, so no policy-dependent subsetting of runs occurs anywhere in the
primary analysis. This replaces an earlier draft that made the probe's dial
value merely improbable and excluded contaminated runs post hoc; that was
rejected because subset membership would itself depend on how the agent
explored.

**Stated limit, not papered over.** What is excluded by construction is the
probe *configuration*. The probe's underlying *transition* — `(d2=0, ℓ) → d2=1`
for the increment label `ℓ` — may well have been experienced during acquisition,
and cannot be made unreachable without removing the evidence the agent needs.
Q1 therefore does not distinguish "stored `ℓ → d2+`" from any richer
representation, and is not intended to: functional acquisition is the claim,
representation format is not. The count of runs whose history contained that
transition is reported as a **descriptive statistic only** — never as a filter.

## 6. The probe, and observational equivalence

At step `N+1`, in **both arms identically**, the environment forces

    S* :  d1 = 0,  t1 = 0,  d2 = 0,  t2 = 1

Dial 1 is already satisfied; dial 2 is off by exactly one in the `+` direction,
so exactly one action is optimal — the dial-2 **increment** — and `+1` / `−1`
are not tied.

Everything observable at the probe is identical across arms and across runs:

| at the probe | MATCHED | MISMATCHED |
|---|---|---|
| observation | `S*` | `S*` |
| available actions | `{α,β,γ,δ}` | `{α,β,γ,δ}` |
| objective | same | same |
| steps elapsed / budget | `N` | `N` |
| forced state-set event | yes | yes |
| **acquisition history, and `h_t`** | **under `R`** | **under `R′`** |

The forced state-set happens in both arms, so the discontinuity itself is not a
cue. The only informational difference entering the probe is `h_t`.

**Measured quantity: the first action taken at the probe** — before any
probe-phase transition exists. No eval-time evidence about `R` can have reached
the agent.

## 7. Matched control

| arm | acquisition dynamics | probe dynamics |
|---|---|---|
| **MATCHED** | rule instance `R` | `R` |
| **MISMATCHED** | rule instance `R′ ≠ R` | `R` |

Same seeds, same budget, same observation distribution, same reward density,
paired. In MISMATCHED the map is silently re-drawn at the probe; since only the
first probe action is measured and the observation is forced to `S*`, the switch
is undetectable in principle.

Floor `1/4`. Ceiling: an oracle handed `R` (measurement tooling, exempt under the
protocol's standing exceptions).

## 8. Shortcut audit

**The randomization argument.** `R` is drawn uniformly and independently of
everything outside the agent's own transitions. Therefore *any policy that does
not use acquisition-phase transitions has expected probe accuracy exactly 1/4* —
however clever it is.

Under the §6 construction this gets sharper: the probe observation is `S*` in
**every** run and both arms, so an observation-only policy emits the *same label*
every time, and that label is the increment in exactly `1/4` of rule draws.

| candidate shortcut | probe accuracy | why |
|---|---|---|
| constant action | 1/4 | label→effect is a fresh uniform draw |
| any function of the observation alone | 1/4 | observation is constant at `S*`; ⊥ `R` |
| lookup on exact state | n/a → 1/4 | `S*` is unreachable in acquisition (§5) |
| lookup on any subset containing `t2` | n/a → 1/4 | `t2 = 1` never observed |
| lookup on subsets excluding `t2` | **≤1/4** | those states were dial-1 work or undo situations; predicts a dial-1 label or the decrement |
| detect the `R → R′` switch | 0 bits | forced `S*`, zero actions taken since |
| notice the forced state-set | 0 bits | occurs in both arms |
| exploit at probe time | n/a | only the first action counts |
| infer `R` from the rendering | 0 bits | no action identity in the observation |

### What the randomization argument does NOT close

It covers policies independent of acquisition experience. It does not cover
policies that use that experience *without acquiring the target regularity*, and
one such policy beats chance:

> A learner that has associated the two dial-2 labels with *dial 2* — without
> ever learning which of them increments — has narrowed the probe answer to two
> labels. **Expected accuracy 0.5, direction never acquired.**

This is kept, not engineered away: it turns the probe into a small assay of *how
much* was acquired. Note it is genuinely distinct from mere suppression — a
learner that only marked those labels *bad* (they cost steps in acquisition)
avoids them at the probe and lands **below** chance.

## 9. The ladder, predeclared

    < 0.25   SUPPRESSED   dial-2 labels avoided, or reward-following
                          presses a dial-1 label — interpretable, not a bug
    ≈ 0.25   NONE         nothing about R was acquired
    ≈ 0.50   PARTIAL      "these two labels act on dial 2" acquired,
                          direction NOT acquired
    → 1.00   FULL         "this label increments dial 2", applied in a
                          configuration that was unreachable during acquisition

**The two measurements answer different questions and are never merged:**

- *paired* MATCHED − MISMATCHED → **was the improvement caused by the relevant
  experience?**
- *absolute level* on the ladder → **how much of the mapping was acquired?**

A result on the `0.50` plateau is reported as **PARTIAL**, never as PASS. Fixed
here, before any number exists.

## 10. Predeclared prediction (protocol rule 4)

    C (must improve):     MATCHED first-action accuracy > 1/4
    D (must NOT improve): MISMATCHED first-action accuracy stays at 1/4

If **both** arms rise above chance the effect is non-specific and Q1 is **not**
demonstrated. Likely causes: `R` leaking into the observation, an action bias
interacting with `S*`, or probe-time evidence reaching the agent. Investigate the
instrument; do not report the number.

## 11. Backstops, to be run before any agent exists

- **B1.** Over the enumerated transition table, exhaustively compute the best
  probe accuracy achievable by every lookup table fitted to acquisition-reachable
  data. Required: `0.25`. Anything above rejects the design.
- **B2.** Uniform-random agent. Required: `0.25 ± noise`. Deviation means the
  probe or the scoring is broken, not the agent.
- **B3.** Oracle handed `R`. Required: `1.00`. Anything less means the probe is
  not solvable and no failure may be attributed to any agent.

## 12. Measurement (protocol rule 7)

Paired on identical seeds, MATCHED vs MISMATCHED. Per-run outcome is Bernoulli
against a `0.25` floor, so this needs tens of runs, not four. Report the mean
paired difference with its standard error and the seed count; add seeds until the
difference clears twice its standard error. **Every number carries its `n`.**

## 13. What PASS and FAIL license

**PASS** (FULL, and MATCHED > MISMATCHED above noise) licenses exactly one
sentence:

> The agent acquired, from its own interaction, information about a previously
> unknown action-effect mapping, and used it in a configuration that could not
> occur during acquisition.

It does **not** license: *learned causality*, *learned abstraction*,
*generalization*, *a world model*, *core knowledge found*, or any claim about
which part of the agent did it. The claim is small on purpose.

**FAIL does not license "K₀/U insufficient".** Three-valued, and these checks are
cheap and run first:

1. **Was the evidence ever present?** Count presses of the increment label during
   acquisition and whether their effect was observed. Zero ⇒ this is an
   **exploration** failure, attributed elsewhere in the discriminator table.
2. **Did anything reach the probe?** Confirm `h_t` at the probe depends on
   acquisition at all. If not, the failure is state-persistence.
3. **Is the probe solvable?** B3 must be `1.00`.

Only with 1–3 clean does the result become **FAILURE ATTRIBUTED**. Otherwise
**FAILURE UNATTRIBUTABLE** — a valid terminal state, reported as one.

## 14. Deliberately deferred

- Composition of independently acquired rules → possible **Q2**, only after PASS.
- Raw / high-dimensional observation encoding (a later transfer axis; the
  symbolic tuple is chosen so that B1 can be exhaustive).
- Representation-format requirements of any kind.
- `U`, the substrate, ARC-AGI-3.
