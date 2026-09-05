# B1′ — result

Instrument specified and committed at `5a568bf`, before the run.
Code: `backstops/b1_prime.py`. Reproduce: `python3 backstops/b1_prime.py`
(from `backstops/`). Exhaustive and deterministic over all 24 rule instances
and all 16 feature subsets; no seeds, noise band zero.

    B1  (majority vote, frozen at a24e111)  0.5000
    B1' (max over the locality class)       0.5000

## Predeclared checks

| # | prediction | observed | |
|---|---|---|---|
| 1 | `B1′ ≥ 0.5000` by dominance | `0.5000` | OK |
| 3 | `{d1,t1}` = `0.5000`, known minimal case | `0.5000` | OK |
| 4 | every subset containing `t2` = `0.2500` | 8/8 | OK |

## The open question, answered

> **2. Is `B1′ > 0.5000` on some subset?**

**No.** `0.5000` is the maximum of the class. Majority vote attained it.

The frozen `0.50` in `B1_FAILURE_FREEZE.md` was therefore the right number, and
is now the right number *for an established reason*: it is the best any
`K`-indexed lookup learner can do, not the best one particular fitting rule
could do. The instrument freeze's stronger-claims caveat — that `0.5000` was not
shown to be the best achievable for the class — is discharged.

Q1 remains **REJECTED** at `a4877fb`. Nothing here changes that, and the
rejection rested on `> 0.25`, not on the exact value.

## Unpredicted result: four subsets reach `0.50`, by two different routes

The freeze recorded one shortcut key. There are four: `{d2}`, `{d1,d2}`,
`{d2,t1}`, `{d1,t1}`. And they split into two structurally distinct mechanisms.

**Identification by presence.** At `{d1,t1}`, key `(0,0)`, the local vote vector
is `(0,0,2,2)` — the two dial-2 labels carry all the votes, because dial 1 is
already satisfied at that key and the only remaining work is on dial 2. The
answer is one of the two *voted* labels. This is the case the freeze documented.

**Identification by absence.** At `{d2}`, key `d2=0`, the local vote vector is
`(0,0,8,8)` — and the answer is one of the two labels with **zero** votes. At
`d2 = t2 = 0` dial 2 is already matched, so no dial-2 label ever makes progress
there; only dial-1 labels are ever voted for. A learner at that key has still
identified the dial-2 pair exactly — as the complement of what it voted for.

Both routes land on the same wall: the local data pins the *pair* and never the
*direction*. That is why the answer is `0.50` at every reachable key rather
than `0.50` at one key and something else elsewhere.

## Correction to a claim in `B1_FAILURE_FREEZE.md`

That document records, under "Corroborating results":

> The SUPPRESSED prediction of §9 is confirmed exhaustively. Subsets `{d2}`,
> `{d1,d2}`, `{d2,t1}` score exactly `0.0000` — below chance, as §9 predicted
> for reward-following. This was a predeclared qualitative call and it came out
> right.

Under B1′ those three subsets score `0.5000`, not `0.0000`. The freeze is not
edited — freezes are not amended — so the correction is recorded here:

- The `0.0000` is a property of **majority vote**, which cannot express "emit a
  label I never voted for". It is not a property of what is knowable at those
  keys.
- The word **"exhaustively"** in that bullet does not hold. The call was
  confirmed for one fitting rule over an under-specified class.
- §9's SUPPRESSED rung is **not** invalidated. It describes what a *reward-
  following agent* does, and a reward-following agent does avoid those labels.
  What is invalidated is using a lookup-table sweep as exhaustive confirmation
  of it.
- Direction of the error: the freeze under-stated Q1's exposure at three of the
  four shortcut keys, and stated it as confirmation of a prediction.

## Sharpened constraint for Q1′

`B1_PRIME_INSTRUMENT.md` closed with:

> passes B1′ only if, for every `K`, the local fitting data at `π_K(S*)` leaves
> the probed label unidentified — either because the key is unreachable, or
> because the data at that key is symmetric across all four labels.

The identification-by-absence result sharpens the second clause, and the sharper
form is the one Q1′ must be designed against:

> **Balancing the votes is not sufficient.** The zero-vote entries carry as much
> information as the non-zero ones. The requirement is that the local vote
> vector at `π_K(S*)` be **invariant under every label permutation that changes
> the correct probe answer** — not merely that it be numerically even.

Concretely, a design in which some feature key makes the probed dial "the one
with nothing to do here" fails for exactly the same reason as one in which it
makes the probed dial "the only thing to do here". Satisfying the other dials to
isolate the probed one fails both ways round.

## Status

    B1' INSTRUMENT EXECUTED AS SPECIFIED -- all self-checks pass
    B1' = 0.5000, the maximum of the class
    Q1 UNCHANGED AND STILL REJECTED
    one claim in B1_FAILURE_FREEZE.md corrected, that freeze left unedited

No successor environment, agent, or `U` is introduced here.
