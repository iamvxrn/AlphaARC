"""Is d=1 the conservative bottom, or only the bottom of the exactly-d family?

Artifact for dialogue/013-opus.md. Compares two admissible alternative families
under the post-freeze split protocol of dialogue/008-astra.md. Both are
exchangeable under source renaming and fully sampled before the held-out source
J is drawn, so both pass the same admissibility test H_one passes.

    exactly-d   exactly d exceptional sources, chosen uniformly
    iid-q       each source deviates independently with probability q

Exact rational arithmetic, no sampling, no agent.

Run: python3 referee/conservative_bottom.py
"""

from fractions import Fraction
from math import comb

Q = Fraction(1, 24)


def exact_d(N, d):
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


def iid_q(N, q):
    """Deviation rate q. m counts deviants among the N-1 observed sources.

    With at least one non-deviant observed, agreement pins every observed
    deviant onto the common rule and the learned rule is that common rule. With
    every observed source deviant, they need only agree with each other, and
    what is learned then says nothing about the held-out source.
    """
    pA = num = Fraction(0)
    for m in range(N):
        w = comb(N - 1, m) * q ** m * (1 - q) ** (N - 1 - m)
        if m < N - 1:
            pAm, acc = Q ** m, (1 - q) * 1 + q * Fraction(1, 4)
        else:
            pAm, acc = Q ** (N - 2), Fraction(1, 4)
        pA += w * pAm
        num += w * pAm * acc
    return pA, num / pA


def predictive(pA, acc, prior=Fraction(1, 2)):
    return (prior + (1 - prior) * pA * acc) / (prior + (1 - prior) * pA)


def main():
    print("Conservative bottom across two admissible families\n" + "=" * 68)
    for N in (3, 4, 5):
        d1 = predictive(*exact_d(N, 1))
        scan = [(predictive(*iid_q(N, Fraction(i, 100))), Fraction(i, 100))
                for i in range(1, 100)]
        lo, q_lo = min(scan)
        hi_end = max(scan[0][0], scan[-1][0])
        print("\n  N={}".format(N))
        print("    exactly-d, d=1                        {:.4f}".format(float(d1)))
        print("    iid-q, minimum at q={:<7}           {:.4f}".format(str(q_lo), float(lo)))
        print("    iid-q, worst endpoint of the scan     {:.4f}".format(float(hi_end)))
        print("    conservative bottom: {}".format(
            "iid-q, BELOW d=1" if lo < d1 else "exactly-d at d=1"))

    print("\n" + "=" * 68)
    print("d=1 is the bottom in both families at every N tried. Two families is")
    print("not all families, so this is evidence and not a theorem: the")
    print("admissible ambiguity set still has to be declared.")
    print()
    print("Note the shape of iid-q: its minimum sits at an interior q, and both")
    print("endpoints of the scan are far above it. A sensitivity report that")
    print("scanned only the endpoints of a declared family would have missed")
    print("that family's own most conservative member.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
