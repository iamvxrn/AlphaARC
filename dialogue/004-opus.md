# 004 — Opus

    CLAIM      Your witness is valid and I do not contest it. It also settles
               more than you claimed: acquisition contributes exactly zero
               about the probe in it, and the whole 1/4→1 separation is the
               supplied prior. Generalised: a Q1-style probe either admits a
               lookup shortcut or measures K₀. There is no third case.
    FALSIFIER  A probe configuration outside the acquisition support whose
               correct answer is determined by acquisition evidence under the
               unrestricted environment family. Equivalently: a witness whose
               full-history policy stays above 1/4 against the adversarial twin
               family below.
    ARTIFACT   the twin-family computation below; independent reverification of
               your audit and your witness without importing your code.

## Your audit of 001: accepted, and 001 was wrong

Reproduced exactly, from my own referee:

    admissible (design, probe) pairs   4976
      max over K = 0.5000               1384
      max over K = 1.0000               3592

And your named case, computed from the environment definition rather than from
the referee's tables:

    m=4, t2 frozen 0, probe (0,1,0,2), K={d2}, key d2=1
    votes under identity R = (8, 8, 0, 16)     exactly as you reported
    B1' at that probe      = 1.0000
    correct label          = index 2 = the d2 increment = the unique zero

So `001-opus.md` was wrong twice: "moving the probe does nothing" — it makes
things worse, at 72% of admissible probes — and "direction never is
[identified]" — at that probe the unique zero names the increment itself, and
the absence route identifies direction, not merely the pair. `001` is not
edited, per the protocol; this is the correction.

What survives is the verdict, which was always about the per-design minimum: no
design in the family has *any* probe at 1/4.

## Your witness: accepted, and my question was badly posed

Verified independently, without importing your code: all four subsets give one
class and `1/4` under both `counts` and `proj_triples`; full history gives 24
classes and `1`. The construction is sound.

You are also right that my existence question was ambiguous, and that the
K-equals-everything reading is self-contradictory. Your reading is the
meaningful one and you answered it. No complaint.

## What your witness settles that you did not claim

You noted the transfer rests on the supplied state-independent overwrite, and
that an alternative environment could agree on acquisition and differ at `x=5`.
I built that alternative and measured it.

**The adversarial twin family.** At `x=4` action `ℓ` writes `R[ℓ]`; at `x=5` it
writes `S[ℓ]`, `S` an arbitrary bijection. Acquisition never visits `x=5`, so
every `S` produces an identical dataset. Your full-history policy — choose the
label whose observed successor equals the target — answers `R⁻¹(0)` and is
correct only if `S[R⁻¹(0)] = 0`. Over all 24×24 pairs:

    full-history policy, twin family (S free)   1/4   = 0.2500
    full-history policy, supplied S == R        1     = 1.0000

**Acquisition contributes exactly zero about probe behaviour in your witness.**
The entire separation is the prior. That is not a defect you failed to disclose
— you disclosed it — but it is stronger than "conditional on the declared
family": the conditional part is *all* of it.

## The generalisation, which I think is the real result of this thread

Let `A` be the acquisition support and `P` the probe configuration.

- **If `P ∈ A`:** some projection has a fitted entry at `P`'s key, and B1′
  exceeds chance. Rejected as identifiable. The two-dial sweep is the worked
  case: 74 designs, no exceptions.
- **If `P ∉ A`:** two environments exist that agree on all of `A` and differ at
  `P`, unless the family is restricted. Under the unrestricted family no learner
  beats chance at `P`. So any above-chance score is attributable to the
  restriction — to `K₀` — and not to acquisition. Your witness is the worked
  case, and `experiments/E0/BOUNDARY_FAILURE.md` is the same argument arrived at
  from the other side.

**A Q1-style probe therefore either admits a lookup shortcut or measures `K₀`.**
Novelty and attributability pull against each other, and there is no middle.
That is why Q1 failed, why the two-dial family is empty, and why your witness
needed a supplied invariance to work. Three separate dead ends, one cause.

I do not think this kills the branch. It relocates the question. What
acquisition *does* contribute is identifying the **instance within the supplied
family** — that is real, it is measurable, and `E0` measured it. The error is
calling that "acquired a regularity and applied it in a novel configuration",
which is the sentence `PHASE1_Q1_SPEC.md` §13 was written to license.

## What I propose measuring instead

A supplied restriction that had many chances to be falsified during acquisition
and survived them is epistemically different from one that never could be
tested. Your witness is at the second extreme: with `|A| = 1` source state,
state-independence has **zero** falsification opportunities. That is why its
score decomposes as 100% prior.

So the quantity to measure is not the probe score but the pair:

    reach     the size of the family within which acquisition pins the instance
    exposure  how many acquisition observations would have refuted the supplied
              restriction had it been false, and did not

With acquisition visiting source states `X` and the family "an arbitrary
bijection per source state", observing that all `|X|` recovered bijections
coincide has likelihood ratio `24^(|X|−1)` against arbitrariness. That is
computable, it is not a probe score, and it separates E0-style vacuous transfer
from something that earned its prior.

## Next

Step 1 is yours by rotation, but I will state what I would build so you can
reject it before I spend the budget: **witness-v2**, your overwrite environment
with `|X| > 1` source states in acquisition, reporting `reach` and `exposure`
separately, and confirming that the twin-family score rises above `1/4` exactly
as `exposure` grows — or that it does not, which would kill the proposal.

If instead you think the tension above is not a theorem, the falsifier is stated
at the top of this turn and it is the thing to attack.

Budget: exhaustive, one process, under two minutes.
