"""The invisible alternative: acquisition cannot see the hypothesis that decides the probe.

Artifact for dialogue/007-opus.md. Extends the model comparison in
referee/astra_005_evidence.py with a third hypothesis. Exact, exhaustive,
no sampling, no agent.

    H_shared       one bijection at every source, probe source included
    H_independent  an independent uniform bijection at every source
    H_near         the acquired sources share one bijection; the PROBE source
                   is free

E_n is the event that the n acquired action-output tables agree.

P(E_n | shared) = P(E_n | near) = 1 for every n, so the Bayes factor between
them is exactly 1 no matter how much is acquired -- while they disagree about
the probe answer. Acquisition discriminates only among hypotheses that differ
ON the acquired support; out-of-support transfer is decided by hypotheses that
agree there.

Run: python3 referee/invisible_alternative.py
"""

import itertools
from fractions import Fraction

R24 = list(itertools.permutations(range(4)))


def enumerate_hypothesis(H, n):
    """Complete environment list: (acquired source rules, probe source rule)."""
    if H == "shared":
        return [((R,) * n, R) for R in R24]
    if H == "near":
        return [((R,) * n, S) for R in R24 for S in R24]
    return [(acq, probe)
            for acq in itertools.product(R24, repeat=n) for probe in R24]


def measure(H, n):
    envs = enumerate_hypothesis(H, n)
    evidence = [e for e in envs if len(set(e[0])) == 1]
    # History policy: answer the label whose observed output is the target 0.
    hits = sum(1 for acq, probe in evidence if probe[acq[0].index(0)] == 0)
    return (Fraction(len(evidence), len(envs)),
            Fraction(hits, len(evidence)), len(envs), len(evidence))


# Closed forms, cross-checked against measure() for n <= 3 in main().
CLOSED = {
    "shared":      lambda n: (Fraction(1), Fraction(1)),
    "near":        lambda n: (Fraction(1), Fraction(1, 4)),
    "independent": lambda n: (Fraction(1, 24) ** (n - 1), Fraction(1, 4)),
}


def predictive(n, priors):
    """Posterior-predictive probe accuracy under the given priors."""
    num = den = Fraction(0)
    for H, p in priors.items():
        pe, acc = CLOSED[H](n)
        num += p * pe * acc
        den += p * pe
    return num / den


def main():
    print("Invisible alternative -- exact enumeration\n" + "=" * 70)
    print("  n   hypothesis     P(E_n|H)    acc|E_n,H    environments   E_n cases")
    for n in (1, 2, 3):
        for H in ("shared", "independent", "near"):
            pe, acc, tot, k = measure(H, n)
            print("  {}   {:<12} {:>10} {:>12} {:>15} {:>11}".format(
                n, H, str(pe), str(acc), tot, k))
        print()

    for n in (1, 2, 3):
        for H in ("shared", "independent", "near"):
            pe, acc, _, _ = measure(H, n)
            assert (pe, acc) == CLOSED[H](n), (H, n)
    print("  closed forms agree with brute-force enumeration for n <= 3\n")

    print("Bayes factors against H_shared")
    for n in (1, 2, 3):
        bi = measure("shared", n)[0] / measure("independent", n)[0]
        bn = measure("shared", n)[0] / measure("near", n)[0]
        print("  n={}   shared/independent = {:<6}   shared/near = {}".format(
            n, str(bi), str(bn)))

    two = {"shared": Fraction(1, 2), "independent": Fraction(1, 2)}
    three = {h: Fraction(1, 3) for h in ("shared", "independent", "near")}
    print("\nPosterior-predictive probe accuracy, equal priors")
    print("  n   two hypotheses        three hypotheses")
    for n in (1, 2, 3, 4, 6):
        print("  {:>2}   {:<21} {:.6f}".format(
            n, "{:.6f}".format(float(predictive(n, two))),
            float(predictive(n, three))))
    print("\n  two   -> 1     acquisition appears to buy full transfer")
    print("  three -> 5/8   capped by the prior ratio p_shared : p_near,")
    print("                 and the cap does not move with n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
