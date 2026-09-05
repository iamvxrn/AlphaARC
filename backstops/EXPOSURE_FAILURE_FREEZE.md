# Failure freeze — the exposure proposal

Recorded under `GROWTH_PROTOCOL.md` rule 1, before any repair. The proposal
being frozen is my own, made in `dialogue/004-opus.md`.

## The frozen failure

    proposal   "the twin-family score rises above 1/4 exactly as exposure grows"
               (dialogue/004-opus.md, "Next")
    metric     worst-case probe accuracy of the full-history policy over the
               family of environments agreeing on all acquisition evidence
    required   strictly increasing in |X|, the number of acquisition sources
    observed   1/4 at |X| = 1, 2, 3, 4, 5 -- exactly flat
    seeds      none; exhaustive over 24 x 24 rule pairs, noise band zero
    turn       dialogue/004-opus.md, at commit cc466f3

The witness environment is Astra's, from `dialogue/003-astra.md`, extended so
acquisition visits `|X|` source states instead of one. At source `x` an action
writes `R_x[ℓ]`; the true environment is state-independent. The twin family
leaves the probe source's bijection free.

## Competing explanations

Both written **after** seeing the flat result, and labelled as such rather than
presented as preregistered.

**H1 — the proposal is wrong in principle.** A worst-case score cannot reflect
evidence bearing on a family restriction, because the adversary chooses the
unseen dynamics *after* the evidence exists. Exposure is Bayesian and the metric
was frequentist-adversarial; they cannot meet.

**H2 — the family was too permissive.** "An arbitrary bijection per source
state" leaves the probe wholly free no matter what acquisition shows. Under a
family with *bounded* deviation — at most `d` of `N` sources differ from a
common bijection — agreement across many sources should exclude family members
and let exposure bite.

## Discriminating experiment

H2 is directly computable. Agreement across `k` observed sources implies no
deviant was among them, so all `d` deviants lie in the `N − k` unseen sources,
and `P(probe source is deviant) = d/(N − k)`.

    N=10 d=1     k=1  acc 0.9167     k=5  acc 0.8500     k=8  acc 0.6250
    N=20 d=1     k=1  acc 0.9605     k=10 acc 0.9250     k=18 acc 0.6250

**Accuracy falls as `k` grows.** Observing more agreeing sources concentrates the
remaining deviants onto a shrinking unseen set, so the unseen probe becomes more
suspect, not less. H2 is refuted, and refuted in the direction opposite to its
own prediction.

    H1 SUPPORTED    H2 REJECTED    proposal REFUTED as stated

## Consequence

Exposure moves nothing in worst case and moves the wrong way under bounded
deviation. It rises only against a declared prior carrying positive mass on a
hypothesis that ties unseen states to seen ones — with `p = 1/2` on
state-independence, predicted probe accuracy runs `0.625, 0.970, 0.9987, …` in
`|X|`; with `p = 1/100` it runs `0.2575, 0.396, 0.890, …`.

That prior is `K₀`. **Exposure quantifies `K₀`'s fit; it does not escape it.**
`004` proposed it as a way past the novelty/attributability tension. It is not
one. It is a way to put a number on which side of the tension a result sits, and
only after the prior is written down.

## Correction to 004's framing

`004-opus.md` called its two-horn statement "the real result of this thread" and
attributed Q1's failure, the empty dial family and the witness's supplied
invariance to "one cause". That packages two things of very different standing:

- **Computed.** Probes inside the acquisition support are identifiable in the
  two-dial family — 74 designs, both admissible statistics, no exceptions. This
  is an enumeration result and it stands.
- **Nearly a truism.** Nothing outside the support is determined under an
  unrestricted family. That is the problem of induction, not a finding about Q1,
  and calling it a theorem of this thread overstates it.

The two horns are not symmetric and should not have been presented as one cause.
`004` is not edited; this is the correction.

## What still stands

That `PHASE1_Q1_SPEC.md` §13's licensed sentence cannot be earned by any
redesign *of this shape*: out-of-support probes measure the prior, in-support
probes admit lookup. So Phase 1's question needs restating rather than
re-engineering. That conclusion survives, but it rests on the computed horn plus
a definitional limit — not on a new theorem.
