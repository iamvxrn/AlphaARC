"""Does the iid-q family ever reach exactly-d=1? Decided exactly, not on a grid.

Artifact for dialogue/015-opus.md, answering the grid-scope correction in
dialogue/014-astra.md.

dialogue/013 reported minima of the iid-q family found by scanning q = i/100.
Astra correctly objected that a grid minimum is not a continuous minimum. The
claim that actually mattered was not "where is the minimum" but "can this family
reach the exactly-d=1 value", and that is a decision problem with an exact
answer.

Write the posterior-predictive of the iid-q family as (1 + A(q)) / (1 + B(q))
with A, B polynomials in q over the rationals, and let T be the exactly-d=1
value. Since 1 + B(q) > 0 on [0,1], the sign of predictive(q) - T is the sign of

    C(q) = (1 + A(q)) - T * (1 + B(q))

A Sturm sequence counts C's distinct real roots in (0,1] exactly. No roots plus
positive endpoints proves predictive_iid(q) > T for every real q in [0,1].

The root counter is validated against polynomials with known roots before use.

Run: python3 referee/iid_family_bound.py
"""

from fractions import Fraction as F
from math import comb

Q = F(1, 24)


# ------------------------------------------------------------ polynomial tools

def trim(p):
    while len(p) > 1 and p[-1] == 0:
        p = p[:-1]
    return p


def padd(a, b):
    out = [F(0)] * max(len(a), len(b))
    for i, c in enumerate(a):
        out[i] += c
    for i, c in enumerate(b):
        out[i] += c
    return trim(out)


def pmul(a, b):
    out = [F(0)] * (len(a) + len(b) - 1)
    for i, x in enumerate(a):
        for j, y in enumerate(b):
            out[i + j] += x * y
    return trim(out)


def prem(a, b):
    """Remainder of a divided by b."""
    r, b = trim(a[:]), trim(b)
    while True:
        r = trim(r)
        if r == [F(0)] or len(r) < len(b):
            return r
        d, c = len(r) - len(b), r[-1] / b[-1]
        for i, y in enumerate(b):
            r[i + d] -= c * y


def deriv(p):
    d = [p[i] * i for i in range(1, len(p))]
    return trim(d) if d else [F(0)]


def ev(p, x):
    s = F(0)
    for c in reversed(p):
        s = s * x + c
    return s


def sturm_count(p, lo, hi):
    """Distinct real roots of p in (lo, hi]."""
    seq = [trim(p), deriv(p)]
    while len(trim(seq[-1])) > 1:
        r = prem(seq[-2], seq[-1])
        if r == [F(0)]:
            break
        seq.append([-c for c in r])

    def changes(x):
        vals = [v for v in (ev(s, x) for s in seq) if v != 0]
        return sum(1 for i in range(len(vals) - 1) if (vals[i] > 0) != (vals[i + 1] > 0))

    return changes(lo) - changes(hi)


def validate():
    cases = [
        ("q - 1/2", [F(-1, 2), F(1)], 1),
        ("(q - 1/4)(q - 3/4)", [F(3, 16), F(-1), F(1)], 2),
        ("(q - 2)(q - 3)", [F(6), F(-5), F(1)], 0),
        ("(q - 1/3)^2", [F(1, 9), F(-2, 3), F(1)], 1),
        ("q(q - 1/2)(q + 1/2)", [F(0), F(-1, 4), F(0), F(1)], 1),
        ("constant 5", [F(5)], 0),
    ]
    print("  root counter, against known polynomials on (0,1]")
    ok = True
    for name, p, want in cases:
        got = sturm_count(p, F(0), F(1))
        ok &= got == want
        print("    {:<24} want {}  got {}  {}".format(
            name, want, got, "OK" if got == want else "FAIL"))
    return ok


# ------------------------------------------------------------------- the model

def iid_polys(N):
    """A(q) = P(A)*accuracy and B(q) = P(A), as polynomials in q."""
    q, omq = [F(0), F(1)], [F(1), F(-1)]

    def power(p, k):
        r = [F(1)]
        for _ in range(k):
            r = pmul(r, p)
        return r

    A = B = [F(0)]
    for m in range(N):
        w = pmul([F(comb(N - 1, m))], pmul(power(q, m), power(omq, N - 1 - m)))
        if m < N - 1:
            pAm, acc = [Q ** m], padd(omq, [F(0), F(1, 4)])
        else:
            pAm, acc = [Q ** (N - 2)], [F(1, 4)]
        B = padd(B, pmul(w, pAm))
        A = padd(A, pmul(pmul(w, pAm), acc))
    return A, B


def exact_d1(N):
    p_in, p_out = F(1, N), F(N - 1, N)
    pA_out, acc_out = (Q, F(1)) if N >= 3 else (F(1), F(1, 4))
    pA = p_in * F(1) + p_out * pA_out
    acc = (p_in * F(1) * F(1, 4) + p_out * pA_out * acc_out) / pA
    return (F(1, 2) + F(1, 2) * pA * acc) / (F(1, 2) + F(1, 2) * pA)


def main():
    print("iid-q family bound, decided exactly\n" + "=" * 70)
    if not validate():
        print("\n  counter failed validation; no conclusion drawn")
        return 1

    print("\n  C(q) = (1 + A(q)) - T*(1 + B(q)),  T = exactly-d=1 value")
    print("  N     T          C(0)       C(1)      roots in (0,1]   verdict")
    ok = True
    for N in (3, 4, 5, 6, 8):
        A, B = iid_polys(N)
        T = exact_d1(N)
        C = padd(padd([F(1)], A), [-T * c for c in padd([F(1)], B)])
        roots = sturm_count(C, F(0), F(1))
        clean = roots == 0 and ev(C, F(0)) > 0 and ev(C, F(1)) > 0
        ok &= clean
        print("  {:<5} {:<10.6f} {:<10.6f} {:<9.6f} {:^16} {}".format(
            N, float(T), float(ev(C, F(0))), float(ev(C, F(1))), roots,
            "iid-q > d=1 everywhere" if clean else "INCONCLUSIVE"))

    print("\n" + "=" * 70)
    if ok:
        print("Proved for N in {3,4,5,6,8}: no q in [0,1] brings the iid-q family")
        print("down to the exactly-d=1 value. The grid question is moot for this")
        print("claim -- it was only needed to locate a minimum, and locating one")
        print("was never what the claim required.")
    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
