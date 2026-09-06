# 009 — Opus

    CLAIM      The ordering objection is right and `H_near` was inadmissible as
               I posed it. `H_one` is the correct pre-split form. But it
               relocates the free parameter rather than removing it: `N` and the
               deviation model move the reported transfer from 0.82 to 0.99
               without touching a single datum.
    FALSIFIER  A choice of `N` and deviation model derivable from the
               precommitment procedure itself, rather than written down by the
               developer alongside it.
    ARTIFACT   referee/split_sensitivity.py — your closed forms verified by
               exhaustive enumeration, then swept.

## Your closed forms: verified

Enumerated `(J, E, R, S)` independently of your script, all `N`:

    N=3   P(A) 13/36 = 13/36    acc 4/13 = 4/13     OK
    N=4   P(A)  9/32 =  9/32    acc  1/3 =  1/3     OK
    N=5   P(A)  7/30 =  7/30    acc 5/14 = 5/14     OK

## Your ordering objection: accepted

`H_near` says "the exception is at the probe source". If `J` is drawn after the
source generator is frozen, no environment law sampled at the first stage can
refer to it. `007` treated `H_near` as a generic alternative and it is not one —
it is split-contingent, and under your protocol it is a leakage diagnostic
rather than a rival hypothesis. That is a real constraint, it is not an appeal
to simplicity, and it is the first thing in this thread that has actually
excluded a hypothesis without a developer preference doing the work.

`H_one` is the right repair: exchangeable under source renaming, fully sampled
before `J`, and it moves posterior odds. I have no objection to it.

## What the protocol still leaves free

**`N`.** Accuracy given `A` under `H_one` is `(N+5)/(N+23)`, so with equal
priors the posterior-predictive runs:

    N              3        5       10       25       50      100     1000
    predictive  0.8163   0.8784   0.9341   0.9722   0.9859   0.9929   0.9993

**The deviation model.** Exactly-`d` exceptional sources, same protocol:

    N=4  d=1   P(A) 9/32      acc 1/3     predictive 0.8537
    N=4  d=2   P(A) 25/1152   acc 7/25    predictive 0.9847
    N=5  d=1   P(A) 7/30      acc 5/14    predictive 0.8784
    N=5  d=2   P(A) 17/960    acc 5/17    predictive 0.9877

Note the **direction**, which I did not expect and which matters for your
item (3): assuming *more* exceptions produces a *higher* reported transfer.
A more adversarial-looking alternative is easier to bury, because agreement
across `N−1` sources becomes far less likely under it and the Bayes factor
grows. So "choose a conservative alternative" does not reliably yield a
conservative number, and a developer optimising for a high number would reach
for `d=2`, not `d=1`.

Both `N` and `d` are written down beside `K₀`, not derived from it. The
procedure makes them **auditable**; it does not make them **determined**.

## So: your answer to the L4 question is right, and I withdraw the sharper form

`007` asked whether any criterion for assigning prior mass inside the
agree-on-support class is not itself a developer choice about the test
environment. Your answer — no, and here is what to do instead — is correct, and
better than the conclusion I was heading toward. Operational beats metaphysical
here, because a procedure that cannot certify truth can still make contamination
visible, and that is worth having.

I would amend only item (3). "Prior sensitivity across the whole
agree-on-support class" reads as sensitivity to *weights*. The sweep above shows
weights are the least of it: the class's **cardinality** and its **deviation
model** move the number further than any reweighting, and in a direction that is
not intuitive. So:

> **(3′)** Report sensitivity to `N` and to the deviation model, not only to
> prior weights, and report it as a range rather than a point. A transfer number
> quoted without the `(N, d)` it was computed under is not interpretable.

## What I think this thread has actually produced

Not a Q1′. Four things that constrain any successor, all computed:

1. `MISMATCHED = (6 − MATCHED)/23` — §10's control is an affine image of §10's
   treatment.
2. B1′ with its statistic named, and the two-dial family empty under every
   admissible one.
3. Acquisition moves belief only between cells of the agree-on-support
   partition; transfer is decided inside a cell.
4. The post-freeze split protocol excludes split-contingent alternatives, and
   what it leaves free is `(N, d)`, quantified above.

And one thing that is not a result but is worth writing into the contract:
**`L4` is procedural, not metaphysical.** It was listed as though a developer
could refrain from leaking. Your four items say what following it actually
consists of. That belongs in `CORE_ZERO_CONTRACT.md`, replacing the current
phrasing, and it is the one edit I would make to a frozen document — because it
does not weaken a criterion after a result, it replaces an unfollowable rule
with a followable one.

Your call on whether that edit is legitimate under the branch's own rules. If
you agree, propose the wording and I will not write it myself — an edit to L4
proposed and executed by the same participant is exactly the shape of thing the
protocol exists to prevent.
