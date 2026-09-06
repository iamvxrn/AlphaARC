"""Fine rational-grid scope audit of dialogue/013's iid-q minima."""

from fractions import Fraction
from math import comb


Q = Fraction(1, 24)


def score(source_count, q):
    agree = numerator = Fraction(0)
    for observed_deviants in range(source_count):
        weight = (comb(source_count - 1, observed_deviants) * q ** observed_deviants
                  * (1 - q) ** (source_count - 1 - observed_deviants))
        if observed_deviants < source_count - 1:
            p_agree = Q ** observed_deviants
            accuracy = 1 - Fraction(3, 4) * q
        else:
            p_agree = Q ** (source_count - 2)
            accuracy = Fraction(1, 4)
        agree += weight * p_agree
        numerator += weight * p_agree * accuracy
    conditional_accuracy = numerator / agree
    return (1 + agree * conditional_accuracy) / (1 + agree)


def main():
    coarse = {3: Fraction(11, 25), 4: Fraction(8, 25), 5: Fraction(13, 50)}
    for n, reported in coarse.items():
        candidate = min((score(n, Fraction(i, 10_000)), Fraction(i, 10_000))
                        for i in range(1, 10_000))
        reported_score = score(n, reported)
        print("N={}: reported q={} score={}; fine q={} score={}".format(
            n, reported, reported_score, candidate[1], candidate[0]
        ))
        assert candidate[0] < reported_score, (n, candidate, reported_score)
    print("every reported hundredth-grid minimizer has a lower 1/10000-grid neighbour")


if __name__ == "__main__":
    main()
