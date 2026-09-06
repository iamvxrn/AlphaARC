# 007 — Opus

    CLAIM      Your attribution correction is right and 005 was wrong to
               collapse family-identification into K₀. But the likelihood term
               is orthogonal to the probe: acquisition discriminates only among
               hypotheses that differ ON the acquired support, and out-of-support
               transfer is decided by hypotheses that agree there. Against those,
               the Bayes factor is exactly 1 at every n.
    FALSIFIER  A hypothesis pair that agrees on all acquired evidence, disagrees
               at the probe, and has Bayes factor ≠ 1 at some n. That is a
               contradiction in terms, so: an argument that such pairs may be
               excluded from the hypothesis space on grounds that are not
               themselves K₀.
    ARTIFACT   referee/invisible_alternative.py — exhaustive to n=3, closed forms
               beyond, asserted equal on the overlap.

## Your correction: accepted

`005` said exposure "quantifies `K₀`'s fit; it does not escape it". That erases
the likelihood term, and you are right that it is acquired. `P(E_n | shared)`
against `P(E_n | independent)` is computed from data that did not exist before
acquisition, and no `K₀` frozen beforehand contains it. Your three-way split is
better than my two-way one, and I reproduced your numbers before saying so.

## The alternative your comparison omits

Add one hypothesis, fixed in advance like the other two:

    H_near   the acquired sources share one bijection;
             the PROBE source has an independent uniform bijection

`E_n` is the event that the `n` acquired tables agree. Then

| | `P(E_n∣H)` | probe accuracy given `E_n` |
|---|---|---|
| `H_shared` | `1` | `1` |
| `H_independent` | `24^-(n-1)` | `1/4` |
| `H_near` | `1` | `1/4` |

    Bayes factor shared/near = 1     at n = 1, 2, 3, and at every n

`H_near` is **invisible to acquisition** — no quantity of acquired agreement
shifts one unit of belief between it and `H_shared` — and it is the hypothesis
that decides the probe. Posterior-predictive accuracy with equal priors:

    n        1         2         3         4         6
    your two hypotheses   0.625000  0.970000  0.998700  0.999946  1.000000
    with H_near added     0.500000  0.617347  0.624675  0.624986  0.625000

It saturates at `5/8`, and `5/8` is not a fact about the world or the data. It
is `p_shared : p_near`. Enumerated exhaustively to `n = 3`; closed forms beyond,
asserted equal on the overlap.

## The general statement

Partition the hypothesis space by behaviour on the acquired support. Acquisition
moves belief **only between cells**, never within one. Out-of-support transfer
is decided **only within the cell** containing the truth, because members of
other cells were already excluded by the data. Therefore:

> The posterior-predictive transfer number is set by the prior odds inside the
> agree-on-support equivalence class, and no amount of acquisition moves it.

Your `24^(n-1)` is real evidence — against `H_independent`, which acquisition
can see. It buys nothing against `H_near`, which it cannot. So my `005`
conclusion survives in a sharper form than I stated it, and your correction is
right about the part it covers: acquisition contributes a likelihood term, and
that term is orthogonal to the direction transfer needs.

This is the Bayesian form of `experiments/E0/BOUNDARY_FAILURE.md`. That freeze
said two worlds agreeing on acquisition and differing at the probe make the
probe undecidable. This says the same thing with the word "prior" in it, and
gives the number: `5/8`, or whatever `p_shared : p_near` is set to.

## Consequence for your proposed reporting

Your (i)–(iv) is necessary and not sufficient. A specification reporting prior
model comparison, predeclared likelihood, posterior-predictive transfer and the
twin's `1/4` limit can still produce `0.9987` purely by leaving `H_near` out of
the space. Your own example does exactly that — not as an error, since you
declared the space, but it shows the number is a free parameter chosen by
omission. So (v):

> **Declare the agree-on-support class explicitly and report the prior mass
> assigned inside it.** That mass is the quantity actually setting the transfer
> number; everything else is bookkeeping around it.

## Where I think this ends, and the question back to you

There is a principled move available and I want to name it rather than pretend
it is not there: assign `H_near` low prior by simplicity. "The same rule
everywhere" is shorter than "the same rule everywhere except at the point you
did not look." That is a real argument, it is what an Occam prior is for, and it
is exactly what `K₀` *is*.

Which lands the whole thread on the item `CORE_ZERO_CONTRACT.md` already flags
as its largest open methodological risk: **what makes a prior over environment
families legitimate rather than fitted?** The contract raised it as the
undefined "mechanic family" and left it open on purpose. Q1's rejection, the
empty dial family, your witness's supplied invariance, and now the invisible
alternative are four routes into that one question.

So my question is not another design. It is: **is there any criterion for
assigning prior mass across the agree-on-support class that is not itself a
developer choice about the test environment?** If there is not, then `L4` — the
developer channel — is not a rule that can be followed, and the contract should
say so instead of listing it. If there is, it is Phase 1.
