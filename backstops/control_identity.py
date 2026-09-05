"""The re-draw control identity, and what it does to PHASE1_Q1_SPEC.md section 10.

Measurement-tooling analysis, exempt under GROWTH_PROTOCOL's standing
exceptions. Deterministic: no seeds, no sampling, no agent.

Claim: for ANY probability distribution p over the four labels,

    sum over all 24 rule instances R of  p[l+(R)]  =  6

where l+(R) is the label that R maps to the probed effect. Each label is the
answer for exactly 3! = 6 of the 24 bijections, so the sum is 6 * sum(p) = 6,
whatever p is. p may come from an oracle, a random policy, an adversary, or an
agent that has acquired everything -- the sum does not move.

Consequence: any control that re-draws R at probe time and re-scores the SAME
prediction is pinned by the matched score,

    MISMATCHED = (6 - MATCHED) / 23

so it cannot vary independently, and it cannot rise when MATCHED rises.

Run: python3 backstops/control_identity.py
"""

from fractions import Fraction
import random

from q1_backstops import ALL_R, correct_label

LABELS = (0, 1, 2, 3)


def redraw_sum(p):
    """Sum of p at the correct label, over all 24 rule instances."""
    return sum(p[correct_label(R)] for R in ALL_R)


def distributions():
    """Policies spanning the range from below chance to perfect."""
    R0 = ALL_R[7]
    right, wrong = correct_label(R0), (correct_label(R0) + 1) % 4
    rng = random.Random(0)
    yield "uniform (acquired nothing)", {a: Fraction(1, 4) for a in LABELS}
    yield "constant label 0", {a: Fraction(int(a == 0)) for a in LABELS}
    yield "oracle for R0 (perfect)", {a: Fraction(int(a == right)) for a in LABELS}
    yield "anti-oracle for R0", {a: Fraction(int(a == wrong)) for a in LABELS}
    yield "pair narrowed, direction not", {
        a: Fraction(1, 2) if a in (right, wrong) else Fraction(0) for a in LABELS}
    w = [rng.randint(1, 9) for _ in LABELS]
    yield "adversarial random", {a: Fraction(w[a], sum(w)) for a in LABELS}


def main():
    print("Re-draw control identity, over all 24 rule instances\n" + "=" * 70)
    print("  {:<32} {:>10}  {:>12}  {:>12}".format(
        "policy", "sum", "MATCHED", "MISMATCHED"))
    ok = True
    for name, p in distributions():
        total = redraw_sum(p)
        matched = p[correct_label(ALL_R[7])]
        mismatched = (total - matched) / 23
        ok &= (total == 6) and (mismatched == (6 - matched) / 23)
        print("  {:<32} {:>10}  {:>12}  {:>12}".format(
            name, str(total), str(matched), str(mismatched)))

    print("\n  sum is 6 for every policy: {}".format("YES" if ok else "NO"))
    print("\n  Therefore MISMATCHED = (6 - MATCHED)/23 identically:")
    for m in (Fraction(1, 4), Fraction(1, 2), Fraction(3, 4), Fraction(1)):
        x = (6 - m) / 23
        print("     MATCHED {:>4}  ->  MISMATCHED {:>6}  = {:.4f}".format(
            str(m), str(x), float(x)))

    print("\n  PHASE1_Q1_SPEC.md section 10 predeclares:")
    print("     C  MATCHED   must improve above 1/4")
    print("     D  MISMATCHED must NOT improve, stays at 1/4")
    print("     'If both arms rise above chance the effect is non-specific.'")
    print("\n  Both arms rising is impossible: D falls monotonically as C rises.")
    print("  D is an affine image of C, not an independent condition.")
    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
