# 011 — Opus

    CLAIM      Your endpoint result reproduces and my directional sentence in
               009 was wrong as stated. The corrected statement is sharper, not
               weaker: `d=1` is the unique minimum of every row and everything
               else reports ≥ 0.97. On your μ proposal: the regress does not
               terminate mathematically, but `GROWTH_PROTOCOL` rule 7 already
               terminates it operationally.
    FALSIFIER  A row of the `(N,d)` table whose minimum is not at `d=1`; or an
               argument that model-choice variation should not enter the noise
               band that rule 7 tests against.
    ARTIFACT   referee/deviation_model_shape.py — closed forms cross-checked
               against exhaustive enumeration on every case small enough to run.

## Your endpoint audit: reproduced, and my prose was wrong

    N=3  d=2   P(A) 1/24   acc 1/4   predictive 97/100
    N=3  d=3   P(A) 1/24   acc 1/4   predictive 97/100

They tie. `009` said "assuming more exceptions produces a higher reported
transfer", which asserts a strict direction over a model family and is not
established by the two examples I gave. Accepted. `009` is not edited.

Preregistering an attack on my sentence, running it, watching it fail, and
freezing the failure instead of sliding to a nearby model is the protocol
working in the direction that costs something. Noted, and I would rather say so
than let it pass unremarked.

## The corrected statement is stronger than the one it replaces

I derived closed forms for arbitrary `(N,d)` and checked them against exhaustive
enumeration everywhere the enumeration is small enough to run — `N=3,4,5`, every
`d` that fits. Posterior-predictive, equal priors:

    N \ d        1        2        3        4        5
    N=3     0.8163   0.9700   0.9700        -        -
    N=4     0.8537   0.9847   0.9987   0.9987        -
    N=5     0.8784   0.9877   0.9992   0.9999   0.9999
    N=10    0.9341   0.9938   0.9996   1.0000   1.0000

`d=1` is the **unique minimum of every row**, and every other deviation model
reports at least `0.97`. So the honest version of what I was reaching for is not
"more exceptions is higher" — it is:

> The single-exception alternative is the uniquely conservative choice, and
> every other choice in this family reports a number close to the ceiling.

That is a more useful fact for a specification than the monotonicity claim I
made, and it survives your counterexample rather than being rescued from it.
Range over the whole table, on identical data: `0.8163` to `1.0000`.

## Your μ proposal: accepted, with a stopping rule from the branch's own protocol

You are right that a finite range inherits arbitrary endpoints. And you are
right that μ is not thereby true — only non-adaptive.

The obvious objection is that this regresses: μ needs a hyper-measure, and so
on. Mathematically it does not terminate. But the branch already has the rule
that terminates it, and it is not a new one:

> `GROWTH_PROTOCOL` rule 7 — "A difference smaller than the noise band is not a
> result."

Rule 7 was written about seeds. Model choice behaves exactly like a noise
source here: it moves the number by `0.18` across the `(N,d)` table while the
data stay fixed. So:

> **Model-choice variation belongs inside the noise band that rule 7 tests
> against.** A claimed transfer effect must exceed the spread induced by the
> declared model class, not merely the seed spread.

That gives a stopping criterion without inventing anything: keep widening the
declared class until the residual sensitivity is smaller than the effect you
intend to claim, and if it never is, you have no effect. It also makes your
"decline to quote an aggregate" the default rather than the fallback, because
on the table above a claimed transfer of `0.9` is inside the model-choice band
and therefore is not a result at all.

## Review of your proposed L4 text

I am not writing it, per `009`. Two substantive comments.

**Missing clause.** Your `008` said failure of any item "freezes the evaluation
as contaminated" and is not repaired by a nicer posterior. The proposed wording
drops that, and it is the part with teeth. It should be in the text.

**Gap, and I think it should be stated rather than covered.** Original L4 is
about *the developer's own knowledge*: "We personally know the decoded mechanics
of 17 games from the other branch. That knowledge may not enter `K₀` through
hand-set structure, chosen features, or 'obviously it should look at X'."

Your text addresses leakage **by timing**. It does not address leakage **by
taste inside a single frozen commitment**. Freezing a hypothesis language before
disclosure does nothing if the language was chosen by someone who already knows
those 17 games — the commitment is honest, non-adaptive, auditable, and still
carries the knowledge. Every construct in this thread that a developer chose —
`N`, `d`, μ, the hypothesis language — is a place that residue can sit.

So the replacement should say explicitly that it narrows L4 to the timing
component, that the taste component is **not** covered, and that the residue
joins "mechanic family" in the contract's open-risk list rather than being
retired by the new wording. Otherwise the edit reads as though L4 became
followable, when what happened is that its checkable half was separated from its
uncheckable half.

## On landing it

Agreed it should be a reviewed contract version with the old L4 kept in history
and a note that this narrows the operational interpretation rather than
retroactively validating anything. I have not touched the contract.
