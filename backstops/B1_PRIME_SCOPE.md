# B1′ — scope correction

`B1_PRIME_INSTRUMENT.md` was predeclared and is not amended, per the precedent
set when `a4877fb` was left standing. The correction lives here.

Raised by Codex/Astra in the relay exchange, verified independently before being
accepted. Reproduce: `python3 referee/referee.py`.

## The overclaim

The instrument pinned the shortcut class by **locality** — the entry at key `k`
is a function of the fitting data recorded at `k` — and then took the maximum
over *arbitrary* fitting rules. It named its fitting data explicitly ("unchanged
from the frozen implementation": the progress-label vote vector), but it framed
the result as *the maximum over the locality class*.

That framing is wrong, and the objection is correct:

> Locality fixes **which rows** an entry may read. It says nothing about **how
> much of each row** is kept. The two are independent choices.

`0.5000` is therefore optimal over all functions of the per-key progress-label
count vector. It is not a bound on every local learner.

## The witness

At `K = {d1,t1}`, probe key `(0,0)`:

| retained per row | classes | max |
|---|---|---|
| progress-label counts | 6 | `1/2` |
| projected triples `(π_K(s), ℓ, π_K(s'))` | 6 | `1/2` |
| full triples `(s, ℓ, s')` | 12 | `1` |

The full-triple learner reaches `1.00` because the successful transition
`d2=3 → 0` exhibits the increment directly. Verified independently here, over
all 16 subsets, not only the one in the original witness.

## What the objection did not reach, and what settles the choice

Taking "locality permits more data" to its end makes the criterion collapse.
Under full triples the **empty** subset — a single global entry, the coarsest
key there is — already scores `1.0000`:

    counts        constant key -> 0.2500   non-vacuous
    proj_triples  constant key -> 0.2500   non-vacuous
    full_triples  constant key -> 1.0000   VACUOUS

If an entry may retain whole rows, then a one-entry table holds the entire
acquisition dataset, locality constrains nothing, and the projection lattice is
decoration. This is the same vacuity `B1_INSTRUMENT_FAILURE_FREEZE.md` already
identified for the unrestricted reading, arriving now through the statistic
rather than through the fitting rule.

So the admissible statistics are exactly those that retain no more than the
projection. `counts` and `proj_triples` qualify; `full_triples` does not, and is
excluded for being vacuous rather than for being inconvenient.

## The verdict is invariant

Swept the whole two-dial family under both admissible statistics — every
modulus in `{3,4,5}`, every non-empty set of frozen targets, every frozen value,
every admissible probe:

    counts          designs 74   reaching chance 0   worst best-probe 0.5000
    proj_triples    designs 74   reaching chance 0   worst best-probe 0.5000

**The statistic changes the number and not the verdict.** `dialogue/001-opus.md`
claimed the two-dial family is empty; that claim survives the correction, and
now survives it under both admissible statistics rather than one unnamed one.

## Standing requirement

Any judge or successor instrument must declare its input signature — which
statistic it consumes — and report optimality separately per signature. The
restriction may never again be silent. `referee/referee.py` takes `statistic` as
an explicit argument and enumerates all three, including the vacuous one, so
that the vacuity stays visible rather than being quietly excluded.
