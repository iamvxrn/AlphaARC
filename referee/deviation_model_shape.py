"""The full (N, d) shape of the post-freeze split protocol's transfer number.

Artifact for dialogue/011-opus.md, replacing the loose directional claim in 009.

Protocol per dialogue/008-astra.md: freeze K0 and the source generator; draw the
held-out source J afterwards; observe the other N-1 complete tables; condition
on their agreement A. The alternative has exactly d exceptional sources drawn
uniformly, each with an independent uniform rule.

Closed forms are cross-checked against exhaustive enumeration wherever the
enumeration is small enough to run.

Run: python3 referee/deviation_model_shape.py
"""

import itertools
from fractions import Fraction

R24 = list(itertools.permutations(range(4)))
Q = Fraction(1, 24)


def brute(N, d):
    ok = tot = 0
    hits = Fraction(0)
    for J in range(N):
        for E in itertools.combinations(range(N), d):
            for R in R24:
                for S in itertools.product(R24, repeat=d):
                    tot += 1
                    rule = {i: R for i in range(N)}
                    for i, e in enumerate(E):
                        rule[e] = S[i]
                    observed = [rule[i] for i in range(N) if i != J]
                    if len(set(observed)) != 1:
                        continue
                    ok += 1
                    hits += 1 if rule[J][observed[0].index(0)] == 0 else 0
    return Fraction(ok, tot), Fraction(hits, ok)


def closed(N, d):
    """P(A) and probe accuracy given A.

    Split on whether the held-out source is one of the exceptions. If any
    normal source is observed, agreement forces every observed exception onto
    the common rule; if every observed source is exceptional, they need only
    agree with each other, and the learned rule then carries no information
    about the held-out one.
    """
    p_in, p_out = Fraction(d, N), Fraction(N - d, N)
    pA_in, acc_in = (Q ** (d - 1) if d <= N - 1 else Q ** (N - 2)), Fraction(1, 4)
    if d <= N - 2:
        pA_out, acc_out = Q ** d, Fraction(1)
    elif d == N - 1:
        pA_out, acc_out = Q ** (d - 1), Fraction(1, 4)
    else:
        pA_out, acc_out = Fraction(0), Fraction(0)
    pA = p_in * pA_in + p_out * pA_out
    return pA, (p_in * pA_in * acc_in + p_out * pA_out * acc_out) / pA


def predictive(pA, acc, prior=Fraction(1, 2)):
    return (prior + (1 - prior) * pA * acc) / (prior + (1 - prior) * pA)


def main():
    print("Deviation-model shape\n" + "=" * 72)
    print("\n1. Closed forms against exhaustive enumeration")
    ok = True
    for N in (3, 4, 5):
        for d in range(1, N + 1):
            if N == 5 and d >= 3:
                continue  # enumeration too large to be worth running
            match = brute(N, d) == closed(N, d)
            ok &= match
            print("   N={} d={}   {}".format(N, d, "OK" if match else "MISMATCH"))
    if not ok:
        return 1

    print("\n2. Posterior-predictive transfer, equal priors on {shared, alternative}")
    print("   N \\ d   " + "".join("{:>9}".format(d) for d in range(1, 8)))
    for N in (3, 4, 5, 6, 8, 10):
        row = ""
        for d in range(1, 8):
            row += "{:>9}".format(
                "-" if d > N else "{:.4f}".format(float(predictive(*closed(N, d)))))
        print("   N={:<7}".format(N) + row)

    print("\n3. What the table says")
    print("   d=1 is the unique minimum of every row. Every other deviation")
    print("   model reports at least 0.97, and most report above 0.999.")
    print("   There is no strict monotonicity in d -- N=3 ties d=2 with d=3,")
    print("   which is the endpoint result frozen in")
    print("   backstops/SPLIT_SENSITIVITY_ENDPOINT_FREEZE.md.")
    print("   Range over the whole table: 0.8163 to 1.0000, on identical data.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
