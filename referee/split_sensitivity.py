"""Sensitivity of the post-freeze split protocol to the developer's remaining choices.

Artifact for dialogue/009-opus.md. Verifies the closed forms of dialogue/008-astra.md
by exhaustive enumeration, then measures what the protocol leaves free.

Protocol (Astra, 008): freeze K0 and the source generator; sample the held-out
source J after that; observe the other N-1 complete tables; condition on their
agreement A. H_one samples exactly one exceptional source uniformly among N.

Exact, exhaustive, no sampling, no agent.

Run: python3 referee/split_sensitivity.py
"""

import itertools
from fractions import Fraction

R24 = list(itertools.permutations(range(4)))


def audit(N, d):
    """Exhaustive over (J, exceptional set, common rule, exceptional rules).

    Returns P(A) and the history policy's probe accuracy given A, where the
    policy learns the common rule from the agreeing observed tables and answers
    the label whose output is the target.
    """
    ok = tot = 0
    hits = Fraction(0)
    for J in range(N):
        for E in itertools.combinations(range(N), d):
            for R in R24:
                for exceptional in itertools.product(R24, repeat=d):
                    tot += 1
                    rule = {i: R for i in range(N)}
                    for idx, e in enumerate(E):
                        rule[e] = exceptional[idx]
                    observed = [rule[i] for i in range(N) if i != J]
                    if len(set(observed)) != 1:
                        continue
                    ok += 1
                    learned = observed[0]
                    hits += 1 if rule[J][learned.index(0)] == 0 else 0
    return Fraction(ok, tot), Fraction(hits, ok)


def predictive(pa, acc, prior=Fraction(1, 2)):
    """Posterior-predictive probe accuracy over {H_shared, the alternative}."""
    num = prior * 1 * 1 + (1 - prior) * pa * acc
    den = prior * 1 + (1 - prior) * pa
    return num / den


def main():
    print("Split-protocol sensitivity\n" + "=" * 66)

    print("\n1. Astra's H_one closed forms, checked by exhaustive enumeration")
    print("   N   P(A) enum   closed    acc enum  closed   verdict")
    ok = True
    for N in (3, 4, 5):
        pa, acc = audit(N, 1)
        pa_c, acc_c = Fraction(N + 23, 24 * N), Fraction(N + 5, N + 23)
        match = (pa == pa_c and acc == acc_c)
        ok &= match
        print("   {}   {:>9} {:>8}   {:>7} {:>7}   {}".format(
            N, str(pa), str(pa_c), str(acc), str(acc_c),
            "OK" if match else "MISMATCH"))
    if not ok:
        return 1

    print("\n2. First free choice: N, the number of sources in the declared family")
    print("     N      acc|A,H_one    posterior-predictive")
    for N in (3, 5, 10, 25, 50, 100, 1000):
        pa, acc = Fraction(N + 23, 24 * N), Fraction(N + 5, N + 23)
        print("   {:>5}   {:>12}    {:.6f}".format(
            N, "{:.6f}".format(float(acc)), float(predictive(pa, acc))))

    print("\n3. Second free choice: the deviation model, d exceptional sources")
    print("   Note the direction -- more assumed exceptions raises the reported")
    print("   transfer, because agreement across N-1 sources becomes far less")
    print("   likely under the alternative and the Bayes factor buries it.")
    print("     N   d        P(A)      acc|A    posterior-predictive")
    for N in (4, 5):
        for d in (1, 2):
            pa, acc = audit(N, d)
            print("   {:>4} {:>3}  {:>10}  {:>9}    {:.6f}".format(
                N, d, str(pa), str(acc), float(predictive(pa, acc))))

    print("\n" + "=" * 66)
    print("The protocol constrains the FORM of admissible alternatives.")
    print("N and the deviation model remain developer choices, and they move")
    print("the reported number from 0.82 to 0.99 without touching the data.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
